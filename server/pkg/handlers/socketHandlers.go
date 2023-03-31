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

	case "DIRECT_MESSAGE":
		err = directMessage(data, h, uid, c)
	case "DIRECT_MESSAGE_UPDATE":
		err = directMessageUpdate(data, h, uid, c)
	case "DIRECT_MESSAGE_DELETE":
		err = directMessageDelete(data, h, uid, c)

	case "FRIEND_REQUEST":
		err = friendRequest(data, h, uid, c)
	case "FRIEND_REQUEST_RESPONSE":
		err = friendRequestResponse(data, h, uid, c)

	case "START_WATCHING":
		err = startWatching(data, h, uid, c)
	case "STOP_WATCHING":
		err = stopWatching(data, h, uid, c)
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
	if err = conn.QueryRow(ctx, selectRoomExistsStmt.Name, data.RoomID).Scan(&roomExists); err != nil {
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
	if err = conn.QueryRow(ctx, selectRoomStmt.Name, data.RoomID).Scan(&private, &author_id); err != nil {
		return fmt.Errorf("Internal error")
	}

	if private && author_id != uid {
		membershipExistsStmt, err := conn.Conn().Prepare(ctx, "join_room_select_room_membership_stmt", "SELECT EXISTS(SELECT 1 FROM members WHERE LOWER(user_id) = LOWER($1))")
		if err != nil {
			return fmt.Errorf("Internal error")
		}

		membershipExists := false
		if err = conn.QueryRow(ctx, membershipExistsStmt.Name, uid).Scan(&membershipExists); err != nil {
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
	if err = conn.QueryRow(ctx, selectChannelStmt.Name, data.RoomID).Scan(&mainChannelId, &mainChannelName); err != nil {
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
	if err = conn.QueryRow(ctx, selectChannelStmt.Name, data.RoomID).Scan(&mainChannelId, &mainChannelName); err != nil {
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
	if err = conn.QueryRow(ctx, selectChannelStmt.Name, data.ChannelID).Scan(&room_id); err != nil {
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
	if err := conn.QueryRow(ctx, insertStmt.Name, content, uid, data.ChannelID).Scan(&id); err != nil {
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
		log.Println("ERR A:", err, err)
		return fmt.Errorf("Internal error")
	}

	content := strings.TrimSpace(data.Content)

	if _, err := conn.Exec(ctx, stmt.Name, content, uid, data.MsgID); err != nil {
		if err != pgx.ErrNoRows {
			log.Println("ERR B:", err, err)
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
		log.Println("ERR A:", err, err)
		return fmt.Errorf("Internal error")
	}

	if _, err = conn.Exec(ctx, stmt.Name, uid, data.MsgID); err != nil {
		if err != pgx.ErrNoRows {
			log.Println("ERR B:", err, err)
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

func directMessage(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketvalidation.DirectMessage{}
	if err := UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	var blocked bool
	selectBlockedStmt := "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocked_id = $1 AND blocker_id = $2)"
	if err := conn.QueryRow(ctx, selectBlockedStmt, data.Uid, uid).Scan(&blocked); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blocked {
		return fmt.Errorf("This user has blocked your account")
	}

	var blocker bool
	selectBlockerStmt := "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocker_id = $1 AND blocked_id = $2)"
	if err := conn.QueryRow(ctx, selectBlockerStmt, uid, data.Uid).Scan(&blocker); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blocker {
		return fmt.Errorf("You have blocked this user, you must unblock them to message them")
	}

	createMsgStmt := "INSERT INTO direct_messages (content, author_id, recipient_id) VALUES ($1, $2, $3) RETURNING id"
	var id string
	content := strings.TrimSpace(data.Content)
	if err := conn.QueryRow(ctx, createMsgStmt, content, uid, data.Uid).Scan(&id); err != nil {
		return fmt.Errorf("Internal error")
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids: []string{uid, data.Uid},
		Data: socketmessages.DirectMessage{
			ID:        id,
			Content:   content,
			CreatedAt: time.Now().Format(time.RFC3339),
			AuthorID:  uid,
		},
		MessageType: "DIRECT_MESSAGE",
	}

	return nil
}

func directMessageUpdate(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketvalidation.DirectMessageUpdate{}
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

	selectMsgStmt, err := conn.Conn().Prepare(ctx, "direct_message_update_select_stmt", "SELECT recipient_id FROM direct_messages WHERE author_id = $1 AND id = $2")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var recipient_id string
	if err = h.DB.QueryRow(ctx, selectMsgStmt.Name, uid, data.MsgID).Scan(&recipient_id); err != nil {
		return fmt.Errorf("Internal error")
	}

	updateMsgStmt, err := conn.Conn().Prepare(ctx, "direct_message_update_stmt", "UPDATE direct_messages SET content = $1 WHERE id = $2")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	content := strings.TrimSpace(data.Content)

	if _, err = h.DB.Exec(ctx, updateMsgStmt.Name, content, data.MsgID); err != nil {
		return fmt.Errorf("Internal error")
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids: []string{uid, recipient_id},
		Data: socketmessages.DirectMessageUpdate{
			ID:      data.MsgID,
			Content: content,
		},
		MessageType: "DIRECT_MESSAGE_UPDATE",
	}

	return nil
}

func directMessageDelete(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketvalidation.DirectMessageDelete{}
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

	selectMsgStmt, err := conn.Conn().Prepare(ctx, "direct_message_delete_select_stmt", "SELECT recipient_id FROM direct_messages WHERE author_id = $1 AND id = $2")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var recipient_id string
	if err = h.DB.QueryRow(ctx, selectMsgStmt.Name, uid, data.MsgID).Scan(&recipient_id); err != nil {
		return fmt.Errorf("Internal error")
	}

	deleteMsgStmt, err := conn.Conn().Prepare(ctx, "direct_message_delete_stmt", "DELETE FROM direct_messages WHERE id = $1")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	if _, err = h.DB.Exec(ctx, deleteMsgStmt.Name, data.MsgID); err != nil {
		return fmt.Errorf("Internal error")
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids: []string{uid, recipient_id},
		Data: socketmessages.DirectMessageUpdate{
			ID: data.MsgID,
		},
		MessageType: "DIRECT_MESSAGE_DELETE",
	}

	return nil
}

func friendRequest(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketvalidation.FriendRequest{}
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

	selectBlockedStmt, err := conn.Conn().Prepare(ctx, "friend_request_select_blocked_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocked_id = $1 AND blocker_id = $2)")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var blockedExists bool
	if err = h.DB.QueryRow(ctx, selectBlockedStmt.Name, uid, data.Uid).Scan(&blockedExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blockedExists {
		return fmt.Errorf("This user has blocked your account")
	}

	selectBlockerStmt, err := conn.Conn().Prepare(ctx, "friend_request_select_blocker_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocker_id = $1 AND blocked_id = $2)")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var blockerExists bool
	if err = h.DB.QueryRow(ctx, selectBlockerStmt.Name, uid, data.Uid).Scan(&blockerExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blockedExists {
		return fmt.Errorf("You have blocked this user, you must unblock them first")
	}

	insertFriendRequestStmt, err := conn.Conn().Prepare(ctx, "friend_request_insert_stmt", "INSERT INTO friend_requests (friender,friended) VALUES($1, $2)")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	if _, err = h.DB.Exec(ctx, insertFriendRequestStmt.Name, uid, data.Uid); err != nil {
		return fmt.Errorf("Internal error")
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids: []string{uid, data.Uid},
		Data: socketmessages.FriendRequest{
			Friender: uid,
			Friended: data.Uid,
		},
		MessageType: "FRIEND_REQUEST",
	}

	return nil
}

func friendRequestResponse(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketvalidation.FriendRequestResponse{}
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

	selectExistsStmt, err := conn.Conn().Prepare(ctx, "friend_request_response_select_stmt", "SELECT EXISTS(SELECT 1 FROM friend_requests WHERE friender = $1 AND friended = $2)")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var friendRequestExists bool
	if err = h.DB.QueryRow(ctx, selectExistsStmt.Name, data.Friender, uid).Scan(&friendRequestExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if !friendRequestExists {
		return fmt.Errorf("This user did not send you a friend request")
	}

	deleteStmt, err := conn.Conn().Prepare(ctx, "friend_request_response_delete_stmt", "DELETE FROM friend_requests WHERE friender = $1 AND friended = $2")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	if _, err = h.DB.Exec(ctx, deleteStmt.Name, data.Friender, uid); err != nil {
		return fmt.Errorf("Internal error")
	}

	if data.Accepted {
		insertStmt, err := conn.Conn().Prepare(ctx, "friend_request_response_insert_stmt", "INSERT INTO friends (friender,friended) VALUES($1, $2)")
		if err != nil {
			return fmt.Errorf("Internal error")
		}
		if _, err = h.DB.Exec(ctx, insertStmt.Name, data.Friender, uid); err != nil {
			return fmt.Errorf("Internal error")
		}
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids:        []string{uid, data.Friender},
		MessageType: "FRIEND_REQUEST_RESPONSE",
		Data: socketmessages.FriendRequestResponse{
			Accepted: data.Accepted,
			Friended: uid,
		},
	}

	return nil
}

func startWatching(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketvalidation.StartStopWatching{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	subName := ""

	switch data.Entity {
	case "ROOM":
		subName = fmt.Sprintf("room:%v", data.ID)
	case "USER":
		subName = fmt.Sprintf("user:%v", data.ID)
	case "BIO":
		subName = fmt.Sprintf("bio:%v", data.ID)
	default:
		return fmt.Errorf("Unrecognized entity")
	}

	h.SocketServer.JoinSubscriptionByWs <- socketServer.RegisterUnregisterSubsConnWs{
		Conn:    c,
		SubName: subName,
	}

	return nil
}

func stopWatching(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketvalidation.StartStopWatching{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	subName := ""

	switch data.Entity {
	case "ROOM":
		subName = fmt.Sprintf("room:%v", data.ID)
	case "USER":
		subName = fmt.Sprintf("user:%v", data.ID)
	case "BIO":
		subName = fmt.Sprintf("bio:%v", data.ID)
	default:
		return fmt.Errorf("Unrecognized entity")
	}

	h.SocketServer.LeaveSubscriptionByWs <- socketServer.RegisterUnregisterSubsConnWs{
		Conn:    c,
		SubName: subName,
	}

	return nil
}
