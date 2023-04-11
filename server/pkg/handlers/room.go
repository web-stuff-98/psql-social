package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/nfnt/resize"
	"github.com/web-stuff-98/psql-social/pkg/channelRTCserver"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	"github.com/web-stuff-98/psql-social/pkg/responses"
	socketMessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
	"github.com/web-stuff-98/psql-social/pkg/validation"
)

func (h handler) CreateRoom(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	v := validator.New()
	body := &validation.CreateUpdateRoom{}
	if err := json.Unmarshal(ctx.Body(), &body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}
	if err := v.Struct(body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	name := strings.TrimSpace(body.Name)

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	existsStmt, err := conn.Conn().Prepare(rctx, "create_room_exists_stmt", "SELECT EXISTS(SELECT 1 FROM rooms WHERE LOWER(name) = LOWER($1));")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	exists := false
	if err := conn.QueryRow(rctx, existsStmt.Name, name).Scan(&exists); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	if exists {
		return fiber.NewError(fiber.StatusBadRequest, "There is already an other room by that name")
	}

	insertRoomStmt, err := conn.Conn().Prepare(rctx, "insert_room_stmt", "INSERT INTO rooms (name, author_id, private) VALUES ($1, $2, $3) RETURNING id;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var id string
	if err := conn.QueryRow(rctx, insertRoomStmt.Name, name, uid, body.Private).Scan(&id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	insertChannelStmt, err := conn.Conn().Prepare(rctx, "insert_main_channel_stmt", "INSERT INTO room_channels (name, main, room_id) VALUES ($1, $2, $3);")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	if _, err := conn.Exec(rctx, insertChannelStmt.Name, "Main channel", true, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	outChangeData := make(map[string]interface{})
	outChangeData["ID"] = id
	outChangeData["name"] = name
	outChangeData["is_private"] = body.Private
	outChangeData["author_id"] = uid

	h.SocketServer.SendDataToUser <- socketServer.UserMessageData{
		Uid: uid,
		Data: socketMessages.ChangeEvent{
			Type:   "INSERT",
			Entity: "ROOM",
			Data:   outChangeData,
		},
		MessageType: "CHANGE",
	}

	ctx.Response().Header.Add("Content-Type", "text/plain")
	ctx.WriteString(id)
	ctx.Status(fiber.StatusCreated)

	return nil
}

func (h handler) UpdateRoom(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	v := validator.New()
	body := &validation.CreateUpdateRoom{}
	if err := json.Unmarshal(ctx.Body(), &body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}
	if err := v.Struct(body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	room_id := ctx.Params("id")
	if room_id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	selectStmt, err := conn.Conn().Prepare(rctx, "update_room_select_stmt", "SELECT author_id FROM rooms WHERE id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var author_id string
	if err := conn.QueryRow(rctx, selectStmt.Name, room_id).Scan(&author_id); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "Room not found")
		}
	}

	if author_id != uid {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	updateStmt, err := conn.Conn().Prepare(rctx, "update_room_stmt", "UPDATE rooms SET name = $1, private = $2 WHERE id = $3;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	name := strings.TrimSpace(body.Name)
	if _, err := conn.Exec(rctx, updateStmt.Name, name, body.Private, room_id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	outChangeData := make(map[string]interface{})
	outChangeData["ID"] = room_id
	outChangeData["name"] = name
	outChangeData["is_private"] = body.Private

	h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
		SubName: fmt.Sprintf("room:%v", room_id),
		Data: socketMessages.ChangeEvent{
			Type:   "UPDATE",
			Entity: "ROOM",
			Data:   outChangeData,
		},
		MessageType: "CHANGE",
	}

	return nil
}

func (h handler) UpdateRoomChannel(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	channel_id := ctx.Params("id")
	if channel_id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	v := validator.New()
	body := &validation.CreateUpdateChannel{}
	if err := json.Unmarshal(ctx.Body(), &body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}
	if err := v.Struct(body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	selectChannelStmt, err := conn.Conn().Prepare(rctx, "update_channel_select_channel_stmt", "SELECT room_id FROM room_channels WHERE id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	var room_id string
	if err = conn.QueryRow(rctx, selectChannelStmt.Name, channel_id).Scan(&room_id); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "Channel not found")
		}
	}

	selectRoomStmt, err := conn.Conn().Prepare(rctx, "update_channel_select_room_stmt", "SELECT author_id FROM rooms WHERE id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	var author_id string
	if err = conn.QueryRow(rctx, selectRoomStmt.Name, room_id).Scan(&author_id); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "Room not found")
		}
	}

	if author_id != uid {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	if body.Main {
		// if promoting channel to main, need to set other channels "main" value to false first, there can only be one main
		if _, err = h.DB.Exec(rctx, "UPDATE room_channels SET main = FALSE WHERE room_id = $1;", room_id); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}

		updateChannelStmt, err := conn.Conn().Prepare(rctx, "update_channel_update_with_main_stmt", "UPDATE room_channels SET name = $1, main = $2 WHERE id = $3;")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
		if _, err = conn.Exec(rctx, updateChannelStmt.Name, body.Name, body.Main, channel_id); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	} else {
		// otherwise don't update main
		updateChannelStmt, err := conn.Conn().Prepare(rctx, "update_channel_update_without_main_stmt", "UPDATE room_channels SET name = $1 WHERE id = $2;")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
		if _, err = conn.Exec(rctx, updateChannelStmt.Name, body.Name, channel_id); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}

	selectChannelsStmt, err := conn.Conn().Prepare(rctx, "update_channel_select_channels_stmt", "SELECT id FROM room_channels WHERE room_id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	channel_sub_names := []string{}

	if rows, err := conn.Query(rctx, selectChannelsStmt.Name, room_id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		defer rows.Close()

		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
			channel_sub_names = append(channel_sub_names, fmt.Sprintf("channel:%v", id))
		}
	}

	changeData := make(map[string]interface{})
	changeData["ID"] = channel_id
	changeData["name"] = body.Name
	if body.Main {
		changeData["main"] = true
	}
	h.SocketServer.SendDataToSubs <- socketServer.SubscriptionsMessageData{
		SubNames: channel_sub_names,
		Data: socketMessages.ChangeEvent{
			Type:   "UPDATE",
			Entity: "CHANNEL",
			Data:   changeData,
		},
		MessageType: "CHANGE",
	}

	return nil
}

func (h handler) DeleteRoomChannel(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	channel_id := ctx.Params("id")
	if channel_id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	selectChannelStmt, err := conn.Conn().Prepare(rctx, "delete_channel_select_channel_stmt", "SELECT room_id,main FROM room_channels WHERE id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	var room_id string
	var main bool
	if err = conn.QueryRow(rctx, selectChannelStmt.Name, channel_id).Scan(&room_id, &main); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "Channel not found")
		}
	}

	selectRoomStmt, err := conn.Conn().Prepare(rctx, "delete_channel_select_room_stmt", "SELECT author_id FROM rooms WHERE id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	var author_id string
	if err = conn.QueryRow(rctx, selectRoomStmt.Name, room_id).Scan(&author_id); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "Room not found")
		}
	}

	if author_id != uid {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	if main {
		return fiber.NewError(fiber.StatusBadRequest, "You cannot delete the main channel, create a new main channel first, or promote another channel")
	}

	deleteChannelStmt, err := conn.Conn().Prepare(rctx, "delete_channel_delete_stmt", "DELETE FROM room_channels WHERE id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	if _, err = conn.Exec(rctx, deleteChannelStmt.Name, channel_id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	selectChannelsStmt, err := conn.Conn().Prepare(rctx, "delete_channel_select_channels_stmt", "SELECT id FROM room_channels WHERE room_id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	channel_sub_names := []string{}

	if rows, err := conn.Query(rctx, selectChannelsStmt.Name, room_id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		defer rows.Close()

		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
			channel_sub_names = append(channel_sub_names, fmt.Sprintf("channel:%v", id))
		}
	}

	changeData := make(map[string]interface{})
	changeData["ID"] = channel_id
	h.SocketServer.SendDataToSubs <- socketServer.SubscriptionsMessageData{
		SubNames: channel_sub_names,
		Data: socketMessages.ChangeEvent{
			Type:   "DELETE",
			Entity: "CHANNEL",
			Data:   changeData,
		},
		MessageType: "CHANGE",
	}

	return nil
}

