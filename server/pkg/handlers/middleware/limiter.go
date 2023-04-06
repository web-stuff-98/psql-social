package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
)

type BlockInfo struct {
	LastRequest      time.Time `json:"last_request"`
	RequestsInWindow uint16    `json:"reqs_in_window"`
}

type SimpleLimiterOpts struct {
	Window        time.Duration `json:"window"`
	MaxReqs       uint16        `json:"max_reqs"`
	BlockDuration time.Duration `json:"block_dur"`
	Message       string        `json:"msg"`
	RouteName     string        `json:"-"`
}

func errMsg(ctx *fasthttp.RequestCtx, s int, m string) {
	ctx.Response.Header.Add("Content-Type", "text/plain")
	ctx.SetStatusCode(s)
	ctx.WriteString(m)
}

func BasicRateLimiter(next fasthttp.RequestHandler, opts SimpleLimiterOpts, rdb *redis.Client, db *pgxpool.Pool) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		rctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		// Find IP block info on redis
		ipInfoKey := "simple-api-limiter-info:" + ctx.RemoteAddr().String() + "=" + opts.RouteName
		ipInfoCmd := rdb.Get(rctx, ipInfoKey)
		ipInfo := &BlockInfo{}
		if ipInfoCmd.Err() == nil {
			// IP block info for route found
			ipInfoString := ipInfoCmd.Val()
			err := json.Unmarshal([]byte(ipInfoString), ipInfo)
			if err != nil {
				errMsg(ctx, http.StatusInternalServerError, "Internal error")
				return
			}
			// Check if blocked
			if ipInfo.RequestsInWindow >= opts.MaxReqs {
				if time.Now().After(ipInfo.LastRequest.Add(opts.BlockDuration)) {
					// The IP was blocked, but is now no longer blocked, so delete the block info and do next
					delCmd := rdb.Del(rctx, ipInfoKey)
					if delCmd.Err() != nil {
						if delCmd.Err() != redis.Nil {
							errMsg(ctx, http.StatusInternalServerError, "Internal error")
							return
						}
					}
					next(ctx)
					return
				} else {
					// The IP is blocked, extend redis key expiration to end of block duration and send err msg
					secsRemaining := ipInfo.LastRequest.Add(opts.BlockDuration).Second() - time.Now().Second()
					setCmd := rdb.Set(rctx, ipInfoKey, ipInfoString, time.Duration(secsRemaining*1000000000))
					if setCmd.Err() != nil {
						errMsg(ctx, http.StatusInternalServerError, "Internal error")
						return
					}
					var msg string
					if opts.Message != "" {
						msg = opts.Message
					} else {
						msg = "Too many requests"
					}
					errMsg(ctx, http.StatusTooManyRequests, msg)
					return
				}
			}
			// If not blocked add to the number of requests
			if ipInfo.LastRequest.Before(time.Now().Add(-opts.Window)) {
				ipInfo.RequestsInWindow = 1
			} else {
				ipInfo.RequestsInWindow++
			}
			ipInfo.LastRequest = time.Now()
			ipInfoBytes, err := json.Marshal(ipInfo)
			if err != nil {
				errMsg(ctx, http.StatusInternalServerError, "Internal error")
				return
			}
			// Set the ip block info
			setCmd := rdb.Set(rctx, ipInfoKey, string(ipInfoBytes), opts.Window)
			if setCmd.Err() != nil {
				errMsg(ctx, http.StatusInternalServerError, "Internal error")
				return
			}
		} else {
			// IP block info for route was not found, set it, but first check if its because there was an internal error
			if ipInfoCmd.Err() != redis.Nil {
				errMsg(ctx, http.StatusInternalServerError, "Internal error")
				return
			} else {
				// No internal error, create IP block info on redis and do next
				ipInfo = &BlockInfo{
					LastRequest:      time.Now(),
					RequestsInWindow: 1,
				}
				ipInfoBytes, err := json.Marshal(ipInfo)
				if err != nil {
					errMsg(ctx, http.StatusInternalServerError, "Internal error")
					return
				}
				// Set the ip block info
				setCmd := rdb.Set(rctx, ipInfoKey, string(ipInfoBytes), opts.Window)
				if setCmd.Err() != nil {
					errMsg(ctx, http.StatusInternalServerError, "Internal error")
					return
				}
				next(ctx)
				return
			}
		}
		next(ctx)
	}
}
