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
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/nfnt/resize"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	"github.com/web-stuff-98/psql-social/pkg/responses"
	socketMessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
	"github.com/web-stuff-98/psql-social/pkg/validation"
	"golang.org/x/crypto/bcrypt"
)

func (h handler) Login(ctx *fiber.Ctx) error {
	v := validator.New()
	body := &validation.Login{}
	if err := json.Unmarshal(ctx.Body(), &body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}
	if err := v.Struct(body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	stmt, err := conn.Conn().Prepare(rctx, "login_stmt", `
	SELECT id,password FROM users WHERE LOWER(username) = LOWER($1);
	`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var id, hash string
	if err = conn.QueryRow(rctx, stmt.Name, strings.TrimSpace(body.Username)).Scan(&id, &hash); err != nil {
		if err == pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusNotFound, "Not found")
		} else {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}
	if os.Getenv("ENVIRONMENT") == "PRODUCTION" {
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)); err != nil {
			if err == bcrypt.ErrMismatchedHashAndPassword {
				return fiber.NewError(fiber.StatusUnauthorized, "Invalid credentials")
			} else {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
		}
	} else {
		if body.Password != hash {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid credentials")
		}
	}

	if cookie, err := authHelpers.Authorize(h.RedisClient, rctx, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		ctx.Cookie(cookie)
		ctx.Response().Header.Add("Content-Type", "application/json")
		ctx.WriteString(id)

		return nil
	}
}

func (h handler) Register(ctx *fiber.Ctx) error {
	v := validator.New()
	body := &validation.Register{}
	if err := json.Unmarshal(ctx.Body(), &body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}
	if err := v.Struct(body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	if !body.Policy {
		return fiber.NewError(fiber.StatusBadRequest, "You must agree to the policy")
	}

	if !authHelpers.PasswordValidates(body.Password) {
		return fiber.NewError(fiber.StatusBadRequest, "Password does not meet requirements")
	}

	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	existsStmt, err := conn.Conn().Prepare(rctx, "register_exists_stmt", `
	SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = LOWER($1));
	`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	exists := false
	if err := conn.QueryRow(rctx, existsStmt.Name, strings.TrimSpace(body.Username)).Scan(&exists); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}
	if exists {
		return fiber.NewError(fiber.StatusBadRequest, "There is already another user using that name")
	}

	var id string

	// dont hash passwords in development mode, because it doesn't work with CGO and I need to use the -race flag to debug
	if os.Getenv("ENVIRONMENT") == "PRODUCTION" {
		if hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 14); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			insertStmt, err := conn.Conn().Prepare(rctx, "register_insert_stmt", `
			INSERT INTO users (username, password, role) VALUES ($1, $2, 'USER') RETURNING id;
			`)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}

			if err := conn.QueryRow(rctx, insertStmt.Name, body.Username, string(hash)).Scan(&id); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
		}
	} else {
		insertStmt, err := conn.Conn().Prepare(rctx, "register_insert_nohash_stmt", `
		INSERT INTO users (username, password, role) VALUES ($1, $2, 'USER') RETURNING id;
		`)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}

		if err := conn.QueryRow(rctx, insertStmt.Name, body.Username, body.Password).Scan(&id); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}

	if cookie, err := authHelpers.Authorize(h.RedisClient, rctx, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		ctx.Response().Header.Add("Content-Type", "text/plain")
		ctx.Cookie(cookie)
		ctx.WriteString(id)
	}

	return nil
}

func (h handler) Logout(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	if uid, sid, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB); err != nil {
		log.Println(err)
		ctx.Cookie(authHelpers.GetClearedCookie())
		return fiber.NewError(fiber.StatusForbidden, "You are not logged in")
	} else {
		h.SocketServer.CloseConnChan <- uid
		authHelpers.DeleteSession(h.RedisClient, rctx, sid)
		ctx.Cookie(authHelpers.GetClearedCookie())
	}

	return nil
}