func (h handler) CreateRoomChannel(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	v := validator.New()
	body := &validation.CreateUpdateChannel{}
	if err := json.Unmarshal(ctx.Body(), &body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}
	if err := v.Struct(body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	room_id := ctx.Params("id")
	if room_id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	selectRoomStmt, err := conn.Conn().Prepare(rctx, "create_channel_select_room_stmt", "SELECT author_id FROM rooms WHERE id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	var author_id string
	if err = conn.Conn().QueryRow(rctx, selectRoomStmt.Name, room_id).Scan(&author_id); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "Room not found")
		}
	}
	if author_id != uid {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	channelExistsStmt, err := conn.Conn().Prepare(rctx, "create_channel_select_channel_exists_stmt", "SELECT EXISTS(SELECT 1 FROM room_channels WHERE LOWER(name) = LOWER($1) AND room_id = $2);")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	var channelExists = false
	if err = conn.QueryRow(rctx, channelExistsStmt.Name, body.Name, room_id).Scan(&channelExists); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	if channelExists {
		return fiber.NewError(fiber.StatusBadRequest, "You already have another channel by that name")
	}

	// if the new channel being created is the main channel, set "main" on other channels to false
	if body.Main {
		updateMainStmt, err := conn.Conn().Prepare(rctx, "create_channel_update_main_stmt", "UPDATE room_channels SET main = FALSE WHERE room_id = $1;")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
		if _, err = conn.Exec(rctx, updateMainStmt.Name, room_id); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}

	insertStmt, err := conn.Conn().Prepare(rctx, "create_channel_insert_stmt", "INSERT INTO room_channels (name,main,room_id) VALUES($1,$2,$3) RETURNING id;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	var channel_id string
	if err = conn.Conn().QueryRow(rctx, insertStmt.Name, body.Name, body.Main, room_id).Scan(&channel_id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	selectChannelsStmt, err := conn.Conn().Prepare(rctx, "create_channel_select_channels_stmt", "SELECT id FROM room_channels WHERE room_id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	channel_sub_names := []string{}
	if rows, err := conn.Query(rctx, selectChannelsStmt.Name, room_id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		defer rows.Close()

		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
			channel_sub_names = append(channel_sub_names, fmt.Sprintf("channel:%v", id))
		}
	}

	changeData := make(map[string]interface{})
	changeData["ID"] = channel_id
	changeData["name"] = body.Name
	changeData["main"] = body.Main
	h.SocketServer.SendDataToSubs <- socketServer.SubscriptionsMessageData{
		SubNames: channel_sub_names,
		Data: socketMessages.ChangeEvent{
			Type:   "INSERT",
			Entity: "CHANNEL",
			Data:   changeData,
		},
		MessageType: "CHANGE",
	}

	return nil
}

func (h handler) GetRoom(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	room_id := ctx.Params("id")
	if room_id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	existsStmt, err := conn.Conn().Prepare(rctx, "get_room_exists_stmt", "SELECT EXISTS(SELECT 1 FROM bans WHERE user_id = $1 AND room_id = $2);")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	banExists := false
	if err := conn.QueryRow(rctx, existsStmt.Name, uid, room_id).Scan(&banExists); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}
	if banExists {
		return fiber.NewError(fiber.StatusForbidden, "You are banned from this room")
	}

	selectStmt, err := conn.Conn().Prepare(rctx, "get_room_select_stmt", "SELECT id,name,author_id,private FROM rooms WHERE id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var id, name, author_id string
	var private bool
	if err := conn.QueryRow(rctx, selectStmt.Name, room_id).Scan(&id, &name, &author_id, &private); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "Room not found")
		}
	}

	if private && uid != author_id {
		isMember := false
		memberStmt, err := conn.Conn().Prepare(rctx, "get_room_get_membership_stmt", "SELECT EXISTS(SELECT 1 FROM members WHERE user_id = $1 AND room_id = $2);")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}

		if err := conn.QueryRow(rctx, memberStmt.Name, uid, room_id).Scan(&isMember); err != nil {
			if err != pgx.ErrNoRows {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
		}
		if !isMember {
			return fiber.NewError(fiber.StatusForbidden, "You are not a member of this room")
		}
	}

	if bytes, err := json.Marshal(responses.Room{
		ID:       id,
		Name:     name,
		AuthorID: author_id,
		Private:  private,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		ctx.Response().Header.Add("Content-Type", "application/json")
		ctx.Write(bytes)
	}

	return nil
}

