package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
	"github.com/web-stuff-98/psql-social/pkg/db"
	"github.com/web-stuff-98/psql-social/pkg/handlers"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db := db.Init()
	h := handlers.New(db)

	log.Printf("API opening on port %v", os.Getenv("PORT"))
	log.Fatalln(fasthttp.ListenAndServe(":"+os.Getenv("PORT"), func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/api/ping":
			h.Ping(ctx)
		case "/api/acc/login":
			h.Login(ctx)
		case "/api/acc/register":
			h.Register(ctx)
		default:
			ctx.Error("Not found", fasthttp.StatusNotFound)
		}
	}))
}
