package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Init() *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalln("Unable to create pool:", err)
		return nil
	} else {
		log.Println("Created pool")
		return pool
	}
}
