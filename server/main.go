package main

import (
	"log"
	"os"
	"time"

	"github.com/adhityaramadhanus/fasthttpcors"
	"github.com/fasthttp/router"
	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
	attachmentServer "github.com/web-stuff-98/psql-social/pkg/attachmentServer"
	callServer "github.com/web-stuff-98/psql-social/pkg/callServer"
	"github.com/web-stuff-98/psql-social/pkg/channelRTCserver"
	"github.com/web-stuff-98/psql-social/pkg/db"
	"github.com/web-stuff-98/psql-social/pkg/handlers"
	mw "github.com/web-stuff-98/psql-social/pkg/handlers/middleware"
	rdb "github.com/web-stuff-98/psql-social/pkg/redis"
	socketLimiter "github.com/web-stuff-98/psql-social/pkg/socketLimiter"
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
	sl := socketLimiter.Init(rdb)
	defer db.Close()

	h := handlers.New(db, rdb, ss, cs, cRTCs, as, sl)

	r := router.New()

	r.POST("/api/acc/login", mw.BasicRateLimiter(h.Login, mw.SimpleLimiterOpts{
		Window:        time.Minute * 20,
		MaxReqs:       3,
		BlockDuration: time.Hour * 12,
		Message:       "Too many requests",
		RouteName:     "login",
	}, rdb, db))
	r.POST("/api/acc/logout", mw.BasicRateLimiter(h.Logout, mw.SimpleLimiterOpts{
		Window:        time.Minute * 20,
		MaxReqs:       3,
		BlockDuration: time.Hour * 12,
		Message:       "Too many requests",
		RouteName:     "logout",
	}, rdb, db))
	r.POST("/api/acc/register", mw.BasicRateLimiter(h.Register, mw.SimpleLimiterOpts{
		Window:        time.Minute * 20,
		MaxReqs:       3,
		BlockDuration: time.Hour * 12,
		Message:       "Too many requests",
		RouteName:     "register",
	}, rdb, db))
	r.POST("/api/acc/refresh", mw.BasicRateLimiter(h.Refresh, mw.SimpleLimiterOpts{
		Window:        time.Second * 1,
		MaxReqs:       10,
		BlockDuration: time.Hour * 3,
		Message:       "Too many requests",
		RouteName:     "refresh",
	}, rdb, db))
	r.POST("/api/acc/bio", mw.BasicRateLimiter(h.UpdateBio, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       10,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "update-bio",
	}, rdb, db))
	r.POST("/api/acc/pfp", mw.BasicRateLimiter(h.UploadPfp, mw.SimpleLimiterOpts{
		Window:        time.Minute * 2,
		MaxReqs:       10,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "upload-pfp",
	}, rdb, db))
	r.GET("/api/acc/uids", mw.BasicRateLimiter(h.GetConversees, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-conversees",
	}, rdb, db))
	r.GET("/api/acc/friends", mw.BasicRateLimiter(h.GetFriends, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-friends",
	}, rdb, db))
	r.GET("/api/acc/conv/{id}", mw.BasicRateLimiter(h.GetConversation, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-conversation",
	}, rdb, db))

	r.POST("/api/room", mw.BasicRateLimiter(h.CreateRoom, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "create-room",
	}, rdb, db))
	r.POST("/api/room/{id}/img", mw.BasicRateLimiter(h.UploadRoomImage, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "upload-room-image",
	}, rdb, db))
	r.GET("/api/room/{id}/img", mw.BasicRateLimiter(h.GetRoomImage, mw.SimpleLimiterOpts{
		Window:        time.Second * 30,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-room-image",
	}, rdb, db))
	r.GET("/api/rooms", mw.BasicRateLimiter(h.GetRooms, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-rooms",
	}, rdb, db))
	r.PATCH("/api/room/{id}", mw.BasicRateLimiter(h.UpdateRoom, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "update-room",
	}, rdb, db))
	r.GET("/api/room/{id}", mw.BasicRateLimiter(h.GetRoom, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-room",
	}, rdb, db))
	r.DELETE("/api/room/{id}", mw.BasicRateLimiter(h.DeleteRoom, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "delete-room",
	}, rdb, db))
	r.GET("/api/room/channel/{id}", mw.BasicRateLimiter(h.GetRoomChannel, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-room-channel",
	}, rdb, db))
	r.PATCH("/api/room/channel/{id}", mw.BasicRateLimiter(h.UpdateRoomChannel, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "update-room-channel",
	}, rdb, db))
	r.DELETE("/api/room/channel/{id}", mw.BasicRateLimiter(h.DeleteRoomChannel, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "delete-room-channel",
	}, rdb, db))
	r.POST("/api/room/{id}/channels", mw.BasicRateLimiter(h.CreateRoomChannel, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "create-room-channel",
	}, rdb, db))
	r.GET("/api/room/channels/{id}", mw.BasicRateLimiter(h.GetRoomChannels, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-room-channels",
	}, rdb, db))

	r.GET("/api/user/bio/{id}", mw.BasicRateLimiter(h.GetUserBio, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-bio",
	}, rdb, db))
	r.GET("/api/user/pfp/{id}", mw.BasicRateLimiter(h.GetUserPfp, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-pfp",
	}, rdb, db))
	r.GET("/api/user/{id}", mw.BasicRateLimiter(h.GetUser, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-user",
	}, rdb, db))
	r.POST("/api/user/name", mw.BasicRateLimiter(h.GetUserByName, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-user-by-name",
	}, rdb, db))

	r.POST("/api/attachment/metadata", mw.BasicRateLimiter(h.CreateAttachmentMetadata, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "create-attachment-metadata",
	}, rdb, db))
	r.POST("/api/attachment/chunk/{id}", mw.BasicRateLimiter(h.UploadAttachmentChunk, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "upload-attachment-chunk",
	}, rdb, db))
	r.GET("/api/attachment/{id}", mw.BasicRateLimiter(h.DownloadAttachment, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "download-attachment-chunk",
	}, rdb, db))

	r.GET("/api/ws", h.WebSocketEndpoint)

	corsHandler := fasthttpcors.NewCorsHandler(fasthttpcors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"POST", "PATCH", "PUT", "GET", "OPTIONS", "DELETE"},
		AllowCredentials: true,
	})

	log.Printf("API opening on port %v", os.Getenv("PORT"))
	log.Fatalln(fasthttp.ListenAndServe(":"+os.Getenv("PORT"), corsHandler.CorsMiddleware(r.Handler)))
}
