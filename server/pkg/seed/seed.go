package seed

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GenerateSeed(userCount int, roomCount int, db *pgxpool.Pool) {
	users := []string{}
	rooms := []string{}

	// generate users
	for i := 0; i < roomCount; i++ {
		uid := generateUser(i, db)
		users = append(users, uid)
	}

	// generate rooms
	for i := 0; i < userCount; i++ {
		randInt, err := rand.Int(rand.Reader, big.NewInt(int64(len(users))))
		if err != nil {
			log.Fatalf("Error generating random index in seed function:%v", err)
		}
		rid := generateRoom(i, users[randInt.Int64()], db)
		rooms = append(rooms, rid)
	}
}

func generateRoom(index int, uid string, db *pgxpool.Pool) string {
	var id string
	if err := db.QueryRow(context.Background(), `
	INSERT INTO rooms (
		name,
		private,
		author_id,
		seeded
	) VALUES($1,$2,$3,TRUE) RETURNING id;
	`, fmt.Sprintf("TestAcc%v", index+1), false, uid).Scan(&id); err != nil {
		log.Fatalf("Error in generate room seed function:%v", err)
	}
	return id
}

func generateUser(index int, db *pgxpool.Pool) string {
	var id string
	if err := db.QueryRow(context.Background(), `
	INSERT INTO users (
		username,
		role,
		seeded,
		password
	) VALUES($1,$2,TRUE,'$2a$12$0/nq/i75R27Tu2dgqvSEiuKCrcmB20ibWIbbPEseG72Brt8yGeAkG') RETURNING id;
	`, fmt.Sprintf("TestAcc%v", index+1), "USER").Scan(&id); err != nil {
		log.Fatalf("Error in generate user seed function:%v", err)
	}
	return id
}
