package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/valyala/fasthttp"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	"github.com/web-stuff-98/psql-social/pkg/responses"
	socketmessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
	"github.com/web-stuff-98/psql-social/pkg/validation"
)

func (h handler) CreateRoom(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
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
		Data: socketmessages.ChangeEvent{
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
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
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

	updateStmt, err := conn.Conn().Prepare(rctx, "update_room_stmt", "UPDATE rooms SET name = $1 WHERE id = $2")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	name := strings.TrimSpace(body.Name)
	if _, err := conn.Exec(rctx, updateStmt.Name, name, room_id); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	outChangeData := make(map[string]interface{})
	outChangeData["ID"] = room_id
	outChangeData["name"] = name
	outChangeData["is_private"] = body.Private

	h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
		SubName: fmt.Sprintf("room:%v", room_id),
		Data: socketmessages.ChangeEvent{
			Type:   "UPDATE",
			Entity: "ROOM",
			Data:   outChangeData,
		},
		MessageType: "CHANGE",
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (h handler) GetRoom(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
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

// Retrieve the users own rooms, and rooms they are a member of
func (h handler) GetRooms(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
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
			rows.Scan(&room_id)
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
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
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

	deleteStmt, err := conn.Conn().Prepare(rctx, "delete_room_stmt", "DELETE FROM rooms WHERE room_id = $1 AND author_id = $2")
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
		Data: socketmessages.ChangeEvent{
			Data:   outChangeData,
			Entity: "ROOM",
			Type:   "DELETE",
		},
		MessageType: "CHANGE",
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (h handler) GetRoomChannel(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
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

	selectChannelStmt, err := conn.Conn().Prepare(rctx, "get_room_channel_select_channel_stmt", "SELECT id,content,author_id,created_at FROM room_messages WHERE room_channel_id = $1 ORDER BY created_at DESC LIMIT 50")
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

		err = rows.Scan(&id, &content, &author_id, &created_at)

		if err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}

		messages = append(messages, responses.RoomMessage{
			ID:        id,
			Content:   content,
			AuthorID:  author_id,
			CreatedAt: created_at.Time.Format(time.RFC3339),
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
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
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
