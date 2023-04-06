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
	"github.com/jackc/pgx/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/nfnt/resize"
	"github.com/valyala/fasthttp"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	"github.com/web-stuff-98/psql-social/pkg/responses"
	socketMessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
	"github.com/web-stuff-98/psql-social/pkg/validation"
)

func (h handler) CreateRoom(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	v := validator.New()
	body := &validation.CreateUpdateRoom{}
	if err := json.Unmarshal(ctx.Request.Body(), &body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}
	if err := v.Struct(body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(body.Name)

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	existsStmt, err := conn.Conn().Prepare(rctx, "create_room_exists_stmt", "SELECT EXISTS(SELECT 1 FROM rooms WHERE LOWER(name) = LOWER($1))")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	exists := false
	if err := conn.QueryRow(rctx, existsStmt.Name, name).Scan(&exists); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	if exists {
		ResponseMessage(ctx, "There is already an other room by that name", fasthttp.StatusBadRequest)
		return
	}

	insertRoomStmt, err := conn.Conn().Prepare(rctx, "insert_room_stmt", "INSERT INTO rooms (name, author_id, private) VALUES ($1, $2, $3) RETURNING id")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	var id string
	if err := conn.QueryRow(rctx, insertRoomStmt.Name, name, uid, body.Private).Scan(&id); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	insertChannelStmt, err := conn.Conn().Prepare(rctx, "insert_main_channel_stmt", "INSERT INTO room_channels (name, main, room_id) VALUES ($1, $2, $3)")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	if _, err := conn.Exec(rctx, insertChannelStmt.Name, "Main channel", true, id); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	outChangeData := make(map[string]interface{})
	outChangeData["ID"] = id
	outChangeData["name"] = name
	outChangeData["is_private"] = body.Private

	h.SocketServer.SendDataToUser <- socketServer.UserMessageData{
		Uid: uid,
		Data: socketMessages.ChangeEvent{
			Type:   "INSERT",
			Entity: "ROOM",
			Data:   outChangeData,
		},
		MessageType: "CHANGE",
	}

	ctx.Response.Header.Add("Content-Type", "text/plain")
	ctx.WriteString(id)
	ctx.SetStatusCode(fasthttp.StatusCreated)
}

func (h handler) UpdateRoom(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	v := validator.New()
	body := &validation.CreateUpdateRoom{}
	if err := json.Unmarshal(ctx.Request.Body(), &body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}
	if err := v.Struct(body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}

	room_id := ctx.UserValue("id").(string)
	if room_id == "" {
		ResponseMessage(ctx, "Provide a room ID", fasthttp.StatusBadRequest)
		return
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	selectStmt, err := conn.Conn().Prepare(rctx, "update_room_select_stmt", "SELECT author_id FROM rooms WHERE id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	var author_id string
	if err := conn.QueryRow(rctx, selectStmt.Name, room_id).Scan(&author_id); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Room not found", fasthttp.StatusNotFound)
		}
		return
	}

	if author_id != uid {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	updateStmt, err := conn.Conn().Prepare(rctx, "update_room_stmt", "UPDATE rooms SET name = $1, private = $2 WHERE id = $3")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	name := strings.TrimSpace(body.Name)
	if _, err := conn.Exec(rctx, updateStmt.Name, name, body.Private, room_id); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
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

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (h handler) UpdateRoomChannel(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	channel_id := ctx.UserValue("id").(string)
	if channel_id == "" {
		ResponseMessage(ctx, "Provide a channel ID", fasthttp.StatusBadRequest)
		return
	}

	v := validator.New()
	body := &validation.CreateUpdateChannel{}
	if err := json.Unmarshal(ctx.Request.Body(), &body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}
	if err := v.Struct(body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	selectChannelStmt, err := conn.Conn().Prepare(rctx, "update_channel_select_channel_stmt", "SELECT room_id FROM room_channels WHERE id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	var room_id string
	if err = conn.QueryRow(rctx, selectChannelStmt.Name, channel_id).Scan(&room_id); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Channel not found", fasthttp.StatusNotFound)
		}
		return
	}

	selectRoomStmt, err := conn.Conn().Prepare(rctx, "update_channel_select_room_stmt", "SELECT author_id FROM rooms WHERE id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	var author_id string
	if err = conn.QueryRow(rctx, selectRoomStmt.Name, room_id).Scan(&author_id); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Room not found", fasthttp.StatusNotFound)
		}
		return
	}

	if author_id != uid {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	if body.Main {
		// if promoting channel to main, need to set other channels "main" value to false first, there can only be one main
		if _, err = h.DB.Exec(rctx, "UPDATE room_channels SET main = FALSE WHERE room_id = $1", room_id); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}

		updateChannelStmt, err := conn.Conn().Prepare(rctx, "update_channel_update_with_main_stmt", "UPDATE room_channels SET name = $1, main = $2 WHERE id = $3")
		if err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
		if _, err = conn.Exec(rctx, updateChannelStmt.Name, body.Name, body.Main, channel_id); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	} else {
		// otherwise don't update main

		updateChannelStmt, err := conn.Conn().Prepare(rctx, "update_channel_update_without_main_stmt", "UPDATE room_channels SET name = $1 WHERE id = $2")
		if err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
		if _, err = conn.Exec(rctx, updateChannelStmt.Name, body.Name, channel_id); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}

	selectChannelsStmt, err := conn.Conn().Prepare(rctx, "update_channel_select_channels_stmt", "SELECT id FROM room_channels WHERE room_id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	channel_sub_names := []string{}

	if rows, err := conn.Query(rctx, selectChannelsStmt.Name, room_id); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		defer rows.Close()

		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
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

	ResponseMessage(ctx, "Channel updated", fasthttp.StatusOK)
}

func (h handler) DeleteRoomChannel(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	channel_id := ctx.UserValue("id").(string)
	if channel_id == "" {
		ResponseMessage(ctx, "Provide a channel ID", fasthttp.StatusBadRequest)
		return
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	selectChannelStmt, err := conn.Conn().Prepare(rctx, "delete_channel_select_channel_stmt", "SELECT room_id,main FROM room_channels WHERE id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	var room_id string
	var main bool
	if err = conn.QueryRow(rctx, selectChannelStmt.Name, channel_id).Scan(&room_id, &main); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Channel not found", fasthttp.StatusNotFound)
		}
		return
	}

	selectRoomStmt, err := conn.Conn().Prepare(rctx, "delete_channel_select_room_stmt", "SELECT author_id FROM rooms WHERE id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	var author_id string
	if err = conn.QueryRow(rctx, selectRoomStmt.Name, room_id).Scan(&author_id); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Room not found", fasthttp.StatusNotFound)
		}
		return
	}

	if author_id != uid {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	if main {
		ResponseMessage(ctx, "You cannot delete the main channel, create a new main channel first, or promote another channel", fasthttp.StatusBadRequest)
		return
	}

	deleteChannelStmt, err := conn.Conn().Prepare(rctx, "delete_channel_delete_stmt", "DELETE FROM room_channels WHERE id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	if _, err = conn.Exec(rctx, deleteChannelStmt.Name, channel_id); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	selectChannelsStmt, err := conn.Conn().Prepare(rctx, "delete_channel_select_channels_stmt", "SELECT id FROM room_channels WHERE room_id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	channel_sub_names := []string{}

	if rows, err := conn.Query(rctx, selectChannelsStmt.Name, room_id); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		defer rows.Close()

		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
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

	ResponseMessage(ctx, "Channel deleted", fasthttp.StatusOK)
}

func (h handler) CreateRoomChannel(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	v := validator.New()
	body := &validation.CreateUpdateChannel{}
	if err := json.Unmarshal(ctx.Request.Body(), &body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}
	if err := v.Struct(body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	room_id := ctx.UserValue("id").(string)
	if room_id == "" {
		ResponseMessage(ctx, "Provide a room ID", fasthttp.StatusBadRequest)
		return
	}

	selectRoomStmt, err := conn.Conn().Prepare(rctx, "create_channel_select_room_stmt", "SELECT author_id FROM rooms WHERE id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	var author_id string
	if err = conn.Conn().QueryRow(rctx, selectRoomStmt.Name, room_id).Scan(&author_id); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Room not found", fasthttp.StatusNotFound)
		}
		return
	}
	if author_id != uid {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	channelExistsStmt, err := conn.Conn().Prepare(rctx, "create_channel_select_channel_exists_stmt", "SELECT EXISTS(SELECT 1 FROM room_channels WHERE LOWER(name) = LOWER($1) AND room_id = $2)")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	var channelExists = false
	if err = conn.QueryRow(rctx, channelExistsStmt.Name, body.Name, room_id).Scan(&channelExists); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	if channelExists {
		ResponseMessage(ctx, "There is already another channel by that name", fasthttp.StatusBadRequest)
		return
	}

	// if the new channel being created is the main channel, set "main" on other channels to false
	if body.Main {
		updateMainStmt, err := conn.Conn().Prepare(rctx, "create_channel_update_main_stmt", "UPDATE room_channels SET main = FALSE WHERE room_id = $1")
		if err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
		if _, err = conn.Exec(rctx, updateMainStmt.Name, room_id); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}

	insertStmt, err := conn.Conn().Prepare(rctx, "create_channel_insert_stmt", "INSERT INTO room_channels (name,main,room_id) VALUES($1,$2,$3) RETURNING id")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	var channel_id string
	if err = conn.Conn().QueryRow(rctx, insertStmt.Name, body.Name, body.Main, room_id).Scan(&channel_id); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	selectChannelsStmt, err := conn.Conn().Prepare(rctx, "create_channel_select_channels_stmt", "SELECT id FROM room_channels WHERE room_id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	channel_sub_names := []string{}
	if rows, err := conn.Query(rctx, selectChannelsStmt.Name, room_id); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		defer rows.Close()

		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
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

	ResponseMessage(ctx, "Channel created", fasthttp.StatusCreated)
}

func (h handler) GetRoom(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	room_id := ctx.UserValue("id").(string)
	if room_id == "" {
		ResponseMessage(ctx, "Provide a room ID", fasthttp.StatusBadRequest)
		return
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	existsStmt, err := conn.Conn().Prepare(rctx, "get_room_exists_stmt", "SELECT EXISTS(SELECT 1 FROM bans WHERE user_id = $1 AND room_id = $2)")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	banExists := false
	if err := conn.QueryRow(rctx, existsStmt.Name, uid, room_id).Scan(&banExists); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}
	if banExists {
		ResponseMessage(ctx, "You are banned from this room", fasthttp.StatusBadRequest)
		return
	}

	selectStmt, err := conn.Conn().Prepare(rctx, "get_room_select_stmt", "SELECT id,name,author_id,private FROM rooms WHERE id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	var id, name, author_id string
	var private bool
	if err := conn.QueryRow(rctx, selectStmt.Name, room_id).Scan(&id, &name, &author_id, &private); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Room not found", fasthttp.StatusNotFound)
		}
		return
	}

	if private && uid != author_id {
		isMember := false
		memberStmt, err := conn.Conn().Prepare(rctx, "get_room_get_membership_stmt", "SELECT EXISTS(SELECT 1 FROM members WHERE user_id = $1 AND room_id = $2)")
		if err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}

		if err := conn.QueryRow(rctx, memberStmt.Name, uid, room_id).Scan(&isMember); err != nil {
			if err != pgx.ErrNoRows {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
		}
		if !isMember {
			ResponseMessage(ctx, "You are not a member of this room", fasthttp.StatusBadRequest)
			return
		}
	}

	if bytes, err := json.Marshal(responses.Room{
		ID:       id,
		Name:     name,
		AuthorID: author_id,
		Private:  private,
	}); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
	} else {
		ctx.Response.Header.Add("Content-Type", "application/json")
		ctx.Write(bytes)
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
}

func (h handler) GetRoomImage(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	_, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	id := ctx.UserValue("id").(string)
	if id == "" {
		ResponseMessage(ctx, "Provide a room ID", fasthttp.StatusBadRequest)
		return
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	selectStmt, err := conn.Conn().Prepare(rctx, "get_room_image_select_stmt", "SELECT picture_data,mime FROM room_pictures WHERE room_id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	var pictureData pgtype.Bytea
	var mime string
	if err = conn.QueryRow(context.Background(), selectStmt.Name, id).Scan(&pictureData, &mime); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Pfp not found", fasthttp.StatusNotFound)
		}
		return
	}

	ctx.Response.Header.Add("Content-Type", mime)
	ctx.Response.Header.Add("Content-Length", strconv.Itoa(len(pictureData.Bytes)))
	ctx.Write(pictureData.Bytes)
	ctx.SetStatusCode(fasthttp.StatusOK)
}

// Retrieve the users own rooms, and rooms they are a member of
func (h handler) GetRooms(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	rooms := []responses.Room{}

	// retrieve the users own rooms first
	if rows, err := h.DB.Query(rctx, "SELECT id,name,private,author_id,created_at FROM rooms WHERE author_id = $1;", uid); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	} else {
		defer rows.Close()

		for rows.Next() {
			var id, name, author_id string
			var created_at pgtype.Timestamptz
			var private bool

			err = rows.Scan(&id, &name, &private, &author_id, &created_at)

			if err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
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
	if rows, err := h.DB.Query(rctx, "SELECT room_id FROM members WHERE user_id = $1", uid); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "No rooms found", fasthttp.StatusNotFound)
		}
		return
	} else {
		defer rows.Close()

		for rows.Next() {
			var room_id string
			err = rows.Scan(&room_id)

			if err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}

			memberOf = append(memberOf, room_id)
		}
	}

	if len(memberOf) > 0 {
		query := "SELECT id, name, private, author_id, created_at FROM rooms WHERE id = ANY($1)"
		if rows, err := h.DB.Query(rctx, query, memberOf); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		} else {
			defer rows.Close()

			for rows.Next() {
				var id, name, author_id string
				var created_at pgtype.Timestamptz
				var private bool

				err = rows.Scan(&id, &name, &private, &author_id, &created_at)

				if err != nil {
					ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
					return
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
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		ctx.Response.Header.Add("Content-Type", "application/json")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.Write(data)
	}
}

func (h handler) DeleteRoom(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	room_id := ctx.UserValue("id").(string)
	if room_id == "" {
		ResponseMessage(ctx, "Provide a room ID", fasthttp.StatusBadRequest)
		return
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	deleteStmt, err := conn.Conn().Prepare(rctx, "delete_room_stmt", "DELETE FROM rooms WHERE id = $1 AND author_id = $2")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	if _, err := conn.Exec(rctx, deleteStmt.Name, room_id, uid); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
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

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (h handler) UploadRoomImage(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	id := ctx.UserValue("id").(string)
	if id == "" {
		ResponseMessage(ctx, "Provide an ID", fasthttp.StatusBadRequest)
		return
	}

	if conn, err := h.DB.Acquire(rctx); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		if selectAuthorStmt, err := conn.Conn().Prepare(rctx, "upload_room_img_select_author_stmt", "SELECT author_id FROM rooms WHERE id = $1"); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		} else {
			var author_id string
			if err = conn.Conn().QueryRow(rctx, selectAuthorStmt.Name, id).Scan(&author_id); err != nil {
				if err != pgx.ErrNoRows {
					ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				} else {
					ResponseMessage(ctx, "Room not found", fasthttp.StatusNotFound)
				}
				return
			}
			if author_id != uid {
				ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
				return
			}
		}
	}

	fh, err := ctx.FormFile("file")
	if err != nil {
		ResponseMessage(ctx, "Error loading file", fasthttp.StatusInternalServerError)
		return
	}
	if fh.Size > 30*1024*1024 {
		ResponseMessage(ctx, "Maxiumum file size allowed is 20mb", fasthttp.StatusBadRequest)
		return
	}

	mime := fh.Header.Get("Content-Type")
	if mime != "image/jpeg" && mime != "image/png" {
		ResponseMessage(ctx, "Unsupported file format - only jpeg and png allowed", fasthttp.StatusBadRequest)
		return
	}

	file, err := fh.Open()
	if err != nil {
		ResponseMessage(ctx, "Error loading file", fasthttp.StatusInternalServerError)
		return
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
		ResponseMessage(ctx, "Only JPEG and PNG are supported", fasthttp.StatusBadRequest)
		return
	}
	if decodeErr != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	buf := &bytes.Buffer{}
	if img.Bounds().Dx() > img.Bounds().Dy() {
		img = resize.Resize(300, 0, img, resize.Lanczos3)
	} else {
		img = resize.Resize(0, 300, img, resize.Lanczos3)
	}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	imgBytes := buf.Bytes()

	exists := false
	err = h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM room_pictures WHERE room_id = $1);", id).Scan(&exists)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	if exists {
		if _, err := h.DB.Exec(rctx, "UPDATE room_pictures SET picture_data = $1 WHERE room_id = $2;", imgBytes, id); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	} else {
		if _, err := h.DB.Exec(rctx, `INSERT INTO room_pictures (room_id,picture_data,mime) VALUES ($1,$2,'image/jpeg');`, id, imgBytes); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
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

	ResponseMessage(ctx, "Image processed successfully", fasthttp.StatusOK)
}

func (h handler) GetRoomChannel(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	room_channel_id := ctx.UserValue("id").(string)
	if room_channel_id == "" {
		ResponseMessage(ctx, "Provide a room channel ID", fasthttp.StatusBadRequest)
		return
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	selectStmt, err := conn.Conn().Prepare(rctx, "get_room_channel_select_stmt", "SELECT room_id FROM room_channels WHERE id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	var room_id string
	if err := conn.QueryRow(rctx, selectStmt.Name, room_channel_id).Scan(&room_id); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Room channel not found", fasthttp.StatusNotFound)
		}
		return
	}

	banExistsStmt, err := conn.Conn().Prepare(rctx, "get_room_channel_ban_exists_stmt", "SELECT EXISTS(SELECT 1 FROM bans WHERE user_id = $1 AND room_id = $2)")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	banExists := false
	if err := conn.QueryRow(rctx, banExistsStmt.Name, uid, room_id).Scan(&banExists); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}
	if banExists {
		ResponseMessage(ctx, "You are banned from this room", fasthttp.StatusBadRequest)
		return
	}

	selectRoomStmt, err := conn.Conn().Prepare(rctx, "get_room_channel_select_room_stmt", "SELECT private, author_id FROM rooms WHERE id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	var private bool
	var author_id string
	if err := conn.QueryRow(rctx, selectRoomStmt.Name, room_id).Scan(&private, &author_id); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Room not found", fasthttp.StatusNotFound)
		}
		return
	}

	if private && uid != author_id {
		membershipExists, err := conn.Conn().Prepare(rctx, "get_room_channel_member_stmt", "SELECT EXISTS(SELECT 1 FROM members WHERE user_id = $1 AND room_id = $2)")
		if err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}

		isMember := false
		if err := conn.QueryRow(rctx, membershipExists.Name, uid, room_id).Scan(&isMember); err != nil {
			if err != pgx.ErrNoRows {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
		}
		if !isMember {
			ResponseMessage(ctx, "You are not a member of this room", fasthttp.StatusBadRequest)
			return
		}
	}

	selectChannelStmt, err := conn.Conn().Prepare(rctx, "get_room_channel_select_channel_stmt", "SELECT id,content,author_id,created_at,has_attachment FROM room_messages WHERE room_channel_id = $1 ORDER BY created_at ASC LIMIT 50")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	rows, err := conn.Query(rctx, selectChannelStmt.Name, room_channel_id)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer rows.Close()

	messages := []responses.RoomMessage{}

	for rows.Next() {
		var id, content, author_id string
		var created_at pgtype.Timestamptz
		var has_attachment bool

		err = rows.Scan(&id, &content, &author_id, &created_at, &has_attachment)

		if err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}

		messages = append(messages, responses.RoomMessage{
			ID:            id,
			Content:       content,
			AuthorID:      author_id,
			CreatedAt:     created_at.Time.Format(time.RFC3339),
			HasAttachment: has_attachment,
		})
	}

	if bytes, err := json.Marshal(messages); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		ctx.SetContentType("application/json")
		ctx.Write(bytes)
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
}

