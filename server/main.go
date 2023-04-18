package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
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

type spaHandler struct {
	staticPath string
	indexPath  string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db := db.Init()
	rdb := rdb.Init()
	csdc := make(chan string)    // call server disconnect channel
	cRTCsdc := make(chan string) // room channel WebRTC chat disconnect channel
	ss := socketServer.Init(csdc, cRTCsdc)
	as := attachmentServer.Init(ss, db)
	cRTCs := channelRTCserver.Init(ss, db, cRTCsdc)
	cs := callServer.Init(ss, csdc)
	sl := socketLimiter.Init(rdb)

	sqlBytes, err := ioutil.ReadFile("./schema.sql")
	if err != nil {
		log.Fatalf("Unable to read SQL file: %v\n", err)
	}
	sql := string(sqlBytes)
	if _, err := db.Exec(context.Background(), sql); err != nil {
		log.Fatalf("Unable to execute SQL schema: %v\n", err)
	}

	defer db.Close()

	h := handlers.New(db, rdb, ss, cs, cRTCs, as, sl)
	app := fiber.New()

	allowedOrigin := "http://localhost:5173,http://localhost:8080"
	if os.Getenv("ENVIRONMENT") == "PRODUCTION" {
		allowedOrigin = "https://psql-social.herokuapp.com"
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigin,
		AllowMethods:     "POST, PATCH, PUT, GET, OPTIONS, DELETE",
		AllowCredentials: true,
	}))

	app.Post("/api/acc/login", mw.BasicRateLimiter(h.Login, mw.SimpleLimiterOpts{
		Window:        time.Minute * 20,
		MaxReqs:       3,
		BlockDuration: time.Hour * 12,
		Message:       "Too many requests",
		RouteName:     "login",
	}, rdb, db))
	app.Post("/api/acc/logout", mw.BasicRateLimiter(h.Logout, mw.SimpleLimiterOpts{
		Window:        time.Minute * 20,
		MaxReqs:       3,
		BlockDuration: time.Hour * 12,
		Message:       "Too many requests",
		RouteName:     "logout",
	}, rdb, db))
	app.Post("/api/acc/register", mw.BasicRateLimiter(h.Register, mw.SimpleLimiterOpts{
		Window:        time.Minute * 20,
		MaxReqs:       3,
		BlockDuration: time.Hour * 12,
		Message:       "Too many requests",
		RouteName:     "register",
	}, rdb, db))
	app.Post("/api/acc/refresh", mw.BasicRateLimiter(h.Refresh, mw.SimpleLimiterOpts{
		Window:        time.Second * 1,
		MaxReqs:       10,
		BlockDuration: time.Hour * 3,
		Message:       "Too many requests",
		RouteName:     "refresh",
	}, rdb, db))
	app.Post("/api/acc/bio", mw.BasicRateLimiter(h.UpdateBio, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       10,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "update-bio",
	}, rdb, db))
	app.Post("/api/acc/pfp", mw.BasicRateLimiter(h.UploadPfp, mw.SimpleLimiterOpts{
		Window:        time.Minute * 2,
		MaxReqs:       10,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "upload-pfp",
	}, rdb, db))
	app.Get("/api/acc/uids", mw.BasicRateLimiter(h.GetConversees, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-conversees",
	}, rdb, db))
	app.Get("/api/acc/friends", mw.BasicRateLimiter(h.GetFriends, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-friends",
	}, rdb, db))
	app.Get("/api/acc/blocked", mw.BasicRateLimiter(h.GetBlocked, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-blocked",
	}, rdb, db))
	app.Get("/api/acc/conv/:id", mw.BasicRateLimiter(h.GetConversation, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-conversation",
	}, rdb, db))

	app.Post("/api/room", mw.BasicRateLimiter(h.CreateRoom, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "create-room",
	}, rdb, db))
	app.Post("/api/room/:id/img", mw.BasicRateLimiter(h.UploadRoomImage, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "upload-room-image",
	}, rdb, db))
	app.Get("/api/room/:id/img", mw.BasicRateLimiter(h.GetRoomImage, mw.SimpleLimiterOpts{
		Window:        time.Second * 30,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-room-image",
	}, rdb, db))
	app.Get("/api/rooms", mw.BasicRateLimiter(h.GetRooms, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-rooms",
	}, rdb, db))
	app.Patch("/api/room/:id", mw.BasicRateLimiter(h.UpdateRoom, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "update-room",
	}, rdb, db))
	app.Get("/api/room/:id", mw.BasicRateLimiter(h.GetRoom, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-room",
	}, rdb, db))
	app.Delete("/api/room/:id", mw.BasicRateLimiter(h.DeleteRoom, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "delete-room",
	}, rdb, db))
	app.Get("/api/room/channel/:id", mw.BasicRateLimiter(h.GetRoomChannel, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-room-channel",
	}, rdb, db))
	app.Patch("/api/room/channel/:id", mw.BasicRateLimiter(h.UpdateRoomChannel, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "update-room-channel",
	}, rdb, db))
	app.Delete("/api/room/channel/:id", mw.BasicRateLimiter(h.DeleteRoomChannel, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "delete-room-channel",
	}, rdb, db))
	app.Post("/api/room/:id/channels", mw.BasicRateLimiter(h.CreateRoomChannel, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "create-room-channel",
	}, rdb, db))
	app.Get("/api/room/channels/:id", mw.BasicRateLimiter(h.GetRoomChannels, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-room-channels",
	}, rdb, db))

	app.Get("/api/user/bio/:id", mw.BasicRateLimiter(h.GetUserBio, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-bio",
	}, rdb, db))
	app.Get("/api/user/pfp/:id", mw.BasicRateLimiter(h.GetUserPfp, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-pfp",
	}, rdb, db))
	app.Get("/api/user/:id", mw.BasicRateLimiter(h.GetUser, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-user",
	}, rdb, db))
	app.Post("/api/user/name", mw.BasicRateLimiter(h.GetUserByName, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-user-by-name",
	}, rdb, db))
	app.Post("/api/user/search", mw.BasicRateLimiter(h.SearchUsers, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       90,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "search users",
	}, rdb, db))

	app.Post("/api/attachment/metadata", mw.BasicRateLimiter(h.CreateAttachmentMetadata, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "create-attachment-metadata",
	}, rdb, db))
	app.Post("/api/attachment/chunk/:id", mw.BasicRateLimiter(h.UploadAttachmentChunk, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "upload-attachment-chunk",
	}, rdb, db))
	app.Get("/api/attachment/:id", mw.BasicRateLimiter(h.DownloadAttachment, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       30,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "download-attachment-chunks",
	}, rdb, db))
	app.Get("/api/attachment/video/:id", mw.BasicRateLimiter(h.GetAttachmentVideoPartialContent, mw.SimpleLimiterOpts{
		Window:        time.Minute * 1,
		MaxReqs:       80,
		BlockDuration: time.Minute * 10,
		Message:       "Too many requests",
		RouteName:     "get-attachment-video-chunks",
	}, rdb, db))

	app.Use("/api/ws", h.WebSocketAuth)
	app.Get("/api/ws", h.WebSocketHandler())

	app.Static("/", "./dist")

	log.Printf("API opening on port %v", os.Getenv("PORT"))
	log.Fatalln(app.Listen(":" + os.Getenv("PORT")))
}
