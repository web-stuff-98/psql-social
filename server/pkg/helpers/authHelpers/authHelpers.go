package authHelpers

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func createCookie(token string, expiry time.Time) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     "session_token",
		Value:    token,
		Expires:  expiry,
		MaxAge:   120,
		Secure:   os.Getenv("ENVIRONMENT") == "PRODUCTION",
		HTTPOnly: true,
		SameSite: "Strict",
		Path:     "/",
	}
}

func GetClearedCookie() *fiber.Cookie {
	return &fiber.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Secure:   os.Getenv("ENVIRONMENT") == "PRODUCTION",
		HTTPOnly: true,
		SameSite: "Strict",
		Path:     "/",
	}
}

// copied the regex from some stack overflow post. Same validation as with client.
func PasswordValidates(pass string) bool {
	count := 0
	if 8 <= len(pass) && len(pass) <= 72 {
		if matched, _ := regexp.MatchString(".*\\d.*", pass); matched {
			count++
		}
		if matched, _ := regexp.MatchString(".*[a-z].*", pass); matched {
			count++
		}
		if matched, _ := regexp.MatchString(".*[A-Z].*", pass); matched {
			count++
		}
		if matched, _ := regexp.MatchString(".*[*.!@#$%^&(){}\\[\\]:;<>,.?/~`_+-=|\\\\].*", pass); matched {
			count++
		}
	}
	return count >= 3
}

// Creates the session ID on redis and encodes it as a JWT into a cookie
func Authorize(redisClient *redis.Client, ctx context.Context, uid string) (*fiber.Cookie, error) {
	sid := uuid.New()
	sessionDuration := time.Minute * 2

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Issuer:    sid.String(),
		ExpiresAt: time.Now().Add(sessionDuration).Unix(),
	})
	token, err := claims.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		log.Fatalln("Error in Authorize helper function generating token:", err)
	}
	cookie := createCookie(token, time.Now().Add(sessionDuration))

	cmd := redisClient.Set(ctx, sid.String(), uid, sessionDuration)
	if cmd.Err() != nil {
		log.Fatalln("Redis error in Authorize helper function:", cmd.Err())
	}

	return cookie, nil
}

// Decrypt the JWT stored inside the cookie, queries the db for the user ID and returns the user ID and session ID
func GetUidAndSid(redisClient *redis.Client, ctx *fiber.Ctx, rctx context.Context, db *pgxpool.Pool) (uid string, sid string, err error) {
	cookie := string(ctx.Request().Header.Cookie("session_token"))
	if cookie == "" {
		return "", "", fmt.Errorf("No cookie")
	}

	token, err := jwt.ParseWithClaims(cookie, &jwt.StandardClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("SECRET")), nil
	})
	sessionID := token.Claims.(*jwt.StandardClaims).Issuer
	if sessionID == "" {
		return "", "", fmt.Errorf("Empty value")
	}
	val, err := redisClient.Get(rctx, sessionID).Result()
	if err != nil {
		return "", "", fmt.Errorf("Error retrieving session")
	}

	return val, sessionID, nil
}

func RefreshToken(redisClient *redis.Client, ctx *fiber.Ctx, rctx context.Context, db *pgxpool.Pool) (*fiber.Cookie, error) {
	if uid, sid, err := GetUidAndSid(redisClient, ctx, rctx, db); err != nil {
		return GetClearedCookie(), err
	} else {
		redisClient.Del(rctx, sid)
		if cookie, err := Authorize(redisClient, rctx, uid); err != nil {
			return GetClearedCookie(), err
		} else {
			return cookie, nil
		}
	}
}

func DeleteSession(redisClient *redis.Client, ctx context.Context, sid string) {
	redisClient.Del(ctx, sid)
}
