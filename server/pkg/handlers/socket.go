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

func (h handler) WebSocketHandler() func(*fiber.Ctx) error {
	return websocket.New(func(c *websocket.Conn) {
		h.SocketServer.RegisterConn <- socketServer.ConnectionData{
			Uid:  c.Locals("uid").(string),
			Conn: c,
		}
		defer func() {
			h.SocketServer.UnregisterConn <- c
		}()
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
				if err = json.Unmarshal(p, decoded); err != nil {
					log.Println("Invalid message - connection closed")
					c.Close()
					return
				} else {
					if err := handleSocketEvent(decoded.Data, decoded.Type, h, c.Locals("uid").(string), c); err != nil {
						SendSocketErrorMessage(err.Error(), c)
					}
				}
			}
		}
	})
}

func (h handler) WebSocketAuth(ctx *fiber.Ctx) error {
	if uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, context.Background(), h.DB); err != nil {
		return fiber.ErrForbidden
	} else {
		if websocket.IsWebSocketUpgrade(ctx) {
			ctx.Locals("uid", uid)
			ctx.Locals("open_convs", make(map[string]struct{}))
			return ctx.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}
