package handlers

import (
	"log"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func handleConnection(h handler, ctx *fasthttp.RequestCtx, c *websocket.Conn) {
	for {
		if _, p, err := c.ReadMessage(); err != nil {
			log.Println("ws reader error:", err)
			return
		} else {
			log.Println("Message recieved:", string(p))
		}
	}
}

func (h handler) WebSocketEndpoint(ctx *fasthttp.RequestCtx) {
	if err := upgrader.Upgrade(ctx, func(c *websocket.Conn) {
		handleConnection(h, ctx, c)
	}); err != nil {
		ctx.Error("Internal error", fasthttp.StatusInternalServerError)
	}
}
