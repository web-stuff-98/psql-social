package handlers

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
)

type handler struct {
	DB           *pgxpool.Pool
	RedisClient  *redis.Client
	SocketServer *socketServer.SocketServer
}

func ResponseMessage(ctx *fasthttp.RequestCtx, msg string, code int) {
	ctx.SetStatusCode(code)
	ctx.WriteString(msg)
}

func New(db *pgxpool.Pool, rdb *redis.Client, ss *socketServer.SocketServer) handler {
	return handler{db, rdb, ss}
}
