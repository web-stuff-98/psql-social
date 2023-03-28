package authHelpers

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

func GenerateTokenAndSession(redisClient *redis.Client, ctx context.Context, uid string) (string, error) {
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
	cmd := redisClient.Set(ctx, sid.String(), uid, sessionDuration)
	if cmd.Err() != nil {
		return "", err
	}
	return token, err
}

func GetUidFromToken(redisClient *redis.Client, ctx context.Context, db *pgx.Conn, inToken string) (string, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic in get user ID from session token helper function")
		}
	}()
	token, err := jwt.ParseWithClaims(inToken, &jwt.StandardClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("SECRET")), nil
	})
	sessionID := token.Claims.(*jwt.StandardClaims).Issuer
	if sessionID == "" {
		return "", fmt.Errorf("Empty value")
	}
	val, err := redisClient.Get(ctx, sessionID).Result()
	if err != nil {
		return "", fmt.Errorf("Error retrieving session")
	}
	return val, nil
}
