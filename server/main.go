package main

import (
	"context"
	"log"
	"os"

	"github.com/adhityaramadhanus/fasthttpcors"
	"github.com/fasthttp/router"
	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
	"github.com/web-stuff-98/psql-social/pkg/db"
	"github.com/web-stuff-98/psql-social/pkg/handlers"
	rdb "github.com/web-stuff-98/psql-social/pkg/redis"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db := db.Init()
	rdb := rdb.Init()
	ss := socketServer.Init()

	defer db.Close(context.Background())

	h := handlers.New(db, rdb, ss)

	r := router.New()
	r.POST("/api/acc/login", h.Login)
	r.POST("/api/acc/logout", h.Logout)
	r.POST("/api/acc/register", h.Register)
	r.POST("/api/acc/refresh", h.Refresh)
	r.GET("/api/ws", h.WebSocketEndpoint)

	corsHandler := fasthttpcors.NewCorsHandler(fasthttpcors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowCredentials: true,
	})

	log.Printf("API opening on port %v", os.Getenv("PORT"))
	log.Fatalln(fasthttp.ListenAndServe(":"+os.Getenv("PORT"), corsHandler.CorsMiddleware(r.Handler)))
}
