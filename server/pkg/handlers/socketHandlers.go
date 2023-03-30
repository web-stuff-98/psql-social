package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
	socketvalidation "github.com/web-stuff-98/psql-social/socketValidation"
)

func handleSocketEvent(data map[string]interface{}, event string, h handler, uid string, c *websocket.Conn) error {
	var err error

	switch event {
	case "JOIN_ROOM":
		err = joinRoom(data, h, uid, c)
	case "LEAVE_ROOM":
		err = leaveRoom(data, h, uid, c)
	default:
		return fmt.Errorf("Unrecognized event type")
	}

	return err
}

func UnmarshalMap(m map[string]interface{}, s interface{}) error {
	v := validator.New()
	if err := v.Struct(s); err != nil {
		return fmt.Errorf("Bad request")
	}
	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("Bad request")
	}
	err = json.Unmarshal(b, s)
	if err != nil {
		return fmt.Errorf("Bad request")
	}
	return nil
}

func joinRoom(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketvalidation.JoinLeaveRoomData{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	roomExists := false
	if err = h.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM rooms WHERE LOWER(id) = LOWER($1));", data.RoomID).Scan(&roomExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if !roomExists {
		return fmt.Errorf("Room not found")
	}

	banExists := false
	if err = h.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM bans WHERE LOWER(user_id) = LOWER($1));", uid).Scan(&banExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if banExists {
		return fmt.Errorf("You are banned from this room")
	}

	var private bool
	var author_id string
	if err = h.DB.QueryRow(ctx, "SELECT private,author_id FROM rooms WHERE id = $1", data.RoomID).Scan(&private, &author_id); err != nil {
		return fmt.Errorf("Internal error")
	}

	if private && author_id != uid {
		membershipExists := false
		if err = h.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM members WHERE LOWER(user_id) = LOWER($1));", uid).Scan(&membershipExists); err != nil {
			return fmt.Errorf("Internal error")
		}
		if !membershipExists {
			return fmt.Errorf("You are not a member of this room")
		}
	}

	var mainChannelId, mainChannelName string
	if err = h.DB.QueryRow(ctx, "SELECT id,name FROM room_channels WHERE room_id = $1 AND main = TRUE;", data.RoomID).Scan(&mainChannelId, &mainChannelName); err != nil {
		if err != pgx.ErrNoRows {
			return fmt.Errorf("Internal error")
		} else {
			return fmt.Errorf("Main channel could not be found")
		}
	}

	h.SocketServer.JoinSubscriptionByWs <- socketServer.RegisterUnregisterSubsConnWs{
		Conn:    c,
		SubName: fmt.Sprintf("channel:%v", mainChannelId),
	}

	return nil
}

func leaveRoom(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketvalidation.JoinLeaveRoomData{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var mainChannelId, mainChannelName string
	if err = h.DB.QueryRow(ctx, "SELECT id,name FROM room_channels WHERE room_id = $1 AND main = TRUE;", data.RoomID).Scan(&mainChannelId, &mainChannelName); err != nil {
		if err != pgx.ErrNoRows {
			return fmt.Errorf("Internal error")
		} else {
			return fmt.Errorf("Main channel could not be found")
		}
	}

	recvChan := make(chan map[string]struct{})
	h.SocketServer.GetConnectionSubscriptions <- socketServer.GetConnectionSubscriptions{
		RecvChan: recvChan,
		Conn:     c,
	}
	subs := <-recvChan

	for sub := range subs {
		if strings.HasPrefix(sub, "channel:") {
			h.SocketServer.LeaveSubscriptionByWs <- socketServer.RegisterUnregisterSubsConnWs{
				SubName: sub,
				Conn:    c,
			}
		}
	}

	return nil
}