func (h handler) GetRoomImage(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	_, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	id := ctx.Params("id")
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	selectStmt, err := conn.Conn().Prepare(rctx, "get_room_image_select_stmt", "SELECT picture_data,mime FROM room_pictures WHERE room_id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var pictureData pgtype.Bytea
	var mime string
	if err = conn.QueryRow(context.Background(), selectStmt.Name, id).Scan(&pictureData, &mime); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "Pfp not found")
		}
	}

	ctx.Response().Header.Add("Content-Type", mime)
	ctx.Response().Header.Add("Content-Length", strconv.Itoa(len(pictureData.Bytes)))
	ctx.Write(pictureData.Bytes)

	return nil
}

// Retrieve the users own rooms, and rooms they are a member of
func (h handler) GetRooms(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	rooms := []responses.Room{}

	// retrieve the users own rooms first
	if rows, err := h.DB.Query(rctx, "SELECT id,name,private,author_id,created_at FROM rooms WHERE author_id = $1;", uid); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	} else {
		defer rows.Close()

		for rows.Next() {
			var id, name, author_id string
			var created_at pgtype.Timestamptz
			var private bool

			err = rows.Scan(&id, &name, &private, &author_id, &created_at)

			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}

			rooms = append(rooms, responses.Room{
				ID:        id,
				Name:      name,
				Private:   private,
				AuthorID:  author_id,
				CreatedAt: created_at.Time.String(),
			})
		}
	}

	memberOf := []string{}
	if rows, err := h.DB.Query(rctx, "SELECT room_id FROM members WHERE user_id = $1;", uid); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "No rooms found")
		}
	} else {
		defer rows.Close()

		for rows.Next() {
			var room_id string
			err = rows.Scan(&room_id)

			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}

			memberOf = append(memberOf, room_id)
		}
	}

	if len(memberOf) > 0 {
		query := "SELECT id, name, private, author_id, created_at FROM rooms WHERE id = ANY($1);"
		if rows, err := h.DB.Query(rctx, query, memberOf); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			defer rows.Close()

			for rows.Next() {
				var id, name, author_id string
				var created_at pgtype.Timestamptz
				var private bool

				err = rows.Scan(&id, &name, &private, &author_id, &created_at)

				if err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
				}

				rooms = append(rooms, responses.Room{
					ID:        id,
					Name:      name,
					Private:   private,
					AuthorID:  author_id,
					CreatedAt: created_at.Time.String(),
				})
			}
		}
	}

	if data, err := json.Marshal(rooms); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		ctx.Response().Header.Add("Content-Type", "application/json")
		ctx.Write(data)
	}

	return nil
}

