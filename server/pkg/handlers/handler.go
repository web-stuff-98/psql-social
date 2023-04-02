package handlers

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	callServer "github.com/web-stuff-98/psql-social/pkg/callServer"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
)

type handler struct {
	DB           *pgxpool.Pool
	RedisClient  *redis.Client
	SocketServer *socketServer.SocketServer
	CallServer   *callServer.CallServer
}

func ResponseMessage(ctx *fasthttp.RequestCtx, msg string, code int) {
	ctx.SetStatusCode(code)
	ctx.WriteString(msg)
}

func New(db *pgxpool.Pool, rdb *redis.Client, ss *socketServer.SocketServer, cs *callServer.CallServer) handler {
	return handler{db, rdb, ss, cs}
}
