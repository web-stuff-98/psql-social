package authHelpers

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	socketMessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
)

func createCookie(token string, expiry time.Time) *fiber.Cookie {
	cookie := new(fiber.Cookie)
	cookie.Name = "session_token"
	cookie.Value = token
	cookie.Expires = expiry
	cookie.MaxAge = 120
	cookie.Secure = os.Getenv("ENVIRONMENT") == "PRODUCTION"
	cookie.HTTPOnly = true
	cookie.SameSite = "Strict"
	cookie.Path = "/"
	return cookie
}

func GetClearedCookie() *fiber.Cookie {
	cookie := new(fiber.Cookie)
	cookie.Name = "session_token"
	cookie.Value = ""
	cookie.Expires = time.Unix(0, 0)
	cookie.MaxAge = -1
	cookie.Secure = os.Getenv("ENVIRONMENT") == "PRODUCTION"
	cookie.HTTPOnly = true
	cookie.SameSite = "Strict"
	cookie.Path = "/"
	return cookie
}

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

func GenerateCookieAndSession(redisClient *redis.Client, ctx context.Context, uid string) (*fiber.Cookie, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic in generate token helper function")
		}
	}()

	sid := uuid.New()
	sessionDuration := time.Minute * 2
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Issuer:    sid.String(),
		ExpiresAt: time.Now().Add(sessionDuration).Unix(),
	})
	token, err := claims.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		log.Fatalln("Error in GenerateCookieAndSession helper function generating token:", err)
	}
	cmd := redisClient.Set(ctx, sid.String(), uid, sessionDuration)
	if cmd.Err() != nil {
		log.Fatalln("Redis error in GenerateCookieAndSession helper function:", cmd.Err())
	}
	cookie := createCookie(token, time.Now().Add(sessionDuration))

	return cookie, nil
}

func GetUidAndSidFromCookie(redisClient *redis.Client, ctx *fiber.Ctx, rctx context.Context, db *pgxpool.Pool) (uid string, sid string, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic in get user ID from session token helper function")
		}
	}()

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
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic in refresh session token helper function")
		}
	}()

	if uid, sid, err := GetUidAndSidFromCookie(redisClient, ctx, rctx, db); err != nil {
		return GetClearedCookie(), err
	} else {
		redisClient.Del(rctx, sid)
		if cookie, err := GenerateCookieAndSession(redisClient, rctx, uid); err != nil {
			return GetClearedCookie(), err
		} else {
			return cookie, nil
		}
	}
}

func DeleteSession(redisClient *redis.Client, ctx context.Context, sid string) {
	redisClient.Del(ctx, sid)
}

func DeleteAccount(uid string, db *pgxpool.Pool, ss *socketServer.SocketServer, sleep bool) error {
	if sleep {
		time.Sleep(time.Minute * 20)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if _, err := db.Exec(ctx, "DELETE FROM users WHERE id = $1", uid); err != nil {
		return err
	}

	roomSubs := []string{}

	if rows, err := db.Query(ctx, "SELECT id FROM rooms WHERE id = $1", uid); err != nil {
		return err
	} else {
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			roomSubs = append(roomSubs, fmt.Sprintf("channel:%v", id))
		}
	}

	for _, subName := range roomSubs {
		changeData := make(map[string]interface{})
		changeData["ID"] = strings.Split(subName, ":")[1]
		ss.SendDataToSub <- socketServer.SubscriptionMessageData{
			SubName: subName,
			Data: socketMessages.ChangeEvent{
				Type:   "DELETE",
				Data:   changeData,
				Entity: "ROOM",
			},
			MessageType: "CHANGE",
		}
	}

	return nil
}