func (h handler) Refresh(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	if cookie, err := authHelpers.RefreshToken(h.RedisClient, ctx, rctx, h.DB); err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized. Your session most likely expired.")
	} else {
		ctx.Cookie(cookie)
	}

	return nil
}

func (h handler) GetNotifications(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	directMessageNotifications := []responses.DirectMessageNotification{}
	if rows, err := h.DB.Query(rctx, "SELECT sender_id FROM direct_message_notifications WHERE user_id = $1;", uid); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var sender_id string
			if err = rows.Scan(&sender_id); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
			directMessageNotifications = append(directMessageNotifications, responses.DirectMessageNotification{
				SenderID: sender_id,
			})
		}
	}

	roomMessageNotifications := []responses.RoomMessageNotification{}
	if rows, err := h.DB.Query(rctx, "SELECT channel_id,room_id FROM room_message_notifications WHERE user_id = $1;", uid); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var channel_id, room_id string
			if err = rows.Scan(&channel_id, &room_id); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
			roomMessageNotifications = append(roomMessageNotifications, responses.RoomMessageNotification{
				ChannelID: channel_id,
				RoomID:    room_id,
			})
		}
	}

	if data, err := json.Marshal(responses.Notifications{
		DirectMessageNotifications: directMessageNotifications,
		RoomMessageNotifications:   roomMessageNotifications,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		ctx.Response().Header.Add("Content-Type", "application/json")
		ctx.Write(data)
	}

	return nil
}

func (h handler) UpdateBio(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	v := validator.New()
	bio := &validation.Bio{}
	if err := json.Unmarshal(ctx.Body(), bio); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}
	if err := v.Struct(bio); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	content := strings.TrimSpace(bio.Content)

	var seeded bool
	if err = h.DB.QueryRow(rctx, `
	SELECT seeded FROM users WHERE id = $1;
	`, uid).Scan(&seeded); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	if seeded {
		return fiber.NewError(fiber.StatusBadRequest, "You cannot modify the example accounts")
	}

	exists := false
	err = h.DB.QueryRow(rctx, `
	SELECT EXISTS(SELECT 1 FROM bios WHERE user_id = $1);
	`, uid).Scan(&exists) // added error handling here
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	var id string
	if content == "" {
		if exists {
			if _, err := h.DB.Exec(rctx, `
			DELETE FROM bios WHERE user_id = $1;
			`, uid); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
		}

		msgData := make(map[string]interface{})
		msgData["ID"] = uid
		h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
			SubName: fmt.Sprintf("bio:%v", uid),
			Data: socketMessages.ChangeEvent{
				Type:   "DELETE",
				Entity: "BIO",
				Data:   msgData,
			},
			MessageType: "CHANGE",
		}

		return nil
	} else {
		msgData := make(map[string]interface{})
		msgData["ID"] = uid
		msgData["content"] = content
		h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
			SubName: fmt.Sprintf("bio:%v", uid),
			Data: socketMessages.ChangeEvent{
				Type:   "UPDATE",
				Entity: "BIO",
				Data:   msgData,
			},
			MessageType: "CHANGE",
		}

		if !exists {
			insertStmt, err := conn.Conn().Prepare(rctx, "insert_bio_stmt", `
			INSERT INTO bios (content,user_id) VALUES ($1, $2) RETURNING id;
			`)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}

			err = conn.QueryRow(rctx, insertStmt.Name, content, uid).Scan(&id)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
			ctx.Response().Header.Add("Content-Type", "text/plain")
			ctx.WriteString(id)
			ctx.Status(fiber.StatusCreated)
		} else {
			updateStmt, err := conn.Conn().Prepare(rctx, "update_bio_stmt", `
			UPDATE bios SET content = $1 WHERE user_id = $2 RETURNING id;
			`)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}

			err = conn.QueryRow(rctx, updateStmt.Name, content, uid).Scan(&id)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
			ctx.Response().Header.Add("Content-Type", "text/plain")
			ctx.WriteString(id)
		}
	}

	return nil
}

