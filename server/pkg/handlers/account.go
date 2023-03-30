package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/nfnt/resize"
	"github.com/valyala/fasthttp"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
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

	// Prepare the SQL statement once and reuse it
	stmt, err := h.DB.Prepare(rctx, "login_stmt", "SELECT id,password FROM users WHERE LOWER(username) = LOWER($1)")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	var id, hash string
	// Execute the prepared statement with the parameter
	if err := h.DB.QueryRow(rctx, stmt.Name, strings.TrimSpace(body.Username)).Scan(&id, &hash); err != nil {
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
		ctx.Response.Header.Add("Content-Type", "application/json")
		ctx.WriteString(id)
		ctx.SetStatusCode(fasthttp.StatusOK)
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

	if !authHelpers.PasswordValidates(body.Password) {
		ResponseMessage(ctx, "Password does not meet requirements", fasthttp.StatusBadRequest)
		return
	}

	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	exists := false
	if err := h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = LOWER($1))", strings.TrimSpace(body.Username)).Scan(&exists); err != nil {
		if err != pgx.ErrNoRows {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
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
		ctx.Response.Header.Add("Content-Type", "text/plain")
		ctx.Response.Header.SetCookie(cookie)
		ctx.WriteString(id)
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
}

func (h handler) Logout(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if _, sid, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB); err != nil {
		log.Println(err)
		ctx.Response.Header.SetCookie(authHelpers.GetClearedCookie())
		ResponseMessage(ctx, "Invalid session ID", fasthttp.StatusForbidden)
		return
	} else {
		authHelpers.DeleteSession(h.RedisClient, rctx, sid)
		ctx.Response.Header.SetCookie(authHelpers.GetClearedCookie())
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
}

func (h handler) Refresh(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if cookie, err := authHelpers.RefreshToken(h.RedisClient, ctx, rctx, h.DB); err != nil {
		ResponseMessage(ctx, "Unauthorized. Your session most likely expired", fasthttp.StatusUnauthorized)
	} else {
		ctx.Response.Header.SetCookie(cookie)
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
}

func (h handler) UpdateBio(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	v := validator.New()
	bio := &validation.Bio{}
	if err := json.Unmarshal(ctx.Request.Body(), bio); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}
	if err := v.Struct(bio); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(bio.Content)

	exists := false
	if err := h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM bios WHERE user_id = $1);", uid).Scan(&exists); err != nil {
		log.Println("ERR A:", err)
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	var id string
	if content == "" {
		if exists {
			if _, err := h.DB.Exec(rctx, "DELETE FROM bios WHERE user_id = $1;", uid); err != nil {
				log.Println("ERR B:", err)
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
		}
		ctx.SetStatusCode(fasthttp.StatusOK)
	} else {
		if !exists {
			if err := h.DB.QueryRow(rctx, "INSERT INTO bios (content,user_id) VALUES ($1, $2) RETURNING id;", content, uid).Scan(&id); err != nil {
				log.Println("ERR C:", err)
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
			ctx.Response.Header.Add("Content-Type", "text/plain")
			ctx.WriteString(id)
			ctx.SetStatusCode(fasthttp.StatusCreated)
		} else {
			if err := h.DB.QueryRow(rctx, "UPDATE bios SET content = $1 WHERE user_id = $2 RETURNING id;", content, uid).Scan(&id); err != nil {
				log.Println("ERR D:", err)
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
			ctx.Response.Header.Add("Content-Type", "text/plain")
			ctx.WriteString(id)
			ctx.SetStatusCode(fasthttp.StatusOK)
		}
	}
}

func (h handler) UploadPfp(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	fh, err := ctx.FormFile("file")
	if fh.Size > 20*1024*1024 {
		ResponseMessage(ctx, "Maxiumum file size allowed is 20mb", fasthttp.StatusBadRequest)
		return
	}

	mime := fh.Header.Get("Content-Type")
	if mime != "image/jpeg" && mime != "image/png" {
		ResponseMessage(ctx, "Unsupported file format - only jpeg and png allowed", fasthttp.StatusBadRequest)
		return
	}

	if file, err := fh.Open(); err != nil {
		ResponseMessage(ctx, "Error loading file", fasthttp.StatusInternalServerError)
		return
	} else {
		defer file.Close()

		var isJPEG, isPNG bool
		isJPEG = mime == "image/jpeg"
		isPNG = mime == "image/png"
		if !isJPEG && !isPNG {
			ResponseMessage(ctx, "Only JPEG and PNG are supported", fasthttp.StatusBadRequest)
			return
		}
		var img image.Image
		var decodeErr error
		if isJPEG {
			img, decodeErr = jpeg.Decode(file)
		} else {
			img, decodeErr = png.Decode(file)
		}
		if decodeErr != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
		buf := &bytes.Buffer{}
		if img.Bounds().Dx() > img.Bounds().Dy() {
			img = resize.Resize(300, 0, img, resize.Lanczos3)
		} else {
			img = resize.Resize(0, 300, img, resize.Lanczos3)
		}
		if err := jpeg.Encode(buf, img, nil); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
		pfpBytes := buf.Bytes()

		pictureData := pgtype.Bytea{Bytes: pfpBytes, Status: pgtype.Present}

		exists := false
		if h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM profile_pictures WHERE user_id = $1);", uid).Scan(&exists); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
		if !exists {
			if _, err := h.DB.Exec(rctx, "INSERT INTO profile_pictures (user_id, picture_data, mime) VALUES ($1, $2, $3) RETURNING id;", uid, pictureData.Bytes, mime); err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
			ResponseMessage(ctx, "Pfp created", fasthttp.StatusCreated)
		} else {
			if _, err := h.DB.Exec(rctx, "UPDATE profile_pictures SET picture_data = $1, mime = $2 WHERE user_id = $3;", pictureData.Bytes, mime, uid); err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
			ResponseMessage(ctx, "Pfp updated", fasthttp.StatusOK)
		}
	}
}
