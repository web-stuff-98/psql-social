package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Init() *pgxpool.Pool {
	config, err := pgxpool.ParseConfig(os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalln("Failed to parse DB URL config")
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalln("Unable to create pool:", err)
		return nil
	}
	log.Println("Created pool")
	return pool
}
