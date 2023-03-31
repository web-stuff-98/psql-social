package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	socketmessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
	socketvalidation "github.com/web-stuff-98/psql-social/pkg/socketValidation"
)

func handleSocketEvent(data map[string]interface{}, event string, h handler, uid string, c *websocket.Conn) error {
	var err error

	switch event {
	case "JOIN_ROOM":
		err = joinRoom(data, h, uid, c)
	case "LEAVE_ROOM":
		err = leaveRoom(data, h, uid, c)
	case "ROOM_MESSAGE":
		err = roomMessage(data, h, uid, c)
	case "ROOM_MESSAGE_UPDATE":
		err = roomMessageUpdate(data, h, uid, c)
	case "ROOM_MESSAGE_DELETE":
		err = roomMessageDelete(data, h, uid, c)
	default:
		return fmt.Errorf("Unrecognized event type")
	}

	return err
}

func UnmarshalMap(m map[string]interface{}, s interface{}) error {
	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("Bad request")
	}
	err = json.Unmarshal(b, s)
	if err != nil {
		return fmt.Errorf("Bad request")
	}
	v := validator.New()
	if err := v.Struct(s); err != nil {
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

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectRoomExistsStmt, err := conn.Conn().Prepare(ctx, "join_room_select_room_exists_stmt", "SELECT EXISTS(SELECT 1 FROM rooms WHERE id = $1)")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	roomExists := false
	if err = h.DB.QueryRow(ctx, selectRoomExistsStmt.Name, data.RoomID).Scan(&roomExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if !roomExists {
		return fmt.Errorf("Room not found")
	}

	banExists := false
	if err = h.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM bans WHERE user_id = $1);", uid).Scan(&banExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if banExists {
		return fmt.Errorf("You are banned from this room")
	}

	selectRoomStmt, err := conn.Conn().Prepare(ctx, "join_room_select_room_stmt", "SELECT private,author_id FROM rooms WHERE id = $1")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var private bool
	var author_id string
	if err = h.DB.QueryRow(ctx, selectRoomStmt.Name, data.RoomID).Scan(&private, &author_id); err != nil {
		return fmt.Errorf("Internal error")
	}

	if private && author_id != uid {
		membershipExistsStmt, err := conn.Conn().Prepare(ctx, "join_room_select_room_membership_stmt", "SELECT EXISTS(SELECT 1 FROM members WHERE LOWER(user_id) = LOWER($1))")
		if err != nil {
			return fmt.Errorf("Internal error")
		}

		membershipExists := false
		if err = h.DB.QueryRow(ctx, membershipExistsStmt.Name, uid).Scan(&membershipExists); err != nil {
			return fmt.Errorf("Internal error")
		}
		if !membershipExists {
			return fmt.Errorf("You are not a member of this room")
		}
	}

	selectChannelStmt, err := conn.Conn().Prepare(ctx, "join_room_select_channel_stmt", "SELECT id,name FROM room_channels WHERE room_id = $1 AND main = TRUE")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var mainChannelId, mainChannelName string
	if err = h.DB.QueryRow(ctx, selectChannelStmt.Name, data.RoomID).Scan(&mainChannelId, &mainChannelName); err != nil {
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

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectChannelStmt, err := conn.Conn().Prepare(ctx, "leave_room_select_channel_stmt", "SELECT id,name FROM room_channels WHERE room_id = $1 AND main = TRUE")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var mainChannelId, mainChannelName string
	if err = h.DB.QueryRow(ctx, selectChannelStmt.Name, data.RoomID).Scan(&mainChannelId, &mainChannelName); err != nil {
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

func roomMessage(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketvalidation.RoomMessage{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectChannelStmt, err := conn.Conn().Prepare(ctx, "room_message_select_room_channel_stmt", "SELECT room_id FROM room_channels WHERE id = $1")
	if err != nil {
		if err != pgx.ErrNoRows {
			return fmt.Errorf("Internal error")
		}
		return fmt.Errorf("Channel not found")
	}

	var room_id string
	if err = h.DB.QueryRow(ctx, selectChannelStmt.Name, data.ChannelID).Scan(&room_id); err != nil {
		if err != pgx.ErrNoRows {
			return fmt.Errorf("Internal error")
		}
		return fmt.Errorf("Room not found")
	}

	banExists := false
	if err = h.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM bans WHERE user_id = $1 AND room_id = $2);", uid, room_id).Scan(&banExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if banExists {
		return fmt.Errorf("You are banned from this room")
	}

	var private bool
	var author_id string
	if err = h.DB.QueryRow(ctx, "SELECT private,author_id FROM rooms WHERE id = $1;", room_id).Scan(&private, &author_id); err != nil {
		return fmt.Errorf("Internal error")
	}

	if private && author_id != uid {
		var membershipExists bool
		if err = h.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM members WHERE user_id = $1 AND room_id = $2);", uid, room_id).Scan(&membershipExists); err != nil {
			return fmt.Errorf("Internal error")
		}
		if !membershipExists {
			return fmt.Errorf("You are not a member of this room")
		}
	}

	insertStmt, err := conn.Conn().Prepare(ctx, "insert_room_message_stmt", "INSERT INTO room_messages (content,author_id,room_channel_id) VALUES($1, $2, $3) RETURNING id")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	content := strings.TrimSpace(data.Content)

	var id string
	if err := h.DB.QueryRow(ctx, insertStmt.Name, content, uid, data.ChannelID).Scan(&id); err != nil {
		return fmt.Errorf("Internal error")
	}

	h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
		SubName: fmt.Sprintf("channel:%v", data.ChannelID),
		Data: socketmessages.RoomMessage{
			ID:        id,
			Content:   content,
			CreatedAt: time.Now().Format(time.RFC3339),
			AuthorID:  uid,
		},
		MessageType: "ROOM_MESSAGE",
	}

	return nil
}

