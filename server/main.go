package main

import (
	"log"
	"os"

	"github.com/adhityaramadhanus/fasthttpcors"
	"github.com/fasthttp/router"
	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
	attachmentServer "github.com/web-stuff-98/psql-social/pkg/attachmentServer"
	callServer "github.com/web-stuff-98/psql-social/pkg/callServer"
	"github.com/web-stuff-98/psql-social/pkg/channelRTCserver"
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
	csdc := make(chan string)
	cRTCsdc := make(chan string)
	ss := socketServer.Init(csdc, cRTCsdc)
	as := attachmentServer.Init(ss, db)
	cRTCs := channelRTCserver.Init(ss, db, cRTCsdc)
	cs := callServer.Init(ss, csdc)

	defer db.Close()

	h := handlers.New(db, rdb, ss, cs, cRTCs, as)

	r := router.New()

	r.POST("/api/acc/login", h.Login)
	r.POST("/api/acc/logout", h.Logout)
	r.POST("/api/acc/register", h.Register)
	r.POST("/api/acc/refresh", h.Refresh)
	r.POST("/api/acc/bio", h.UpdateBio)
	r.POST("/api/acc/pfp", h.UploadPfp)
	r.GET("/api/acc/uids", h.GetConversees)
	r.GET("/api/acc/friends", h.GetFriends)
	r.GET("/api/acc/conv/{id}", h.GetConversation)

	r.POST("/api/room", h.CreateRoom)
	r.GET("/api/rooms", h.GetRooms)
	r.PATCH("/api/room/{id}", h.UpdateRoom)
	r.GET("/api/room/{id}", h.GetRoom)
	r.DELETE("/api/room/{id}", h.DeleteRoom)
	r.GET("/api/room/channel/{id}", h.GetRoomChannel)
	r.PATCH("/api/room/channel/{id}", h.UpdateRoomChannel)
	r.DELETE("/api/room/channel/{id}", h.DeleteRoomChannel)
	r.POST("/api/room/{id}/channels", h.CreateRoomChannel)
	r.GET("/api/room/channels/{id}", h.GetRoomChannels)

	r.GET("/api/user/bio/{id}", h.GetUserBio)
	r.GET("/api/user/pfp/{id}", h.GetUserPfp)
	r.GET("/api/user/{id}", h.GetUser)
	r.POST("/api/user/name", h.GetUserByName)

	r.GET("/api/ws", h.WebSocketEndpoint)

	corsHandler := fasthttpcors.NewCorsHandler(fasthttpcors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"POST", "PATCH", "PUT", "GET", "OPTIONS", "DELETE"},
		AllowCredentials: true,
	})

	log.Printf("API opening on port %v", os.Getenv("PORT"))
	log.Fatalln(fasthttp.ListenAndServe(":"+os.Getenv("PORT"), corsHandler.CorsMiddleware(r.Handler)))
}