func (h handler) UploadPfp(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	var seeded bool
	if err = h.DB.QueryRow(rctx, `
	SELECT seeded FROM users WHERE id = $1;
	`, uid).Scan(&seeded); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	if seeded {
		return fiber.NewError(fiber.StatusBadRequest, "You cannot modify the example accounts")
	}

	fh, err := ctx.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	if fh.Size > 30*1024*1024 {
		return fiber.NewError(fiber.StatusBadRequest, "Maximum 30mb")
	}

	mime := fh.Header.Get("Content-Type")
	if mime != "image/jpeg" && mime != "image/png" {
		return fiber.NewError(fiber.StatusBadRequest, "Only jpeg and png allowed")
	}

	file, err := fh.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
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
		return fiber.NewError(fiber.StatusBadRequest, "Only jpeg and png allowed")
	}
	if decodeErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	buf := &bytes.Buffer{}
	if img.Bounds().Dx() > img.Bounds().Dy() {
		img = resize.Resize(300, 0, img, resize.Lanczos3)
	} else {
		img = resize.Resize(0, 300, img, resize.Lanczos3)
	}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	pfpBytes := buf.Bytes()

	exists := false
	err = h.DB.QueryRow(rctx, `
	SELECT EXISTS(SELECT 1 FROM profile_pictures WHERE user_id = $1);
	`, uid).Scan(&exists)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	if exists {
		if _, err := h.DB.Exec(rctx, `
		UPDATE profile_pictures SET picture_data = $1 WHERE user_id = $2;
		`, pfpBytes, uid); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	} else {
		if _, err := h.DB.Exec(rctx, `
		INSERT INTO profile_pictures (user_id,picture_data,mime) VALUES ($1,$2,'image/jpeg');
		`, uid, pfpBytes); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}

	msgData := make(map[string]interface{})
	msgData["ID"] = uid
	h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
		SubName: fmt.Sprintf("user:%v", uid),
		Data: socketMessages.ChangeEvent{
			Type:   "UPDATE_IMAGE",
			Entity: "USER",
			Data:   msgData,
		},
		MessageType: "CHANGE",
	}

	return nil
}

func (h handler) GetConversees(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	selectDmsStmt, err := conn.Conn().Prepare(rctx, "select_conversees_messages_stmt", `
	SELECT author_id,recipient_id FROM direct_messages WHERE author_id = $1 OR recipient_id = $1;
	`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	uids := make(map[string]struct{})
	if rows, err := conn.Query(rctx, selectDmsStmt.Name, uid); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var author_id, recipient_id string
			if err = rows.Scan(&author_id, &recipient_id); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
			if author_id != uid {
				uids[author_id] = struct{}{}
			} else {
				uids[recipient_id] = struct{}{}
			}
		}
	}

	selectFrqsStmt, err := conn.Conn().Prepare(rctx, "select_conversees_friend_reqeusts_stmt", `
	SELECT friender,friended FROM friend_requests WHERE friender = $1 OR friended = $1;
	`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	if rows, err := conn.Query(rctx, selectFrqsStmt.Name, uid); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var friender, friended string
			if err = rows.Scan(&friender, &friended); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
			if friender != uid {
				uids[friender] = struct{}{}
			} else {
				uids[friended] = struct{}{}
			}
		}
	}

	selectInvsStmt, err := conn.Conn().Prepare(rctx, "select_conversees_invitations_stmt", `
	SELECT inviter,invited FROM invitations WHERE inviter = $1 OR invited = $1;
	`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	if rows, err := conn.Query(rctx, selectInvsStmt.Name, uid); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var inviter, invited string
			if err = rows.Scan(&inviter, &invited); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
			if inviter != uid {
				uids[inviter] = struct{}{}
			} else {
				uids[invited] = struct{}{}
			}
		}
	}

	// delete the users own id because sometimes it somehow magically ends up in the map
	delete(uids, uid)

	uidsArr := []string{}
	for k := range uids {
		uidsArr = append(uidsArr, k)
	}

	if outBytes, err := json.Marshal(uidsArr); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		ctx.Response().Header.Add("Content-Type", "application/json")
		ctx.Write(outBytes)
	}

	return nil
}

