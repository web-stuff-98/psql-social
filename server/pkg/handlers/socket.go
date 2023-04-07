package handlers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
)

type decodedMsg struct {
	Type string                 `json:"event_type"`
	Data map[string]interface{} `json:"data"`
}

func SendSocketErrorMessage(m string, c *websocket.Conn) {
	c.WriteJSON(map[string]string{
		"msg": m,
	})
}

func handleConnection(h handler, ctx *fiber.Ctx, uid string, c *websocket.Conn) {
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

func (h handler) WebSocketEndpoint(ctx *fiber.Ctx) error {
	if uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, context.Background(), h.DB); err != nil {
		return ctx.Status(fiber.StatusForbidden).SendString("Forbidden - Log in to gain access")
	} else {
		if websocket.IsWebSocketUpgrade(ctx) {
			ctx.Locals("uid", uid)
			return ctx.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}

func (h handler) WebSocketHandler(c *websocket.Conn) {
	defer func() {
		h.SocketServer.UnregisterConn <- c
	}()
	h.SocketServer.RegisterConn <- socketServer.ConnnectionData{
		Uid:  c.Locals("uid").(string),
		Conn: c,
	}
	handleConnection(h, nil, c.Locals("uid").(string), c)
}