// Retrieves the channels for a room, excluding messages
func (h handler) GetRoomChannels(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	room_id := ctx.UserValue("id").(string)
	if room_id == "" {
		ResponseMessage(ctx, "Provide a room ID", fasthttp.StatusBadRequest)
		return
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	banExistsStmt, err := conn.Conn().Prepare(rctx, "get_room_channel_ban_exists_stmt", "SELECT EXISTS(SELECT 1 FROM bans WHERE user_id = $1 AND room_id = $2)")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	banExists := false
	if err := conn.QueryRow(rctx, banExistsStmt.Name, uid, room_id).Scan(&banExists); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}
	if banExists {
		ResponseMessage(ctx, "You are banned from this room", fasthttp.StatusBadRequest)
		return
	}

	selectRoomStmt, err := conn.Conn().Prepare(rctx, "get_room_channel_select_room_stmt", "SELECT private, author_id FROM rooms WHERE id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	var private bool
	var author_id string
	if err := conn.QueryRow(rctx, selectRoomStmt.Name, room_id).Scan(&private, &author_id); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Room not found", fasthttp.StatusNotFound)
		}
		return
	}

	if private && uid != author_id {
		membershipExists, err := conn.Conn().Prepare(rctx, "get_room_channel_member_stmt", "SELECT EXISTS(SELECT 1 FROM members WHERE user_id = $1 AND room_id = $2)")
		if err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}

		isMember := false
		if err := conn.QueryRow(rctx, membershipExists.Name, uid, room_id).Scan(&isMember); err != nil {
			if err != pgx.ErrNoRows {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
		}
		if !isMember {
			ResponseMessage(ctx, "You are not a member of this room", fasthttp.StatusBadRequest)
			return
		}
	}

	selectChannelsStatement, err := conn.Conn().Prepare(rctx, "get_room_channels_select_stmt", "SELECT id,name,main FROM room_channels WHERE room_id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	channels := []responses.RoomChannelBase{}

	if rows, err := conn.Query(rctx, selectChannelsStatement.Name, room_id); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		defer rows.Close()
		for rows.Next() {
			var id, name string
			var main bool
			if err = rows.Scan(&id, &name, &main); err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}

			channels = append(channels, responses.RoomChannelBase{
				ID:   id,
				Name: name,
				Main: main,
			})
		}

		if data, err := json.Marshal(channels); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		} else {
			ctx.Response.Header.Add("Content-Type", "application/json")
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.Write(data)
		}
	}
}
