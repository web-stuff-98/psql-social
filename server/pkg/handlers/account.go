package handlers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/valyala/fasthttp"
	"github.com/web-stuff-98/psql-social/pkg/responses"
	"github.com/web-stuff-98/psql-social/pkg/validation"
	"golang.org/x/crypto/bcrypt"
)

func (h handler) Login(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("Method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

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

	var hash, username, role string
	if err := h.DB.QueryRow(rctx, "SELECT password,username,role FROM users WHERE LOWER(username) = LOWER($1)", body.Username).Scan(&hash, &username, &role); err != nil {
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

	if outData, err := json.Marshal(responses.User{
		Username: username,
		Role:     role,
	}); err != nil {
		ctx.Error("Internal error", fasthttp.StatusInternalServerError)
	} else {
		ctx.Response.Header.Add("Content-Type", "application/json")
		ctx.Write(outData)
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
}

func (h handler) Register(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("Method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

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
	if err := h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = LOWER($1))", body.Username).Scan(&exists); err != nil {
		ctx.Error("Internal error", fasthttp.StatusInternalServerError)
		return
	}
	if exists {
		ctx.Error("There is another user already registered with that name", fasthttp.StatusBadRequest)
		return
	}

	if hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 14); err != nil {
		ctx.Error("Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		if _, err := h.DB.Exec(rctx, "INSERT INTO users (username, password, role) VALUES ($1, $2, 'USER');", body.Username, string(hash)); err != nil {
			ctx.Error("Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}

	if outData, err := json.Marshal(responses.User{
		Username: body.Username,
		Role:     "USER",
	}); err != nil {
		ctx.Error("Internal error", fasthttp.StatusInternalServerError)
	} else {
		ctx.Response.Header.Add("Content-Type", "application/json")
		ctx.Write(outData)
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
}