func (h handler) DeleteRoom(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	room_id := ctx.Params("id")
	if room_id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	deleteStmt, err := conn.Conn().Prepare(rctx, "delete_room_stmt", "DELETE FROM rooms WHERE id = $1 AND author_id = $2;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	if _, err := conn.Exec(rctx, deleteStmt.Name, room_id, uid); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
		ctx.Status(fiber.StatusNotFound)
	}

	outChangeData := make(map[string]interface{})
	outChangeData["ID"] = room_id

	h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
		SubName: fmt.Sprintf("room:%v", room_id),
		Data: socketMessages.ChangeEvent{
			Data:   outChangeData,
			Entity: "ROOM",
			Type:   "DELETE",
		},
		MessageType: "CHANGE",
	}

	return nil
}

func (h handler) UploadRoomImage(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	id := ctx.Params("id")
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	if conn, err := h.DB.Acquire(rctx); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		defer conn.Release()
		if selectAuthorStmt, err := conn.Conn().Prepare(rctx, "upload_room_img_select_author_stmt", "SELECT author_id FROM rooms WHERE id = $1;"); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			var author_id string
			if err = conn.Conn().QueryRow(rctx, selectAuthorStmt.Name, id).Scan(&author_id); err != nil {
				if err != pgx.ErrNoRows {
					return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
				} else {
					return fiber.NewError(fiber.StatusNotFound, "Room not found")
				}
			}
			if author_id != uid {
				return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
			}
		}
	}

	fh, err := ctx.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Error loading file")
	}
	if fh.Size > 30*1024*1024 {
		return fiber.NewError(fiber.StatusBadRequest, "Maximum 30mb")
	}

	mime := fh.Header.Get("Content-Type")
	if mime != "image/jpeg" && mime != "image/png" {
		return fiber.NewError(fiber.StatusBadRequest, "Only jpeg and png allowed")
	}

	file, err := fh.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Error loading file")
	}
	defer file.Close()

	var img image.Image
	var decodeErr error
	switch mime {
	case "image/jpeg":
		img, decodeErr = jpeg.Decode(file)
	case "image/png":
		img, decodeErr = png.Decode(file)
	default:
		return fiber.NewError(fiber.StatusBadRequest, "Only jpeg and png allowed")
	}
	if decodeErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	buf := &bytes.Buffer{}
	if img.Bounds().Dx() > img.Bounds().Dy() {
		img = resize.Resize(300, 0, img, resize.Lanczos3)
	} else {
		img = resize.Resize(0, 300, img, resize.Lanczos3)
	}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	imgBytes := buf.Bytes()

	exists := false
	err = h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM room_pictures WHERE room_id = $1);", id).Scan(&exists)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	if exists {
		if _, err := h.DB.Exec(rctx, "UPDATE room_pictures SET picture_data = $1 WHERE room_id = $2;", imgBytes, id); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	} else {
		if _, err := h.DB.Exec(rctx, `INSERT INTO room_pictures (room_id,picture_data,mime) VALUES ($1,$2,'image/jpeg');`, id, imgBytes); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}

	msgData := make(map[string]interface{})
	msgData["ID"] = id
	h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
		SubName: fmt.Sprintf("room:%v", id),
		Data: socketMessages.ChangeEvent{
			Type:   "UPDATE_IMAGE",
			Entity: "ROOM",
			Data:   msgData,
		},
		MessageType: "CHANGE",
	}

	return nil
}

