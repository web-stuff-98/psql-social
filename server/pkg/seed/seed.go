package seed

import (
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Generate(db *pgxpool.Pool) {
	prod := os.Getenv("ENVIRONMENT") == "PRODUCTION"
	count := 20
	if prod {
		count = 100
	}

	log.Println("Generating seed...")

	for i := 0; i < count; i++ {
		
	}
}

func GenerateUser(i int) {

}

func GenerateRoom(i int) {

}
