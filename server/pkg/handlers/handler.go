package handlers

import (
	"github.com/jackc/pgx/v5"
	"github.com/valyala/fasthttp"
)

type handler struct {
	DB *pgx.Conn
}

func New(db *pgx.Conn) handler {
	return handler{db}
}

func (h handler) Ping(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
}
