package handlers

import (
	"context"
	"encoding/json"
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
		ctx.Error("Bad request", fasthttp.StatusBadRequest)
		return
	}
	if err := v.Struct(body); err != nil {
		ctx.Error("Bad request", fasthttp.StatusBadRequest)
		return
	}

	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var id, hash, username, role string
	if err := h.DB.QueryRow(rctx, "SELECT id,password,username,role FROM users WHERE LOWER(username) = LOWER($1)", strings.TrimSpace(body.Username)).Scan(&id, &hash, &username, &role); err != nil {
		if err == pgx.ErrNoRows {
			ctx.Error("Account not found", fasthttp.StatusNotFound)
		} else {
			ctx.Error("Internal error", fasthttp.StatusInternalServerError)
		}
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			ctx.Error("Invalid credentials", fasthttp.StatusUnauthorized)
		} else {
			ctx.Error("Internal error", fasthttp.StatusInternalServerError)
		}
		return
	}

	if token, err := authHelpers.GenerateTokenAndSession(h.RedisClient, rctx, id); err != nil {
		ctx.Error("Internal error", fasthttp.StatusInternalServerError)
	} else {
		if outData, err := json.Marshal(responses.UserWithToken{
			Username: username,
			Role:     role,
			Token:    token,
		}); err != nil {
			ctx.Error("Internal error", fasthttp.StatusInternalServerError)
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
		ctx.Error("Bad request", fasthttp.StatusBadRequest)
		return
	}
	if err := v.Struct(body); err != nil {
		ctx.Error("Bad request", fasthttp.StatusBadRequest)
		return
	}

	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	exists := false
	if err := h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = LOWER($1))", strings.TrimSpace(body.Username)).Scan(&exists); err != nil {
		ctx.Error("Internal error", fasthttp.StatusInternalServerError)
		return
	}
	if exists {
		ctx.Error("There is another user already registered with that name", fasthttp.StatusBadRequest)
		return
	}

	var id string
	if hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 14); err != nil {
		ctx.Error("Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		if err := h.DB.QueryRow(rctx, "INSERT INTO users (username, password, role) VALUES ($1, $2, 'USER') RETURNING id;", body.Username, string(hash)).Scan(&id); err != nil {
			ctx.Error("Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}

	if token, err := authHelpers.GenerateTokenAndSession(h.RedisClient, rctx, id); err != nil {
		ctx.Error("Internal error", fasthttp.StatusInternalServerError)
	} else {
		if outData, err := json.Marshal(responses.UserWithToken{
			Username: strings.TrimSpace(body.Username),
			Role:     "USER",
			Token:    token,
		}); err != nil {
			ctx.Error("Internal error", fasthttp.StatusInternalServerError)
		} else {
			ctx.Response.Header.Add("Content-Type", "application/json")
			ctx.Write(outData)
			ctx.SetStatusCode(fasthttp.StatusOK)
		}
	}
}

func (h handler) Refresh(ctx *fasthttp.RequestCtx) {
	oldToken := strings.ReplaceAll(string(ctx.Request.Header.Peek("Authorization")), "Bearer ", "")
	if oldToken == "" {
		ctx.Error("No token provided", fasthttp.StatusUnauthorized)
		return
	}

	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if token, err := authHelpers.RefreshToken(h.RedisClient, rctx, h.DB, oldToken); err != nil {
		ctx.Error("Unauthorized. Your session most likely expired.", fasthttp.StatusUnauthorized)
	} else {
		ctx.Response.Header.Add("Content-Type", "text/plain")
		ctx.Write([]byte(token))
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
}
