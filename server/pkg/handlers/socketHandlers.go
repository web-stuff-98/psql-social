package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/websocket/v2"
	"github.com/jackc/pgx/v5"
	callServer "github.com/web-stuff-98/psql-social/pkg/callServer"
	"github.com/web-stuff-98/psql-social/pkg/channelRTCserver"
	socketLimiter "github.com/web-stuff-98/psql-social/pkg/socketLimiter"
	socketMessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
	socketValidation "github.com/web-stuff-98/psql-social/pkg/socketValidation"
)

// This could maybe do with some code splitting, but I can't be asked

func handleSocketEvent(data map[string]interface{}, event string, h handler, uid string, c *websocket.Conn) error {
	var err error

	recvChan := make(chan error, 1)
	h.SocketLimiter.SocketEvent <- socketLimiter.SocketEvent{
		RecvChan: recvChan,
		Type:     event,
		Conn:     c,
	}
	err = <-recvChan

	close(recvChan)

	if err != nil {
		return err
	}

	switch event {
	case "JOIN_ROOM":
		err = joinRoom(data, h, uid, c)
	case "LEAVE_ROOM":
		err = leaveRoom(data, h, uid, c)
	case "JOIN_CHANNEL":
		err = joinChannel(data, h, uid, c)
	case "LEAVE_CHANNEL":
		err = leaveChannel(data, h, uid, c)

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

	case "INVITATION":
		err = invitation(data, h, uid, c)
	case "INVITATION_RESPONSE":
		err = invitationResponse(data, h, uid, c)

	case "START_WATCHING":
		err = startWatching(data, h, uid, c)
	case "STOP_WATCHING":
		err = stopWatching(data, h, uid, c)

	case "BLOCK":
		err = block(data, h, uid, c)
	case "UNBLOCK":
		err = unblock(data, h, uid, c)

	case "BAN":
		err = ban(data, h, uid, c)
	case "UNBAN":
		err = unban(data, h, uid, c)

	case "CALL_USER":
		err = callUser(data, h, uid, c)
	case "CALL_USER_RESPONSE":
		err = callUserResponse(data, h, uid, c)
	case "CALL_LEAVE":
		err = callLeave(data, h, uid, c)
	case "CALL_WEBRTC_OFFER":
		err = callOffer(data, h, uid, c)
	case "CALL_WEBRTC_ANSWER":
		err = callAnswer(data, h, uid, c)
	case "CALL_WEBRTC_RECIPIENT_REQUEST_REINITIALIZATION":
		err = callRequestReinitialization(data, h, uid, c)
	case "CALL_UPDATE_MEDIA_OPTIONS":
		err = callUpdateMediaOptions(data, h, uid, c)

	case "CHANNEL_WEBRTC_JOIN":
		err = channelWebRTCJoin(data, h, uid, c)
	case "CHANNEL_WEBRTC_LEAVE":
		err = channelWebRTCLeave(data, h, uid, c)
	case "CHANNEL_WEBRTC_SENDING_SIGNAL":
		err = channelWebRTCSendingSignal(data, h, uid, c)
	case "CHANNEL_WEBRTC_RETURNING_SIGNAL":
		err = channelWebRTCReturningSignal(data, h, uid, c)
	case "CHANNEL_WEBRTC_UPDATE_MEDIA_OPTIONS":
		err = channelWebRTCUpdateMediaOptions(data, h, uid, c)

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
	data := &socketValidation.JoinLeaveRoomData{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectRoomExistsStmt, err := conn.Conn().Prepare(ctx, "join_room_select_room_exists_stmt", "SELECT EXISTS(SELECT 1 FROM rooms WHERE id = $1);")
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

	selectRoomStmt, err := conn.Conn().Prepare(ctx, "join_room_select_room_stmt", "SELECT private,author_id FROM rooms WHERE id = $1;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var private bool
	var author_id string
	if err = conn.QueryRow(ctx, selectRoomStmt.Name, data.RoomID).Scan(&private, &author_id); err != nil {
		return fmt.Errorf("Internal error")
	}

	if private && author_id != uid {
		membershipExistsStmt, err := conn.Conn().Prepare(ctx, "join_room_select_room_membership_stmt", "SELECT EXISTS(SELECT 1 FROM members WHERE user_id = $1);")
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

	selectChannelStmt, err := conn.Conn().Prepare(ctx, "join_room_select_channel_stmt", "SELECT id,name FROM room_channels WHERE room_id = $1 AND main = TRUE;")
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
	data := &socketValidation.JoinLeaveRoomData{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectChannelsStmt, err := conn.Conn().Prepare(ctx, "leave_room_select_channels_stmt", "SELECT id FROM room_channels WHERE room_id = $1;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	rows, err := conn.Query(ctx, selectChannelsStmt.Name, data.RoomID)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer rows.Close()

	channelIds := make(map[string]struct{})

	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return fmt.Errorf("Internal error")
		}
		channelIds[id] = struct{}{}
	}

	recvChan := make(chan map[string]struct{}, 1)
	h.SocketServer.GetConnectionSubscriptions <- socketServer.GetConnectionSubscriptions{
		RecvChan: recvChan,
		Conn:     c,
	}
	subs := <-recvChan

	close(recvChan)

	for sub := range subs {
		_, ok := channelIds[sub]
		if strings.HasPrefix(sub, "channel:") && ok {
			h.SocketServer.LeaveSubscriptionByWs <- socketServer.RegisterUnregisterSubsConnWs{
				SubName: sub,
				Conn:    c,
			}
		}
	}

	return nil
}

func joinChannel(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.JoinLeaveChannel{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectChannelStmt, err := conn.Conn().Prepare(ctx, "join_channel_select_channel_stmt", "SELECT room_id FROM room_channels WHERE id = $1;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var room_id string
	if err = conn.Conn().QueryRow(ctx, selectChannelStmt.Name, data.ChannelID).Scan(&room_id); err != nil {
		if err != pgx.ErrNoRows {
			return fmt.Errorf("Internal error")
		}
		return fmt.Errorf("Channel not found")
	}

	var banExists bool
	if err = h.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM bans WHERE user_id = $1 AND room_id = $2);", uid, room_id).Scan(&banExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if banExists {
		return fmt.Errorf("You are banned from this room")
	}

	h.SocketServer.JoinSubscriptionByWs <- socketServer.RegisterUnregisterSubsConnWs{
		SubName: fmt.Sprintf("channel:%v", data.ChannelID),
		Conn:    c,
	}

	return nil
}

func leaveChannel(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.JoinLeaveChannel{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	h.SocketServer.LeaveSubscriptionByWs <- socketServer.RegisterUnregisterSubsConnWs{
		SubName: fmt.Sprintf("channel:%v", data.ChannelID),
		Conn:    c,
	}

	return nil
}

func roomMessage(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.RoomMessage{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectChannelStmt, err := conn.Conn().Prepare(ctx, "room_message_select_room_channel_stmt", "SELECT room_id FROM room_channels WHERE id = $1;")
	if err != nil {
		return fmt.Errorf("Internal error")
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

	insertStmt, err := conn.Conn().Prepare(ctx, "insert_room_message_stmt", "INSERT INTO room_messages (content, author_id, room_channel_id, has_attachment) VALUES($1, $2, $3, $4) RETURNING id;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	content := strings.TrimSpace(data.Content)

	var id string
	if err := conn.QueryRow(ctx, insertStmt.Name, content, uid, data.ChannelID, data.HasAttachment).Scan(&id); err != nil {
		return fmt.Errorf("Internal error")
	}

	subName := fmt.Sprintf("channel:%v", data.ChannelID)

	// get uids of users in the channel, needed for excluding users already in the channel from notifications
	recvChan := make(chan map[string]struct{})
	h.SocketServer.GetSubscriptionUids <- socketServer.GetSubscriptionUids{
		SubName:  subName,
		RecvChan: recvChan,
	}
	uidsMap := <-recvChan
	var receiveNotifications []string
	if rows, err := h.DB.Query(ctx, "SELECT user_id FROM members WHERE room_id = $1;", room_id); err != nil {
		if err != pgx.ErrNoRows {
			return fmt.Errorf("Internal error")
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var uid string
			if err = rows.Scan(&uid); err != nil {
				return fmt.Errorf("Internal error")
			}
			if _, ok := uidsMap[uid]; !ok {
				receiveNotifications = append(receiveNotifications, uid)
			}
		}
	}
	var owner_id string
	if err = h.DB.QueryRow(ctx, "SELECT author_id FROM rooms WHERE id = $1;", room_id).Scan(&owner_id); err != nil {
		return fmt.Errorf("Internal error")
	} else {
		if _, ok := uidsMap[owner_id]; !ok {
			receiveNotifications = append(receiveNotifications, owner_id)
		}
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids: receiveNotifications,
		Data: socketMessages.RoomMessageNotify{
			RoomID:    room_id,
			ChannelID: data.ChannelID,
		},
		MessageType: "ROOM_MESSAGE_NOTIFY",
	}

	h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
		SubName: subName,
		Data: socketMessages.RoomMessage{
			ID:            id,
			Content:       content,
			CreatedAt:     time.Now().Format(time.RFC3339),
			AuthorID:      uid,
			HasAttachment: data.HasAttachment,
		},
		MessageType: "ROOM_MESSAGE",
	}

	if data.HasAttachment {
		h.SocketServer.SendDataToUser <- socketServer.UserMessageData{
			Uid: uid,
			Data: socketMessages.RequestAttachment{
				ID: id,
			},
			MessageType: "REQUEST_ATTACHMENT",
		}
	}

	return nil
}

func roomMessageUpdate(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.RoomMessageUpdate{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	stmt, err := conn.Conn().Prepare(ctx, "room_message_update_stmt", "UPDATE room_messages SET content = $1 WHERE author_id = $2 AND id = $3;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	content := strings.TrimSpace(data.Content)

	if _, err := conn.Exec(ctx, stmt.Name, content, uid, data.MsgID); err != nil {
		if err != pgx.ErrNoRows {
			return fmt.Errorf("Internal error")
		} else {
			return fmt.Errorf("Message not found")
		}
	}

	recvChan := make(chan map[string]struct{}, 1)
	h.SocketServer.GetConnectionSubscriptions <- socketServer.GetConnectionSubscriptions{
		RecvChan: recvChan,
		Conn:     c,
	}
	subs := <-recvChan

	close(recvChan)

	channelName := ""
	for k := range subs {
		if strings.HasPrefix(k, "channel:") {
			channelName = k
			break
		}
	}

	h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
		MessageType: "ROOM_MESSAGE_UPDATE",
		Data: socketMessages.RoomMessageUpdate{
			ID:      data.MsgID,
			Content: content,
		},
		SubName: channelName,
	}

	return nil
}

func roomMessageDelete(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.RoomMessageDelete{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	stmt, err := conn.Conn().Prepare(ctx, "room_message_delete_stmt", "DELETE FROM room_messages WHERE author_id = $1 AND id = $2;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	if _, err = conn.Exec(ctx, stmt.Name, uid, data.MsgID); err != nil {
		if err != pgx.ErrNoRows {
			return fmt.Errorf("Internal error")
		} else {
			return fmt.Errorf("Message not found")
		}
	}

	recvChan := make(chan map[string]struct{}, 1)
	h.SocketServer.GetConnectionSubscriptions <- socketServer.GetConnectionSubscriptions{
		RecvChan: recvChan,
		Conn:     c,
	}
	subs := <-recvChan

	close(recvChan)

	channelName := ""
	for k := range subs {
		if strings.HasPrefix(k, "channel:") {
			channelName = k
			break
		}
	}

	h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
		MessageType: "ROOM_MESSAGE_DELETE",
		Data: socketMessages.RoomMessageDelete{
			ID: data.MsgID,
		},
		SubName: channelName,
	}

	return nil
}

func directMessage(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.DirectMessage{}
	if err := UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	var blocker bool
	selectBlockerStmt := "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocker = $1 AND blocked = $2);"
	if err := conn.QueryRow(ctx, selectBlockerStmt, uid, data.Uid).Scan(&blocker); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blocker {
		return fmt.Errorf("You have blocked this user, you must unblock them to message them")
	}

	var blocked bool
	selectBlockedStmt := "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocked = $1 AND blocker = $2);"
	if err := conn.QueryRow(ctx, selectBlockedStmt, uid, data.Uid).Scan(&blocked); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blocked {
		return fmt.Errorf("This user has blocked your account")
	}

	var id string
	content := strings.TrimSpace(data.Content)
	if err := conn.QueryRow(ctx, "INSERT INTO direct_messages (content, author_id, recipient_id, has_attachment) VALUES ($1, $2, $3, $4) RETURNING id;", content, uid, data.Uid, data.HasAttachment).Scan(&id); err != nil {
		return fmt.Errorf("Internal error")
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids: []string{uid, data.Uid},
		Data: socketMessages.DirectMessage{
			ID:            id,
			Content:       content,
			CreatedAt:     time.Now().Format(time.RFC3339),
			AuthorID:      uid,
			RecipientID:   data.Uid,
			HasAttachment: data.HasAttachment,
		},
		MessageType: "DIRECT_MESSAGE",
	}

	if data.HasAttachment {
		h.SocketServer.SendDataToUser <- socketServer.UserMessageData{
			Uid: uid,
			Data: socketMessages.RequestAttachment{
				ID: id,
			},
			MessageType: "REQUEST_ATTACHMENT",
		}
	}

	return nil
}

func directMessageUpdate(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.DirectMessageUpdate{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectMsgStmt, err := conn.Conn().Prepare(ctx, "direct_message_update_select_stmt", "SELECT recipient_id FROM direct_messages WHERE author_id = $1 AND id = $2;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var recipient_id string
	if err = conn.QueryRow(ctx, selectMsgStmt.Name, uid, data.MsgID).Scan(&recipient_id); err != nil {
		return fmt.Errorf("Internal error")
	}

	updateMsgStmt, err := conn.Conn().Prepare(ctx, "direct_message_update_stmt", "UPDATE direct_messages SET content = $1 WHERE id = $2;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	content := strings.TrimSpace(data.Content)

	if _, err = conn.Exec(ctx, updateMsgStmt.Name, content, data.MsgID); err != nil {
		return fmt.Errorf("Internal error")
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids: []string{uid, recipient_id},
		Data: socketMessages.DirectMessageUpdate{
			ID:          data.MsgID,
			Content:     content,
			AuthorID:    uid,
			RecipientID: recipient_id,
		},
		MessageType: "DIRECT_MESSAGE_UPDATE",
	}

	return nil
}

func directMessageDelete(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.DirectMessageDelete{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectMsgStmt, err := conn.Conn().Prepare(ctx, "direct_message_delete_select_stmt", "SELECT recipient_id FROM direct_messages WHERE author_id = $1 AND id = $2;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var recipient_id string
	if err = conn.QueryRow(ctx, selectMsgStmt.Name, uid, data.MsgID).Scan(&recipient_id); err != nil {
		return fmt.Errorf("Internal error")
	}

	deleteMsgStmt, err := conn.Conn().Prepare(ctx, "direct_message_delete_stmt", "DELETE FROM direct_messages WHERE id = $1;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	if _, err = conn.Exec(ctx, deleteMsgStmt.Name, data.MsgID); err != nil {
		return fmt.Errorf("Internal error")
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids: []string{uid, recipient_id},
		Data: socketMessages.DirectMessageDelete{
			ID:          data.MsgID,
			AuthorID:    uid,
			RecipientID: recipient_id,
		},
		MessageType: "DIRECT_MESSAGE_DELETE",
	}

	return nil
}

func friendRequest(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.FriendRequest{}
	if err := UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectFriendRequestExistsStmt, err := conn.Conn().Prepare(ctx, "friend_request_select_friends_stmt", "SELECT EXISTS(SELECT 1 FROM friend_requests WHERE (friender = $1 AND friended = $2) OR (friender = $2 AND friended = $1));")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var friendRequestExists bool
	if err = conn.QueryRow(ctx, selectFriendRequestExistsStmt.Name, uid, data.Uid).Scan(&friendRequestExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if friendRequestExists {
		return fmt.Errorf("You have already sent or received a friend request from this user")
	}

	selectBlockedStmt, err := conn.Conn().Prepare(ctx, "friend_request_select_blocked_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocked = $1 AND blocker = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var blockedExists bool
	if err = conn.QueryRow(ctx, selectBlockedStmt.Name, data.Uid, uid).Scan(&blockedExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blockedExists {
		return fmt.Errorf("This user has blocked your account")
	}

	selectBlockerStmt, err := conn.Conn().Prepare(ctx, "friend_request_select_blocker_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocker = $1 AND blocked = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var blockerExists bool
	if err = conn.QueryRow(ctx, selectBlockerStmt.Name, uid, data.Uid).Scan(&blockerExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blockerExists {
		return fmt.Errorf("You have blocked this user, you must unblock them first")
	}

	selectFriendsExistsStmt, err := conn.Conn().Prepare(ctx, "friend_request_select_friends_exists_stmt", "SELECT EXISTS(SELECT 1 FROM friends WHERE (friender = $1 AND friended = $2) OR (friender = $2 AND friended = $1));")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var friendsExists bool
	if err = conn.QueryRow(ctx, selectFriendsExistsStmt.Name, uid, data.Uid).Scan(&friendsExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if friendRequestExists {
		return fmt.Errorf("You are already friends with this user")
	}

	insertFriendRequestStmt, err := conn.Conn().Prepare(ctx, "friend_request_insert_stmt", "INSERT INTO friend_requests (friender,friended) VALUES($1, $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	if _, err = conn.Exec(ctx, insertFriendRequestStmt.Name, uid, data.Uid); err != nil {
		return fmt.Errorf("Internal error")
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids: []string{uid, data.Uid},
		Data: socketMessages.FriendRequest{
			Friender:  uid,
			Friended:  data.Uid,
			CreatedAt: time.Now().Format(time.RFC3339),
		},
		MessageType: "FRIEND_REQUEST",
	}

	return nil
}

func friendRequestResponse(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.FriendRequestResponse{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectExistsStmt, err := conn.Conn().Prepare(ctx, "friend_request_response_select_stmt", "SELECT EXISTS(SELECT 1 FROM friend_requests WHERE friender = $1 AND friended = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var friendRequestExists bool
	if err = conn.QueryRow(ctx, selectExistsStmt.Name, data.Friender, uid).Scan(&friendRequestExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if !friendRequestExists {
		return fmt.Errorf("This user did not send you a friend request")
	}

	selectFriendsExistsStmt, err := conn.Conn().Prepare(ctx, "friend_request_response_select_friends_stmt", "SELECT EXISTS(SELECT 1 FROM friends WHERE friender = $1 AND friended = $2 OR (friender = $2 AND friended = $1));")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var friendsExists bool
	if err = conn.QueryRow(ctx, selectFriendsExistsStmt.Name, data.Friender, uid).Scan(&friendsExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if friendsExists {
		return fmt.Errorf("You are already friends with this user")
	}

	deleteStmt, err := conn.Conn().Prepare(ctx, "friend_request_response_delete_stmt", "DELETE FROM friend_requests WHERE friender = $1 AND friended = $2;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	if _, err = conn.Exec(ctx, deleteStmt.Name, data.Friender, uid); err != nil {
		return fmt.Errorf("Internal error")
	}

	selectBlockedStmt, err := conn.Conn().Prepare(ctx, "friend_request_response_select_blocked_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocked = $1 AND blocker = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var blockedExists bool
	if err = conn.QueryRow(ctx, selectBlockedStmt.Name, uid, data.Friender).Scan(&blockedExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blockedExists {
		return fmt.Errorf("This user has blocked your account")
	}

	selectBlockerStmt, err := conn.Conn().Prepare(ctx, "friend_request_response_select_blocker_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocker = $1 AND blocked = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var blockerExists bool
	if err = conn.QueryRow(ctx, selectBlockerStmt.Name, uid, data.Friender).Scan(&blockerExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blockerExists {
		return fmt.Errorf("You have blocked this user, you must unblock them first")
	}

	if data.Accepted {
		insertStmt, err := conn.Conn().Prepare(ctx, "friend_request_response_insert_stmt", "INSERT INTO friends (friender,friended) VALUES($1, $2);")
		if err != nil {
			return fmt.Errorf("Internal error")
		}
		if _, err = conn.Exec(ctx, insertStmt.Name, data.Friender, uid); err != nil {
			return fmt.Errorf("Internal error")
		}
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids:        []string{uid, data.Friender},
		MessageType: "FRIEND_REQUEST_RESPONSE",
		Data: socketMessages.FriendRequestResponse{
			Accepted: data.Accepted,
			Friended: uid,
			Friender: data.Friender,
		},
	}

	return nil
}

func invitation(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.Invitation{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectInvitationExistsStmt, err := conn.Conn().Prepare(ctx, "invitation_select_invitation_stmt", "SELECT EXISTS(SELECT 1 FROM invitations WHERE inviter = $1 AND invited = $2 AND room_id = $3);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var invitationExists bool
	if err = conn.QueryRow(ctx, selectInvitationExistsStmt.Name, data.Uid, uid, data.RoomID).Scan(&invitationExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if invitationExists {
		return fmt.Errorf("You have already sent an invitation to this user")
	}

	selectBlockedStmt, err := conn.Conn().Prepare(ctx, "invitation_select_blocked_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocked = $1 AND blocker = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var blockedExists bool
	if err = conn.QueryRow(ctx, selectBlockedStmt.Name, uid, data.Uid).Scan(&blockedExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blockedExists {
		return fmt.Errorf("This user has blocked your account")
	}

	selectBlockerStmt, err := conn.Conn().Prepare(ctx, "invitation_select_blocker_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocker = $1 AND blocked = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var blockerExists bool
	if err = conn.QueryRow(ctx, selectBlockerStmt.Name, uid, data.Uid).Scan(&blockerExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blockerExists {
		return fmt.Errorf("You have blocked this user, you must unblock them first")
	}

	selectMemberExistsStmt, err := conn.Conn().Prepare(ctx, "invitation_select_member_stmt", "SELECT EXISTS(SELECT 1 FROM members WHERE user_id = $1 AND room_id = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var memberExists bool
	if err = conn.QueryRow(ctx, selectMemberExistsStmt.Name, data.Uid, data.RoomID).Scan(&memberExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blockerExists {
		return fmt.Errorf("This user is already a member of the room")
	}

	insertStmt, err := conn.Conn().Prepare(ctx, "invitation_insert_stmt", "INSERT INTO invitations (inviter, invited, room_id) VALUES($1, $2, $3);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	if _, err = conn.Exec(ctx, insertStmt.Name, uid, data.Uid, data.RoomID); err != nil {
		return fmt.Errorf("Internal error")
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids: []string{uid, data.Uid},
		Data: socketMessages.Invitation{
			Inviter:   uid,
			Invited:   data.Uid,
			RoomID:    data.RoomID,
			CreatedAt: time.Now().Format(time.RFC3339),
		},
		MessageType: "INVITATION",
	}

	return nil
}

func invitationResponse(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.InvitationResponse{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectExistsStmt, err := conn.Conn().Prepare(ctx, "invitation_response_select_stmt", "SELECT EXISTS(SELECT 1 FROM invitations WHERE inviter = $1 AND invited = $2 AND room_id = $3);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var invitationExists bool
	if err = conn.QueryRow(ctx, selectExistsStmt.Name, data.Inviter, uid, data.RoomID).Scan(&invitationExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if !invitationExists {
		return fmt.Errorf("This user did not send you an invitation")
	}

	selectInvitationExistsStmt, err := conn.Conn().Prepare(ctx, "invitation_response_select_member_stmt", "SELECT EXISTS(SELECT 1 FROM members WHERE user_id = $1 AND room_id = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var friendsExists bool
	if err = conn.QueryRow(ctx, selectInvitationExistsStmt.Name, uid, data.RoomID).Scan(&friendsExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if friendsExists {
		return fmt.Errorf("This user is already a member of the room")
	}

	deleteStmt, err := conn.Conn().Prepare(ctx, "invitation_response_delete_stmt", "DELETE FROM invitations WHERE inviter = $1 AND invited = $2 AND room_id = $3;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	if _, err = conn.Exec(ctx, deleteStmt.Name, data.Inviter, uid, data.RoomID); err != nil {
		return fmt.Errorf("Internal error")
	}

	selectBlockedStmt, err := conn.Conn().Prepare(ctx, "invitation_response_select_blocked_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocked = $1 AND blocker = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var blockedExists bool
	if err = conn.QueryRow(ctx, selectBlockedStmt.Name, uid, data.Inviter).Scan(&blockedExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blockedExists {
		return fmt.Errorf("This user has blocked your account")
	}

	selectBlockerStmt, err := conn.Conn().Prepare(ctx, "invitation_response_select_blocker_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocker = $1 AND blocked = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var blockerExists bool
	if err = conn.QueryRow(ctx, selectBlockerStmt.Name, uid, data.Inviter).Scan(&blockerExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blockerExists {
		return fmt.Errorf("You have blocked this user, you must unblock them first")
	}

	if data.Accepted {
		insertStmt, err := conn.Conn().Prepare(ctx, "invitation_response_insert_stmt", "INSERT INTO members (user_id,room_id) VALUES($1, $2);")
		if err != nil {
			return fmt.Errorf("Internal error")
		}
		if _, err = conn.Exec(ctx, insertStmt.Name, uid, data.RoomID); err != nil {
			return fmt.Errorf("Internal error")
		}
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids:        []string{uid, data.Inviter},
		MessageType: "INVITATION_RESPONSE",
		Data: socketMessages.InvitationResponse{
			Accepted: data.Accepted,
			Invited:  uid,
			Inviter:  data.Inviter,
			RoomID:   data.RoomID,
		},
	}

	return nil
}

func ban(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.BanUnban{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	if data.Uid == uid {
		return fmt.Errorf("You cannot ban yourself")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	var author_id string
	if err = h.DB.QueryRow(ctx, "SELECT author_id FROM rooms WHERE room_id = $1;", data.RoomID).Scan(&author_id); err != nil {
		return fmt.Errorf("Internal error")
	}
	if author_id != uid {
		return fmt.Errorf("Only the owner of a room can ban users")
	}

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	blockedExistsStmt, err := conn.Conn().Prepare(ctx, "ban_ban_exists_stmt", "SELECT EXISTS(SELECT 1 FROM bans WHERE user_id = $1 AND room_id = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}

	var banExists bool
	if err = conn.QueryRow(ctx, blockedExistsStmt.Name, data.Uid, data.RoomID).Scan(&banExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if banExists {
		return fmt.Errorf("User is already banned from this room")
	}

	insertBanStmt, err := conn.Conn().Prepare(ctx, "ban_insert_stmt", "INSERT INTO bans (user_id, room_id) VALUES($1, $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	if _, err = conn.Exec(ctx, insertBanStmt.Name, data.Uid, data.RoomID); err != nil {
		return fmt.Errorf("Internal error")
	}

	deleteMsgsStmt, err := conn.Conn().Prepare(ctx, "ban_delete_msgs_stmt", "DELETE FROM room_messages WHERE author_id = $1 AND room_id = $2;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	if _, err = conn.Exec(ctx, deleteMsgsStmt.Name, data.Uid, data.RoomID); err != nil {
		return fmt.Errorf("Internal error")
	}

	selectChannelsStmt, err := conn.Conn().Prepare(ctx, "ban_select_channels_stmt", "SELECT id FROM channels WHERE room_id = $1;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	rows, err := conn.Query(ctx, selectChannelsStmt.Name, data.RoomID)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer rows.Close()
	for rows.Next() {
		var id string

		if err = rows.Scan(&id); err != nil {
			return fmt.Errorf("Internal error")
		}

		h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
			SubName: fmt.Sprintf("channel:%v", id),
			Data: socketMessages.Ban{
				UserID: data.Uid,
				RoomID: data.RoomID,
			},
			MessageType: "BAN",
		}
	}

	return nil
}

func unban(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.BanUnban{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	if data.Uid == uid {
		return fmt.Errorf("You cannot unban yourself")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	var author_id string
	if err = h.DB.QueryRow(ctx, "SELECT author_id FROM rooms WHERE room_id = $1;", data.RoomID).Scan(&author_id); err != nil {
		return fmt.Errorf("Internal error")
	}
	if author_id != uid {
		return fmt.Errorf("Only the owner of a room can unban users")
	}

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	banSelectStmt, err := conn.Conn().Prepare(ctx, "unban_select_ban_stmt", "SELECT EXISTS(SELECT 1 FROM bans WHERE user_id = $1 AND room_id = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var banExists bool
	if err = conn.QueryRow(ctx, banSelectStmt.Name, data.Uid, data.RoomID).Scan(&banExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if !banExists {
		return fmt.Errorf("You cannot unban a user that is not banned")
	}

	deleteStmt, err := conn.Conn().Prepare(ctx, "unban_delete_stmt", "DELETE FROM bans WHERE user_id = $1 AND room_id = $2;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	if _, err = conn.Exec(ctx, deleteStmt.Name, data.Uid, data.RoomID); err != nil {
		return fmt.Errorf("Internal error")
	}

	return nil
}

func block(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.BlockUnBlock{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	if data.Uid == uid {
		return fmt.Errorf("You cannot block yourself")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectStmt, err := conn.Conn().Prepare(ctx, "block_select_block_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocked = $1 AND blocker = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var blockExists bool
	if err = conn.QueryRow(ctx, selectStmt.Name, data.Uid, uid).Scan(&blockExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blockExists {
		return fmt.Errorf("You have already blocked this user")
	}

	insertStmt, err := conn.Conn().Prepare(ctx, "block_insert_block_stmt", "INSERT INTO blocks (blocked, blocker) VALUES($1, $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	if _, err = conn.Exec(ctx, insertStmt.Name, data.Uid, uid); err != nil {
		return fmt.Errorf("Internal error")
	}

	deleteMsgsStmt, err := conn.Conn().Prepare(ctx, "block_delete_msgs_stmt", "DELETE FROM direct_messages WHERE (author_id = $1 AND recipient_id = $2) OR (recipient_id = $1 AND author_id = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	if _, err = conn.Exec(ctx, deleteMsgsStmt.Name, data.Uid, uid); err != nil {
		return fmt.Errorf("Internal error")
	}

	deleteFriendsStmt, err := conn.Conn().Prepare(ctx, "block_delete_friends_stmt", "DELETE FROM friends WHERE (friended = $1 AND friender = $2) OR (friender = $1 AND friended = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	if _, err = conn.Exec(ctx, deleteFriendsStmt.Name, data.Uid, uid); err != nil {
		if err != pgx.ErrNoRows {
			return fmt.Errorf("Internal error")
		}
	}

	h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
		Uids: []string{uid, data.Uid},
		Data: socketMessages.Block{
			Blocker: uid,
			Blocked: data.Uid,
		},
		MessageType: "BLOCK",
	}

	return nil
}

func unblock(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.BlockUnBlock{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	if data.Uid == uid {
		return fmt.Errorf("You cannot unblock yourself")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectStmt, err := conn.Conn().Prepare(ctx, "unblock_select_block_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocked = $1 AND blocker = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var blockExists bool
	if err = conn.QueryRow(ctx, selectStmt.Name, data.Uid, uid).Scan(&blockExists); err != nil {
		return fmt.Errorf("Internal error")
	}
	if !blockExists {
		return fmt.Errorf("You cannot unblock a user that you haven't blocked")
	}

	deleteStmt, err := conn.Conn().Prepare(ctx, "unblock_delete_block_stmt", "DELETE FROM blocks WHERE blocked = $1 AND blocker = $2;")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	if _, err = conn.Exec(ctx, deleteStmt.Name, data.Uid, uid); err != nil {
		return fmt.Errorf("Internal error")
	}

	return nil
}

func callUser(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.CallUser{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	if data.Uid == uid {
		return fmt.Errorf("You cannot call yourself")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	selectBlockedStmt, err := conn.Conn().Prepare(ctx, "call_user_select_blocked_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocked = $1 AND blocker = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var blocked bool
	if err = conn.Conn().QueryRow(ctx, selectBlockedStmt.Name, uid, data.Uid).Scan(&blocked); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blocked {
		return fmt.Errorf("This user has blocked your account")
	}

	selectBlockerStmt, err := conn.Conn().Prepare(ctx, "call_user_select_blocker_stmt", "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocked = $1 AND blocker = $2);")
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	var blocker bool
	if err = conn.Conn().QueryRow(ctx, selectBlockerStmt.Name, data.Uid, uid).Scan(&blocker); err != nil {
		return fmt.Errorf("Internal error")
	}
	if blocked {
		return fmt.Errorf("You cannot call a user you have blocked")
	}

	h.CallServer.CallsPendingChan <- callServer.InCall{
		Caller: uid,
		Called: data.Uid,
	}

	return nil
}

func callUserResponse(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.CallResponse{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	if data.Called != uid && data.Caller != uid || uid == data.Caller && data.Accept {
		return fmt.Errorf("Unauthorized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("Internal error")
	}
	defer conn.Release()

	h.CallServer.ResponseToCallChan <- callServer.InCallResponse{
		Caller: data.Caller,
		Called: data.Called,
		Accept: data.Accept,
	}

	return nil
}

func callLeave(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.CallLeave{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	h.CallServer.LeaveCallChan <- uid

	return nil
}

func callOffer(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.CallOfferAndAnswer{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	h.CallServer.SendCallRecipientOffer <- callServer.CallerSignal{
		Caller:            uid,
		Signal:            data.Signal,
		UserMediaStreamID: data.UserMediaStreamID,
		UserMediaVid:      data.UserMediaVid,
		DisplayMediaVid:   data.DisplayMediaVid,
	}

	return nil
}

func callAnswer(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.CallOfferAndAnswer{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	h.CallServer.SendCalledAnswer <- callServer.CalledSignal{
		Called:            uid,
		Signal:            data.Signal,
		UserMediaStreamID: data.UserMediaStreamID,
		UserMediaVid:      data.UserMediaVid,
		DisplayMediaVid:   data.DisplayMediaVid,
	}

	return nil
}

func callUpdateMediaOptions(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.CallUpdateMediaOptions{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	h.CallServer.UpdateMediaOptions <- callServer.UpdateMediaOptions{
		Uid:               uid,
		UserMediaStreamID: data.UserMediaStreamID,
		UserMediaVid:      data.UserMediaVid,
		DisplayMediaVid:   data.DisplayMediaVid,
	}

	return nil
}

func callRequestReinitialization(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.CallRequestReinitialization{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	h.CallServer.CallRecipientRequestedReInitialization <- uid

	return nil
}

func channelWebRTCJoin(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.ChannelWebRTCJoin{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	h.ChannelRTCServer.JoinChannelRTC <- channelRTCserver.JoinChannel{
		Uid:               uid,
		ChannelID:         data.ChannelID,
		UserMediaStreamID: data.UserMediaStreamID,
		UserMediaVid:      data.UserMediaVid,
		DisplayMediaVid:   data.DisplayMediaVid,
	}

	return nil
}

func channelWebRTCLeave(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.ChannelWebRTCLeave{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	h.ChannelRTCServer.LeaveChannelRTC <- channelRTCserver.LeaveChannel{
		Uid:       uid,
		ChannelID: data.ChannelID,
	}

	return nil
}

func channelWebRTCSendingSignal(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.ChannelWebRTCSendingSignal{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	h.ChannelRTCServer.SignalRTC <- channelRTCserver.SignalRTC{
		Signal:            data.Signal,
		ToUid:             data.Uid,
		Uid:               uid,
		UserMediaStreamID: data.UserMediaStreamID,
		UserMediaVid:      data.UserMediaVid,
		DisplayMediaVid:   data.DisplayMediaVid,
	}

	return nil
}

func channelWebRTCReturningSignal(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.ChannelWebRTCReturningSignal{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	h.ChannelRTCServer.ReturnSignalRTC <- channelRTCserver.ReturnSignalRTC{
		Signal:            data.Signal,
		CallerID:          data.CallerID,
		Uid:               uid,
		UserMediaStreamID: data.UserMediaStreamID,
		UserMediaVid:      data.UserMediaVid,
		DisplayMediaVid:   data.DisplayMediaVid,
	}

	return nil
}

func channelWebRTCUpdateMediaOptions(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.ChannelUpdateMediaOptions{}
	var err error
	if err = UnmarshalMap(inData, data); err != nil {
		return err
	}

	h.ChannelRTCServer.UpdateMediaOptions <- channelRTCserver.UpdateMediaOptions{
		ChannelID:         data.ChannelID,
		Uid:               uid,
		UserMediaStreamID: data.UserMediaStreamID,
		UserMediaVid:      data.UserMediaVid,
		DisplayMediaVid:   data.DisplayMediaVid,
	}

	return nil
}

func startWatching(inData map[string]interface{}, h handler, uid string, c *websocket.Conn) error {
	data := &socketValidation.StartStopWatching{}
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
	data := &socketValidation.StartStopWatching{}
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
