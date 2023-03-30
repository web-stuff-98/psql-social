package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/valyala/fasthttp"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	"github.com/web-stuff-98/psql-social/pkg/responses"
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

	exists := false
	if err := h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM rooms WHERE LOWER(name) = LOWER($1));", name).Scan(&exists); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	if exists {
		ResponseMessage(ctx, "There is already an other room by that name", fasthttp.StatusBadRequest)
		return
	}

	var id string
	if err := h.DB.QueryRow(rctx, "INSERT INTO rooms (name, author_id, private) VALUES ($1, $2, $3) RETURNING id;", name, uid, body.Private).Scan(&id); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
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

	var author_id string
	if err := h.DB.QueryRow(rctx, "SELECT author_id FROM rooms WHERE id = $1;", room_id).Scan(&author_id); err != nil {
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

	name := strings.TrimSpace(body.Name)
	if err := h.DB.QueryRow(rctx, "UPDATE rooms SET name = $1 WHERE id = $2 RETURNING id;", name, room_id).Scan(); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
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

	banExists := false
	if err := h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM bans WHERE user_id = $1 AND room_id = $2);", uid, room_id).Scan(&banExists); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}
	if banExists {
		ResponseMessage(ctx, "You are banned from this room", fasthttp.StatusBadRequest)
		return
	}

	var id, name, author_id string
	var private bool
	if err := h.DB.QueryRow(rctx, "SELECT id,name,author_id,private FROM rooms WHERE id = $1;", room_id).Scan(&id, &name, &author_id, &private); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Room not found", fasthttp.StatusNotFound)
		}
		return
	}

	if private && uid != author_id {
		isMember := false
		if err := h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM members WHERE user_id = $1 AND room_id = $2);", uid, room_id).Scan(&isMember); err != nil {
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
				CreatedAt: created_at.Time.String(),
			})
		}
	}

	// get all the users room memberships
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

	// get the rooms the user is a member of
	if len(memberOf) > 0 {
		query := fmt.Sprintf("SELECT id,name,private,author_id,created_at FROM rooms WHERE id IN (%s);", strings.Join(memberOf, ","))
		if rows, err := h.DB.Query(rctx, query); err != nil {
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

	var room_id string
	if err := h.DB.QueryRow(rctx, "SELECT room_id FROM room_channels WHERE id = $1;", room_channel_id).Scan(&room_id); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Room channel not found", fasthttp.StatusNotFound)
		}
		return
	}

	banExists := false
	if err := h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM bans WHERE user_id = $1 AND room_id = $2);", uid, room_id).Scan(&banExists); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}
	if banExists {
		ResponseMessage(ctx, "You are banned from this room", fasthttp.StatusBadRequest)
		return
	}

	var private bool
	var author_id string
	if err := h.DB.QueryRow(rctx, "SELECT private, author_id FROM rooms WHERE id = $1;", room_id).Scan(&private, &author_id); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Room not found", fasthttp.StatusNotFound)
		}
		return
	}

	if private && uid != author_id {
		isMember := false
		if err := h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM members WHERE user_id = $1 AND room_id = $2);", uid, room_id).Scan(&isMember); err != nil {
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

	rows, err := h.DB.Query(rctx, "SELECT id,content,author_id,created_at FROM room_messages WHERE room_channel_id = $1 ORDER BY created_at DESC LIMIT 50;", room_channel_id)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer rows.Close()

	messages := []responses.RoomMessage{}

	for rows.Next() {
		var id, content, author_id, created_at string

		err = rows.Scan(&id, &content, &author_id, &created_at)

		if err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}

		messages = append(messages, responses.RoomMessage{
			ID:       id,
			Content:  content,
			AuthorID: author_id,
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
