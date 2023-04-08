package middleware

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
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

func errMsg(ctx *fiber.Ctx, s int, m string) error {
	ctx.Set("Content-Type", "text/plain")
	return ctx.Status(s).SendString(m)
}

func BasicRateLimiter(next fiber.Handler, opts SimpleLimiterOpts, rdb *redis.Client, db *pgxpool.Pool) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// bypass limiter for development mode
		if os.Getenv("ENVIRONMENT") != "PRODUCTION" {
			return next(ctx)
		}

		rctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		// Find IP block info on redis
		ipInfoKey := "simple-api-limiter-info:" + ctx.IP() + "=" + opts.RouteName
		ipInfoCmd := rdb.Get(rctx, ipInfoKey)
		ipInfo := &BlockInfo{}
		if ipInfoCmd.Err() == nil {
			// IP block info for route found
			ipInfoString := ipInfoCmd.Val()
			err := json.Unmarshal([]byte(ipInfoString), ipInfo)
			if err != nil {
				return errMsg(ctx, fiber.StatusInternalServerError, "Internal error")
			}
			// Check if blocked
			if ipInfo.RequestsInWindow >= opts.MaxReqs {
				if time.Now().After(ipInfo.LastRequest.Add(opts.BlockDuration)) {
					// The IP was blocked, but is now no longer blocked, so delete the block info and do next
					delCmd := rdb.Del(rctx, ipInfoKey)
					if delCmd.Err() != nil {
						if delCmd.Err() != redis.Nil {
							return errMsg(ctx, fiber.StatusInternalServerError, "Internal error")
						}
					}
					return next(ctx)
				} else {
					// The IP is blocked, extend redis key expiration to end of block duration and send err msg
					secsRemaining := int(ipInfo.LastRequest.Add(opts.BlockDuration).Sub(time.Now()).Seconds())
					expireCmd := rdb.Expire(rctx, ipInfoKey, time.Second*time.Duration(secsRemaining))
					if expireCmd.Err() != nil {
						return errMsg(ctx, fiber.StatusInternalServerError, "Internal error")
					}
					return errMsg(ctx, fiber.StatusTooManyRequests, opts.Message)
				}
			} else {
				ipInfo.RequestsInWindow++
				ipInfo.LastRequest = time.Now()
				ipInfoBytes, err := json.Marshal(ipInfo)
				if err != nil {
					return errMsg(ctx, fiber.StatusInternalServerError, "Internal error")
				}
				setCmd := rdb.Set(rctx, ipInfoKey, string(ipInfoBytes), opts.Window)
				if setCmd.Err() != nil {
					return errMsg(ctx, fiber.StatusInternalServerError, "Internal error")
				}
				return next(ctx)
			}
		} else if ipInfoCmd.Err() == redis.Nil {
			ipInfo.RequestsInWindow = 1
			ipInfo.LastRequest = time.Now()
			ipInfoBytes, err := json.Marshal(ipInfo)
			if err != nil {
				return errMsg(ctx, fiber.StatusInternalServerError, "Internal error")
			}
			setCmd := rdb.Set(rctx, ipInfoKey, string(ipInfoBytes), opts.Window)
			if setCmd.Err() != nil {
				return errMsg(ctx, fiber.StatusInternalServerError, "Internal error")
			}
			return next(ctx)
		} else {
			return errMsg(ctx, fiber.StatusInternalServerError, "Internal error")
		}
	}
}
