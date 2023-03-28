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
		_, p, err := c.ReadMessage()
		if err != nil {
			log.Println("ws reader error:", err)
			return
		}
		log.Println("Message recieved:", string(p))
	}
}

func (h handler) WebSocketEndpoint(ctx *fasthttp.RequestCtx) {
	err := upgrader.Upgrade(ctx, func(c *websocket.Conn) {
		handleConnection(h, ctx, c)
	})
	if err != nil {
		ctx.Error("Internal error", fasthttp.StatusInternalServerError)
	}
}