func roomMessageUpdate(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketvalidation.RoomMessageUpdate{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	stmt, err := conn.Conn().Prepare(ctx, "room_message_update_stmt", "UPDATE room_messages SET content = $1 WHERE author_id = $2 AND id = $3")
	if err != nil {
		log.Println("ERR A:", err)
		return fmt.Errorf("Internal error")
	}

	content := strings.TrimSpace(data.Content)

	if h.DB.QueryRow(ctx, stmt.Name, content, uid, data.MsgID); err != nil {
		if err != pgx.ErrNoRows {
			log.Println("ERR B:", err)
			return fmt.Errorf("Internal error")
		} else {
			return fmt.Errorf("Message not found")
		}
	}

	recvChan := make(chan map[string]struct{})
	h.SocketServer.GetConnectionSubscriptions <- socketServer.GetConnectionSubscriptions{
		RecvChan: recvChan,
		Conn:     c,
	}
	subs := <-recvChan
	channelName := ""
	for k := range subs {
		if strings.HasPrefix(k, "channel:") {
			channelName = k
			break
		}
	}

	h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
		MessageType: "ROOM_MESSAGE_UPDATE",
		Data: socketmessages.RoomMessageUpdate{
			ID:      data.MsgID,
			Content: content,
		},
		SubName: channelName,
	}

	return nil
}

func roomMessageDelete(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketvalidation.RoomMessageDelete{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	stmt, err := conn.Conn().Prepare(ctx, "room_message_delete_stmt", "DELETE FROM room_messages WHERE author_id = $1 AND id = $2")
	if err != nil {
		log.Println("ERR A:", err)
		return fmt.Errorf("Internal error")
	}

	if h.DB.QueryRow(ctx, stmt.Name, uid, data.MsgID); err != nil {
		if err != pgx.ErrNoRows {
			log.Println("ERR B:", err)
			return fmt.Errorf("Internal error")
		} else {
			return fmt.Errorf("Message not found")
		}
	}

	recvChan := make(chan map[string]struct{})
	h.SocketServer.GetConnectionSubscriptions <- socketServer.GetConnectionSubscriptions{
		RecvChan: recvChan,
		Conn:     c,
	}
	subs := <-recvChan
	channelName := ""
	for k := range subs {
		if strings.HasPrefix(k, "channel:") {
			channelName = k
			break
		}
	}

	h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
		MessageType: "ROOM_MESSAGE_DELETE",
		Data: socketmessages.RoomMessageDelete{
			ID: data.MsgID,
		},
		SubName: channelName,
	}

	return nil
}
