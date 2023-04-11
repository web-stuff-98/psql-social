package handlers

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	"github.com/web-stuff-98/psql-social/pkg/responses"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
	"github.com/web-stuff-98/psql-social/pkg/validation"
)

func (h handler) GetUser(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	user_id := ctx.Params("id")
	if user_id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	selectUserStmt, err := conn.Conn().Prepare(rctx, "get_user_select_stmt", "SELECT id,username,role FROM users WHERE id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var id, username, role string
	if err := conn.QueryRow(rctx, selectUserStmt.Name, user_id).Scan(&id, &username, &role); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "User not found")
		}
	}

	var isOnline bool

	if uid != user_id {
		recvChan := make(chan bool, 1)
		h.SocketServer.IsUserOnline <- socketServer.IsUserOnline{
			RecvChan: recvChan,
			Uid:      id,
		}
		isOnline = <-recvChan

		close(recvChan)
	} else {
		isOnline = true
	}

	if bytes, err := json.Marshal(responses.User{
		ID:       id,
		Username: username,
		Role:     role,
		Online:   isOnline,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		ctx.Response().Header.Add("Content-Type", "application/json")
		ctx.Write(bytes)
	}

	return nil
}

func (h handler) GetUserByName(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	_, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	v := validator.New()
	body := &validation.GetUserByName{}
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

	selectUserStmt, err := conn.Conn().Prepare(rctx, "get_user_by_name_select_stmt", "SELECT id FROM users WHERE LOWER(username) = LOWER($1);")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var id string
	if err := conn.QueryRow(rctx, selectUserStmt.Name, strings.TrimSpace(body.Username)).Scan(&id); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "User not found")
		}
	}

	ctx.Response().Header.Add("Content-Type", "text/plain")
	ctx.WriteString(id)

	return nil
}

func (h handler) GetUserBio(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	_, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	user_id := ctx.Params("id")
	if user_id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	selectStmt, err := conn.Conn().Prepare(rctx, "get_user_bio_select_stmt", "SELECT content FROM bios WHERE user_id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var content string
	if err := conn.QueryRow(rctx, selectStmt.Name, user_id).Scan(&content); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			return fiber.NewError(fiber.StatusNotFound, "Bio not found")
		}
	}

	ctx.Response().Header.Add("Content-Type", "text/plain")
	ctx.WriteString(content)

	return nil
}

func (h handler) GetUserPfp(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	_, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	user_id := ctx.Params("id")
	if user_id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	selectStmt, err := conn.Conn().Prepare(rctx, "get_user_pfp_select_stmt", "SELECT picture_data,mime FROM profile_pictures WHERE user_id = $1;")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var pictureData pgtype.Bytea
	var mime string
	if err = conn.QueryRow(context.Background(), selectStmt.Name, user_id).Scan(&pictureData, &mime); err != nil {
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
