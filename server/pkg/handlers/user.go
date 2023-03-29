package handlers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/valyala/fasthttp"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	"github.com/web-stuff-98/psql-social/pkg/responses"
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

	var id, username, role string
	if err := h.DB.QueryRow(rctx, "SELECT id,username,role FROM users WHERE id = $1;", user_id).Scan(&id, &username, &role); err != nil {
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

func (h handler) SearchUser(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	var id, username, role string
	if err := h.DB.QueryRow(ctx, "SELECT id,username,role FROM users WHERE LOWER(username) = LOWER($1);").Scan(&id, &username, &role); err != nil {
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
