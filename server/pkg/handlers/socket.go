package handlers

import (
	"encoding/json"
	"log"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
}

/*
	Messages come in like this, different to my last go projects:
	{ "event_type":string , "data":json }
*/

type decodedMsg struct {
	Type string      `json:"event_type"`
	Data interface{} `json:"data"`
}

func SendSocketErrorMessage(m string, c *websocket.Conn) {
	c.WriteJSON(map[string]string{
		"msg": m,
	})
}

func handleConnection(h handler, ctx *fasthttp.RequestCtx, c *websocket.Conn) {
	for {
		if _, p, err := c.ReadMessage(); err != nil {
			log.Println("ws reader error:", err)
			return
		} else {
			if len(p) == 4 {
				if string(p) == "PING" {
					continue
				}
			}
			decoded := &decodedMsg{}
			if err := json.Unmarshal(p, decoded); err != nil {
				log.Println("Invalid message - connection closed")
				c.Close()
				return
			} else {
				log.Println("Message event recieved:", decoded.Type)
				if err := handleSocketEvent(decoded.Data, decoded.Type, h, c); err != nil {
					log.Println("Socket event error:", err)
					SendSocketErrorMessage(err.Error(), c)
				}
			}
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
