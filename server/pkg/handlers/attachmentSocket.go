package handlers

import (
	"context"
	"log"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	attachmentServer "github.com/web-stuff-98/psql-social/pkg/attachmentServer"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
)

func handleAttachmentConnection(h handler, ctx *fasthttp.RequestCtx, uid string, c *websocket.Conn) {
	for {
		if _, p, err := c.ReadMessage(); err != nil {
			log.Println("attachment ws reader error:", err)
			return
		} else {
			h.AttachmentServer.ChunkChan <- attachmentServer.ChunkData{
				Uid:  uid,
				Data: p,
			}
		}
	}
}

func (h handler) AttachmentWebSocketEndpoint(ctx *fasthttp.RequestCtx) {
	if uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, context.Background(), h.DB); err != nil {
		ResponseMessage(ctx, "Forbidden - Log in to gain access", fasthttp.StatusForbidden)
	} else {
		if err := upgrader.Upgrade(ctx, func(c *websocket.Conn) {
			handleAttachmentConnection(h, ctx, uid, c)
		}); err != nil {
			log.Println(err)
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		}
	}
}
