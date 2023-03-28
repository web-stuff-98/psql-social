package handlers

import (
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
)

type handler struct {
	DB           *pgx.Conn
	RedisClient  *redis.Client
	SocketServer *socketServer.SocketServer
}

func New(db *pgx.Conn, rdb *redis.Client, ss *socketServer.SocketServer) handler {
	return handler{db, rdb, ss}
}
