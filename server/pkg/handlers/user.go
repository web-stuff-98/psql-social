package handlers

import (
	"context"
	"encoding/json"
	"log"
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
		log.Println("ERR A:", err)
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}
	if err := v.Struct(body); err != nil {
		log.Println("ERR B:", err)
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}

	var id string
	if err := h.DB.QueryRow(rctx, "SELECT id FROM users WHERE LOWER(username) = LOWER($1);", strings.TrimSpace(body.Username)).Scan(&id); err != nil {
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

	var content string
	if err := h.DB.QueryRow(rctx, "SELECT content FROM bios WHERE user_id = $1;", user_id).Scan(&content); err != nil {
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

	var pictureData pgtype.Bytea
	var mime string
	if err = h.DB.QueryRow(context.Background(), "SELECT picture_data,mime FROM profile_pictures WHERE user_id = $1;", uid).Scan(&pictureData, &mime); err != nil {
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