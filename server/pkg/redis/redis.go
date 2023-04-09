package rdb

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

func Init() *redis.Client {
	log.Println("Connecting Redis client")

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	var rdb *redis.Client

	opt, _ := redis.ParseURL(redisURL)
	if os.Getenv("ENVIRONMENT") == "PRODUCTION" {
		rdb = redis.NewClient(opt)
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		})
	}

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Redis client connected")

	return rdb
}