func (h handler) GetRoomChannel(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	room_channel_id := ctx.Params("id")
	if room_channel_id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	selectStmt, err := conn.Conn().Prepare(rctx, "get_room_channel_select_stmt", "SELECT room_id FROM room_channels WHERE id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var room_id string
	if err := conn.QueryRow(rctx, selectStmt.Name, room_channel_id).Scan(&room_id); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "Room channel not found")
		}
	}

	banExistsStmt, err := conn.Conn().Prepare(rctx, "get_room_channel_ban_exists_stmt", "SELECT EXISTS(SELECT 1 FROM bans WHERE user_id = $1 AND room_id = $2);")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	banExists := false
	if err := conn.QueryRow(rctx, banExistsStmt.Name, uid, room_id).Scan(&banExists); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}
	if banExists {
		return fiber.NewError(fiber.StatusForbidden, "You are banned from this room")
	}

	selectRoomStmt, err := conn.Conn().Prepare(rctx, "get_room_channel_select_room_stmt", "SELECT private, author_id FROM rooms WHERE id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var private bool
	var author_id string
	if err := conn.QueryRow(rctx, selectRoomStmt.Name, room_id).Scan(&private, &author_id); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "Room not found")
		}
	}

	if private && uid != author_id {
		membershipExists, err := conn.Conn().Prepare(rctx, "get_room_channel_member_stmt", "SELECT EXISTS(SELECT 1 FROM members WHERE user_id = $1 AND room_id = $2);")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}

		isMember := false
		if err := conn.QueryRow(rctx, membershipExists.Name, uid, room_id).Scan(&isMember); err != nil {
			if err != pgx.ErrNoRows {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
		}
		if !isMember {
			return fiber.NewError(fiber.StatusForbidden, "You are not a member of this room")
		}
	}

	selectChannelStmt, err := conn.Conn().Prepare(rctx, "get_room_channel_select_channel_stmt", "SELECT id,content,author_id,created_at,has_attachment FROM room_messages WHERE room_channel_id = $1 ORDER BY created_at ASC LIMIT 50;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	rows, err := conn.Query(rctx, selectChannelStmt.Name, room_channel_id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer rows.Close()
	messages := []responses.RoomMessage{}
	for rows.Next() {
		var id, content, author_id string
		var created_at pgtype.Timestamptz
		var has_attachment bool

		err = rows.Scan(&id, &content, &author_id, &created_at, &has_attachment)

		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}

		messages = append(messages, responses.RoomMessage{
			ID:            id,
			Content:       content,
			AuthorID:      author_id,
			CreatedAt:     created_at.Time.Format(time.RFC3339),
			HasAttachment: has_attachment,
		})
	}

	recvChan := make(chan map[string]struct{}, 1)
	h.ChannelRTCServer.GetChannelUids <- channelRTCserver.GetChannelUids{
		RecvChan:  recvChan,
		ChannelID: room_channel_id,
	}
	uidsMap := <-recvChan

	close(recvChan)

	usersInWebRTC := []string{}
	for k := range uidsMap {
		usersInWebRTC = append(usersInWebRTC, k)
	}

	if bytes, err := json.Marshal(responses.RoomChannel{
		Messages:      messages,
		UsersInWebRTC: usersInWebRTC,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		ctx.Response().Header.Add("Content-Type", "application/json")
		ctx.Write(bytes)
	}

	return nil
}

