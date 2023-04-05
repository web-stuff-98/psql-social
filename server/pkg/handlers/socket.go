package handlers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
)

/*
	Messages come in like this, different to my last go projects:
	{ "event_type":string , "data":json }

	There is a seperate websocket endpoint for attachment chunk
	data
*/

type decodedMsg struct {
	Type string                 `json:"event_type"`
	Data map[string]interface{} `json:"data"`
}

func SendSocketErrorMessage(m string, c *websocket.Conn) {
	c.WriteJSON(map[string]string{
		"msg": m,
	})
}

func handleConnection(h handler, ctx *fasthttp.RequestCtx, uid string, c *websocket.Conn) {
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
				if err := handleSocketEvent(decoded.Data, decoded.Type, h, uid, c); err != nil {
					log.Println("Socket event error:", err)
					SendSocketErrorMessage(err.Error(), c)
				}
			}
		}
	}
}

func (h handler) WebSocketEndpoint(ctx *fasthttp.RequestCtx) {
	if uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, context.Background(), h.DB); err != nil {
		ResponseMessage(ctx, "Forbidden - Log in to gain access", fasthttp.StatusForbidden)
	} else {
		if err := upgrader.Upgrade(ctx, func(c *websocket.Conn) {
			h.SocketServer.RegisterConn <- socketServer.ConnnectionData{
				Uid:  uid,
				Conn: c,
			}
			defer func() {
				h.SocketServer.UnregisterConn <- c
			}()
			handleConnection(h, ctx, uid, c)
		}); err != nil {
			log.Println(err)
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		}
	}
}