func (h handler) GetConversation(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	user_id := ctx.Params("id")
	if user_id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	selectMsgStmt, err := conn.Conn().Prepare(rctx, "get_conversation_select_msgs_stmt", `
	SELECT id,content,author_id,recipient_id,created_at,has_attachment FROM direct_messages WHERE (author_id = $1) OR (recipient_id = $1) ORDER BY created_at ASC LIMIT 50;
	`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	messages := []responses.DirectMessage{}
	if rows, err := conn.Query(rctx, selectMsgStmt.Name, uid); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var id, content, author_id, recipient_id string
			var created_at pgtype.Timestamptz
			var has_attachment bool

			if err = rows.Scan(&id, &content, &author_id, &recipient_id, &created_at, &has_attachment); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}

			messages = append(messages, responses.DirectMessage{
				ID:            id,
				CreatedAt:     created_at.Time.Format(time.RFC3339),
				RecipientID:   recipient_id,
				AuthorID:      author_id,
				Content:       content,
				HasAttachment: has_attachment,
			})
		}
	}

	selectFrqStmt, err := conn.Conn().Prepare(rctx, "get_conversation_select_friend_requests_stmt", `
	SELECT friender,friended,created_at FROM friend_requests WHERE (friender = $1) OR (friended = $1);
	`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	friendRequests := []responses.FriendRequest{}
	if rows, err := conn.Query(rctx, selectFrqStmt.Name, uid); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var friender, friended string
			var created_at pgtype.Timestamptz

			if err = rows.Scan(&friender, &friended, &created_at); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}

			friendRequests = append(friendRequests, responses.FriendRequest{
				Friender:  friender,
				Friended:  friended,
				CreatedAt: created_at.Time.Format(time.RFC3339),
			})
		}
	}

	selectInvStmt, err := conn.Conn().Prepare(rctx, "get_conversation_select_invitations_stmt", `
	SELECT inviter,invited,created_at,room_id FROM invitations WHERE (inviter = $1) OR (invited = $1);
	`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	invitations := []responses.Invitation{}
	if rows, err := conn.Query(rctx, selectInvStmt.Name, uid); err != nil {
		if err != pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var inviter, invited, room_id string
			var created_at pgtype.Timestamptz

			if err = rows.Scan(&inviter, &invited, &created_at, &room_id); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}

			invitations = append(invitations, responses.Invitation{
				Inviter:   inviter,
				Invited:   invited,
				CreatedAt: created_at.Time.Format(time.RFC3339),
				RoomID:    room_id,
			})
		}
	}

	if outBytes, err := json.Marshal(responses.Conversation{
		DirectMessages: messages,
		Invitations:    invitations,
		FriendRequests: friendRequests,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		ctx.Response().Header.Add("Content-Type", "application/json")
		ctx.Write(outBytes)
	}

	return nil
}

func (h handler) GetFriends(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	rows, err := h.DB.Query(rctx, `
	SELECT friender,friended FROM friends WHERE (friender = $1) OR (friended = $1);
	`, uid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer rows.Close()

	uids := []string{}

	for rows.Next() {
		var friender, friended string

		if err = rows.Scan(&friender, &friended); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}

		if friender != uid {
			uids = append(uids, friender)
		} else {
			uids = append(uids, friended)
		}
	}

	if outBytes, err := json.Marshal(uids); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		ctx.Response().Header.Add("Content-Type", "application/json")
		ctx.Write(outBytes)
	}

	return nil
}

func (h handler) GetBlocked(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSid(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	rows, err := h.DB.Query(rctx, `
	SELECT blocked FROM blocks WHERE blocker = $1;
	`, uid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer rows.Close()

	uids := []string{}

	for rows.Next() {
		var blocked string

		if err = rows.Scan(&blocked); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}

		uids = append(uids, blocked)
	}

	if outBytes, err := json.Marshal(uids); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		ctx.Response().Header.Add("Content-Type", "application/json")
		ctx.Write(outBytes)
	}

	return nil
}
