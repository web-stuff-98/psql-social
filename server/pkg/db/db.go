package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

func Init() *pgx.Conn {
	if conn, err := pgx.Connect(context.Background(), os.Getenv("DB_URL")); err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
		return nil
	} else {
		defer conn.Close(context.Background())
		log.Println("Connected to postgres")
		return conn
	}
}
