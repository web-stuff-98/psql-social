package handlers

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/valyala/fasthttp"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	"github.com/web-stuff-98/psql-social/pkg/responses"
	"github.com/web-stuff-98/psql-social/pkg/validation"
	"golang.org/x/crypto/bcrypt"
)

func (h handler) Login(ctx *fasthttp.RequestCtx) {
	v := validator.New()
	body := &validation.LoginRegister{}
	if err := json.Unmarshal(ctx.Request.Body(), &body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}
	if err := v.Struct(body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}

	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var id, hash, username, role string
	if err := h.DB.QueryRow(rctx, "SELECT id,password,username,role FROM users WHERE LOWER(username) = LOWER($1)", strings.TrimSpace(body.Username)).Scan(&id, &hash, &username, &role); err != nil {
		log.Println("ERR C:", err)
		if err == pgx.ErrNoRows {
			ResponseMessage(ctx, "Account not found", fasthttp.StatusNotFound)
		} else {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		}
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			ResponseMessage(ctx, "Invalid credentials", fasthttp.StatusUnauthorized)
		} else {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		}
		return
	}

	if cookie, err := authHelpers.GenerateCookieAndSession(h.RedisClient, rctx, id); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
	} else {
		ctx.Response.Header.SetCookie(cookie)
		if outData, err := json.Marshal(responses.User{
			ID:       id,
			Username: username,
			Role:     role,
		}); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ctx.Response.Header.Add("Content-Type", "application/json")
			ctx.Write(outData)
			ctx.SetStatusCode(fasthttp.StatusOK)
		}
	}
}

func (h handler) Register(ctx *fasthttp.RequestCtx) {
	v := validator.New()
	body := &validation.LoginRegister{}
	if err := json.Unmarshal(ctx.Request.Body(), &body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}
	if err := v.Struct(body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}

	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	exists := false
	if err := h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = LOWER($1))", strings.TrimSpace(body.Username)).Scan(&exists); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	if exists {
		ResponseMessage(ctx, "There is another user already registered with that name", fasthttp.StatusBadRequest)
		return
	}

	var id string
	if hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 14); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		if err := h.DB.QueryRow(rctx, "INSERT INTO users (username, password, role) VALUES ($1, $2, 'USER') RETURNING id;", body.Username, string(hash)).Scan(&id); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}

	if cookie, err := authHelpers.GenerateCookieAndSession(h.RedisClient, rctx, id); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
	} else {
		if outData, err := json.Marshal(responses.User{
			ID:       id,
			Username: strings.TrimSpace(body.Username),
			Role:     "USER",
		}); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		} else {
			ctx.Response.Header.Add("Content-Type", "application/json")
			ctx.Response.Header.SetCookie(cookie)
			ctx.Write(outData)
			ctx.SetStatusCode(fasthttp.StatusOK)
		}
	}
}

func (h handler) Refresh(ctx *fasthttp.RequestCtx) {
	oldToken := strings.ReplaceAll(string(ctx.Request.Header.Peek("Authorization")), "Bearer ", "")
	if oldToken == "" {
		ResponseMessage(ctx, "No token provided", fasthttp.StatusForbidden)
		return
	}

	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if cookie, err := authHelpers.RefreshToken(h.RedisClient, ctx, rctx, h.DB); err != nil {
		ResponseMessage(ctx, "Unauthorized. Your session most likely expired", fasthttp.StatusUnauthorized)
	} else {
		ctx.Response.Header.SetCookie(cookie)
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
}

func (h handler) Logout(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	token := strings.ReplaceAll(string(ctx.Request.Header.Peek("Authorization")), "Bearer ", "")
	if token == "" {
		ResponseMessage(ctx, "No token provided", fasthttp.StatusUnauthorized)
		return
	}

	if _, sid, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB); err != nil {
		ResponseMessage(ctx, "Invalid session ID", fasthttp.StatusForbidden)
		return
	} else {
		authHelpers.DeleteSession(h.RedisClient, rctx, sid)
		ctx.Response.Header.SetCookie(authHelpers.GetClearedCookie())
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
}
