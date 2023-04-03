package handlers

import (
	"os"

	"github.com/fasthttp/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	attachmentserver "github.com/web-stuff-98/psql-social/pkg/attachmentServer"
	callServer "github.com/web-stuff-98/psql-social/pkg/callServer"
	"github.com/web-stuff-98/psql-social/pkg/channelRTCserver"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
)

var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		if os.Getenv("ENVIRONMENT") != "PRODUCTION" {
			return true
		} else {
			// need to add rule here before deploying
			return true
		}
	},
}

type handler struct {
	DB               *pgxpool.Pool
	RedisClient      *redis.Client
	SocketServer     *socketServer.SocketServer
	CallServer       *callServer.CallServer
	ChannelRTCServer *channelRTCserver.ChannelRTCServer
	AttachmentServer *attachmentserver.AttachmentServer
}

func ResponseMessage(ctx *fasthttp.RequestCtx, msg string, code int) {
	ctx.SetStatusCode(code)
	ctx.WriteString(msg)
}

func New(db *pgxpool.Pool, rdb *redis.Client, ss *socketServer.SocketServer, cs *callServer.CallServer, cRTCs *channelRTCserver.ChannelRTCServer, as *attachmentserver.AttachmentServer) handler {
	return handler{db, rdb, ss, cs, cRTCs, as}
}
