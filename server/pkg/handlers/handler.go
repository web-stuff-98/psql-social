package handlers

import (
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	attachmentServer "github.com/web-stuff-98/psql-social/pkg/attachmentServer"
	callServer "github.com/web-stuff-98/psql-social/pkg/callServer"
	"github.com/web-stuff-98/psql-social/pkg/channelRTCserver"
	socketLimiter "github.com/web-stuff-98/psql-social/pkg/socketLimiter"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
)

type handler struct {
	DB               *pgxpool.Pool
	RedisClient      *redis.Client
	SocketServer     *socketServer.SocketServer
	CallServer       *callServer.CallServer
	ChannelRTCServer *channelRTCserver.ChannelRTCServer
	AttachmentServer *attachmentServer.AttachmentServer
	SocketLimiter    *socketLimiter.SocketLimiter
	// users that are pending deletion. Needed to cancel user deletes if they log back in
	UserDeleteList sync.Map
}

func New(db *pgxpool.Pool, rdb *redis.Client, ss *socketServer.SocketServer, cs *callServer.CallServer, cRTCs *channelRTCserver.ChannelRTCServer, as *attachmentServer.AttachmentServer, sl *socketLimiter.SocketLimiter, userDeleteList sync.Map) handler {
	return handler{db, rdb, ss, cs, cRTCs, as, sl, userDeleteList}
}
