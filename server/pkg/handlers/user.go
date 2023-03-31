package handlers

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/valyala/fasthttp"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	"github.com/web-stuff-98/psql-social/pkg/responses"
	"github.com/web-stuff-98/psql-social/pkg/validation"
)

func (h handler) GetUser(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	user_id := ctx.UserValue("id").(string)
	if user_id == "" {
		ResponseMessage(ctx, "Provide a user ID", fasthttp.StatusBadRequest)
		return
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	selectUserStmt, err := conn.Conn().Prepare(rctx, "get_user_select_stmt", "SELECT id,username,role FROM users WHERE id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	var id, username, role string
	if err := conn.QueryRow(rctx, selectUserStmt.Name, user_id).Scan(&id, &username, &role); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "User not found", fasthttp.StatusNotFound)
		}
		return
	}

	if bytes, err := json.Marshal(responses.User{
		ID:       id,
		Username: username,
		Role:     role,
	}); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		ctx.Response.Header.Add("Content-Type", "application/json")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.Write(bytes)
	}
}

func (h handler) GetUserByName(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	v := validator.New()
	body := &validation.GetUserByName{}
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

	selectUserStmt, err := conn.Conn().Prepare(rctx, "get_user_by_name_select_stmt", "SELECT id FROM users WHERE LOWER(username) = LOWER($1)")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	var id string
	if err := conn.QueryRow(rctx, selectUserStmt.Name, strings.TrimSpace(body.Username)).Scan(&id); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "User not found", fasthttp.StatusNotFound)
		}
		return
	}

	ctx.Response.Header.Add("Content-Type", "text/plain")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.WriteString(id)
}

func (h handler) GetUserBio(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	user_id := ctx.UserValue("id").(string)
	if user_id == "" {
		ResponseMessage(ctx, "Provide a user ID", fasthttp.StatusBadRequest)
		return
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	selectStmt, err := conn.Conn().Prepare(rctx, "get_user_bio_select_stmt", "SELECT content FROM bios WHERE user_id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	var content string
	if err := conn.QueryRow(rctx, selectStmt.Name, user_id).Scan(&content); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ResponseMessage(ctx, "Bio not found", fasthttp.StatusNotFound)
		}
		return
	}

	ctx.Response.Header.Add("Content-Type", "text/plain")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.WriteString(content)
}

func (h handler) GetUserPfp(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	user_id := ctx.UserValue("id").(string)
	if user_id == "" {
		ResponseMessage(ctx, "Provide a user ID", fasthttp.StatusBadRequest)
		return
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	selectStmt, err := conn.Conn().Prepare(rctx, "get_user_pfp_select_stmt", "SELECT picture_data,mime FROM profile_pictures WHERE user_id = $1")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	var pictureData pgtype.Bytea
	var mime string
	if err = conn.QueryRow(context.Background(), selectStmt.Name, uid).Scan(&pictureData, &mime); err != nil {
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
