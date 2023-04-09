package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Init() *pgxpool.Pool {
	var config *pgxpool.Config

	parsedConfig, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalln("Failed to parse DB URL config")
	}

	if os.Getenv("ENVIRONMENT") != "PRODUCTION" {
		if err != nil {
			log.Fatalln("Failed to parse DB URL config")
		}
		parsedConfig.MaxConnLifetime = time.Second * 3
		// 1000 because there is no connection limit for local development db
		parsedConfig.MaxConns = 1000
	} else {
		if err != nil {
			log.Fatalln("Failed to parse DB URL config")
		}
		parsedConfig.MaxConnLifetime = time.Second * 3
		// heroku docs for the db addon say that 50 is the max num of connections
		parsedConfig.MaxConns = 50
	}
	config = parsedConfig

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalln("Unable to create pool:", err)
		return nil
	}
	log.Println("Created pool")

	//go monitorPool(pool)

	return pool
}

func monitorPool(pool *pgxpool.Pool) {
	ticker := time.NewTicker(50 * time.Millisecond)
	for _ = range ticker.C {
		stat := pool.Stat()
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
		fmt.Printf("POOL STAT:%v\n", stat.IdleConns())
		fmt.Printf("POOL TOTAL CONNECTIONS:%v\n", stat.TotalConns())
		fmt.Printf("POOL CONSTRUCTING CONNECTIONS:%v\n", stat.ConstructingConns())
	}
}