// Retrieves the channels for a room, excluding messages
func (h handler) GetRoomChannels(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	room_id := ctx.Params("id")
	if room_id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	banExistsStmt, err := conn.Conn().Prepare(rctx, "get_room_channel_ban_exists_stmt", "SELECT EXISTS(SELECT 1 FROM bans WHERE user_id = $1 AND room_id = $2);")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	banExists := false
	if err := conn.QueryRow(rctx, banExistsStmt.Name, uid, room_id).Scan(&banExists); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}
	if banExists {
		return fiber.NewError(fiber.StatusForbidden, "You are banned from this room")
	}

	selectRoomStmt, err := conn.Conn().Prepare(rctx, "get_room_channel_select_room_stmt", "SELECT private, author_id FROM rooms WHERE id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var private bool
	var author_id string
	if err := conn.QueryRow(rctx, selectRoomStmt.Name, room_id).Scan(&private, &author_id); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "Room not found")
		}
	}

	if private && uid != author_id {
		membershipExists, err := conn.Conn().Prepare(rctx, "get_room_channel_member_stmt", "SELECT EXISTS(SELECT 1 FROM members WHERE user_id = $1 AND room_id = $2);")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}

		isMember := false
		if err := conn.QueryRow(rctx, membershipExists.Name, uid, room_id).Scan(&isMember); err != nil {
			if err != pgx.ErrNoRows {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
		}
		if !isMember {
			return fiber.NewError(fiber.StatusForbidden, "You are not a member of this room")
		}
	}

	selectChannelsStatement, err := conn.Conn().Prepare(rctx, "get_room_channels_select_stmt", "SELECT id,name,main FROM room_channels WHERE room_id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	channels := []responses.RoomChannelBase{}

	if rows, err := conn.Query(rctx, selectChannelsStatement.Name, room_id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		defer rows.Close()
		for rows.Next() {
			var id, name string
			var main bool
			if err = rows.Scan(&id, &name, &main); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}

			channels = append(channels, responses.RoomChannelBase{
				ID:   id,
				Name: name,
				Main: main,
			})
		}

		if data, err := json.Marshal(channels); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			ctx.Response().Header.Add("Content-Type", "application/json")
			ctx.Write(data)
		}
	}

	return nil
}
