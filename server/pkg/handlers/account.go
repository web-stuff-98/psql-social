package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/nfnt/resize"
	"github.com/valyala/fasthttp"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	socketmessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
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

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	stmt, err := conn.Conn().Prepare(rctx, "login_stmt", "SELECT id,password FROM users WHERE LOWER(username) = LOWER($1)")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	var id, hash string
	if err := conn.QueryRow(rctx, stmt.Name, strings.TrimSpace(body.Username)).Scan(&id, &hash); err != nil {
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
			log.Println("ERR D:", err)
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

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	existsStmt, err := conn.Conn().Prepare(rctx, "register_exists_stmt", "SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = LOWER($1))")
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	exists := false
	if err := conn.QueryRow(rctx, existsStmt.Name, strings.TrimSpace(body.Username)).Scan(&exists); err != nil {
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
		insertStmt, err := conn.Conn().Prepare(rctx, "register_insert_stmt", "INSERT INTO users (username, password, role) VALUES ($1, $2, 'USER') RETURNING id")
		if err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}

		if err := conn.QueryRow(rctx, insertStmt.Name, body.Username, string(hash)).Scan(&id); err != nil {
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

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	defer conn.Release()

	content := strings.TrimSpace(bio.Content)

	exists := false
	err = h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM bios WHERE user_id = $1);", uid).Scan(&exists) // added error handling here
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}
	var id string
	if content == "" {
		if exists {
			if _, err := h.DB.Exec(rctx, "DELETE FROM bios WHERE user_id = $1;", uid); err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
		}

		msgData := make(map[string]interface{})
		msgData["ID"] = uid
		h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
			SubName:     fmt.Sprintf("bio:%v", uid),
			MessageType: "CHANGE",
			Data: socketmessages.ChangeEvent{
				Type:   "DELETE",
				Entity: "BIO",
				Data:   msgData,
			},
		}

		ResponseMessage(ctx, "Bio deleted successfully.", fasthttp.StatusOK)
	} else {
		msgData := make(map[string]interface{})
		msgData["ID"] = uid
		msgData["content"] = content
		h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
			SubName:     fmt.Sprintf("bio:%v", uid),
			MessageType: "CHANGE",
			Data: socketmessages.ChangeEvent{
				Type:   "UPDATE",
				Entity: "BIO",
				Data:   msgData,
			},
		}

		if !exists {
			insertStmt, err := conn.Conn().Prepare(rctx, "insert_bio_stmt", "INSERT INTO bios (content,user_id) VALUES ($1, $2) RETURNING id")
			if err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}

			err = conn.QueryRow(rctx, insertStmt.Name, content, uid).Scan(&id)
			if err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
			ctx.Response.Header.Add("Content-Type", "text/plain")
			ctx.WriteString(id)
			ctx.SetStatusCode(fasthttp.StatusCreated)
		} else {
			updateStmt, err := conn.Conn().Prepare(rctx, "update_bio_stmt", "UPDATE bios SET content = $1 WHERE user_id = $2 RETURNING id")
			if err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}

			err = conn.QueryRow(rctx, updateStmt.Name, content, uid).Scan(&id)
			if err != nil {
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
	if err != nil {
		ResponseMessage(ctx, "Error loading file", fasthttp.StatusInternalServerError)
		return
	}
	if fh.Size > 20*1024*1024 {
		ResponseMessage(ctx, "Maxiumum file size allowed is 20mb", fasthttp.StatusBadRequest)
		return
	}

	mime := fh.Header.Get("Content-Type")
	if mime != "image/jpeg" && mime != "image/png" {
		ResponseMessage(ctx, "Unsupported file format - only jpeg and png allowed", fasthttp.StatusBadRequest)
		return
	}

	file, err := fh.Open()
	if err != nil {
		ResponseMessage(ctx, "Error loading file", fasthttp.StatusInternalServerError)
		return
	}
	defer file.Close()

	var img image.Image
	var decodeErr error
	switch mime {
	case "image/jpeg":
		img, decodeErr = jpeg.Decode(file)
	case "image/png":
		img, decodeErr = png.Decode(file)
	default:
		ResponseMessage(ctx, "Only JPEG and PNG are supported", fasthttp.StatusBadRequest)
		return
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

	exists := false
	err = h.DB.QueryRow(rctx, "SELECT EXISTS(SELECT 1 FROM profile_pictures WHERE user_id = $1);", uid).Scan(&exists)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	if exists {
		if _, err := h.DB.Exec(rctx, "UPDATE profile_pictures SET picture_data = $1 WHERE user_id = $2;", pfpBytes, uid); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	} else {
		if _, err := h.DB.Exec(rctx, `INSERT INTO profile_pictures (user_id,picture_data,mime) VALUES ($1,$2,'image/jpeg');`, uid, pfpBytes); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}

	msgData := make(map[string]interface{})
	msgData["ID"] = uid
	h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
		SubName:     fmt.Sprintf("user:%v", uid),
		MessageType: "CHANGE",
		Data: socketmessages.ChangeEvent{
			Type:   "UPDATE_IMAGE",
			Entity: "USER",
			Data:   msgData,
		},
	}

	ResponseMessage(ctx, "Profile picture updated successfully", fasthttp.StatusOK)
}
