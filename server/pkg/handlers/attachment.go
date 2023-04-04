package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/valyala/fasthttp"
	attachmentServer "github.com/web-stuff-98/psql-social/pkg/attachmentServer"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	"github.com/web-stuff-98/psql-social/pkg/responses"
	socketMessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
	"github.com/web-stuff-98/psql-social/pkg/validation"
)

func (h handler) CreateAttachmentMetadata(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Uanuthorized", fasthttp.StatusUnauthorized)
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	v := validator.New()
	body := &validation.CreateAttachmentMetadata{}
	if err = json.Unmarshal(ctx.Request.Body(), body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}
	if err = v.Struct(body); err != nil {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusBadRequest)
		return
	}

	var tableName string
	if body.IsRoomMsg {
		tableName = "room_messages_attachment_metadata"
	} else {
		tableName = "direct_messages_attachment_metadata"
	}

	if selectStmt, err := conn.Conn().Prepare(rctx, "create_attachment_metadata_select_stmt", `SELECT EXISTS(SELECT 1 FROM "$1" WHERE id = $2)`); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		var exists bool
		if err = conn.Conn().QueryRow(ctx, selectStmt.Name, tableName, body.ID).Scan(&exists); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
		if exists {
			ResponseMessage(ctx, "Attachment metadata already created", fasthttp.StatusInternalServerError)
			return
		}
	}

	var author_id string
	if selectAuthorStmt, err := conn.Conn().Prepare(rctx, "create_attachment_metadata_select_author_stmt", `SELECT author_id FROM "$1" WHERE id = $2`); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		if err = conn.Conn().QueryRow(rctx, selectAuthorStmt.Name, tableName, body.ID).Scan(&author_id); err != nil {
			if err != pgx.ErrNoRows {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			} else {
				ResponseMessage(ctx, "Message not found", fasthttp.StatusNotFound)
			}
			return
		}
	}

	if author_id != uid {
		ResponseMessage(ctx, "Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	if insertStmt, err := conn.Conn().Prepare(rctx, "create_attachment_metadata_insert_stmt", `INSERT INTO "$1" (meta,name,size,failed,ratio,message_id) VALUES($2,$3,$4,$5,$6,$7)`); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		if _, err = conn.Conn().Exec(rctx, insertStmt.Name, tableName, body.Mime, body.Name, body.Size, false, body.ID); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}

	if body.IsRoomMsg {
		var room_channel_id string
		if selectRoomChannelStmt, err := conn.Conn().Prepare(rctx, "create_attachment_metadata_select_room_stmt", "SELECT room_channel_id FROM room_messages WHERE id = $1"); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		} else {
			if err = conn.QueryRow(rctx, selectRoomChannelStmt.Name, body.ID).Scan(&room_channel_id); err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
		}
		subName := fmt.Sprintf("channel:%v", room_channel_id)
		h.SocketServer.SendDataToSub <- socketServer.SubscriptionMessageData{
			SubName: subName,
			Data: socketMessages.AttachmentMetadataCreated{
				Mime: body.Mime,
				Size: body.Size,
				Name: body.Name,
				ID:   body.ID,
			},
			MessageType: "ATTACHMENT_METADATA_CREATED",
		}
	} else {
		var recipient_id string
		if selectRecipientStmt, err := conn.Conn().Prepare(rctx, "create_attachment_metadata_select_recipient_stmt", "SELECT recipient_id FROM direct_messages WHERE id = $1"); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		} else {
			if err = conn.QueryRow(rctx, selectRecipientStmt.Name, body.ID).Scan(&recipient_id); err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
		}
		h.SocketServer.SendDataToUsers <- socketServer.UsersMessageData{
			Uids: []string{uid, recipient_id},
			Data: socketMessages.AttachmentMetadataCreated{
				Mime: body.Mime,
				Size: body.Size,
				Name: body.Name,
				ID:   body.ID,
			},
			MessageType: "ATTACHMENT_METADATA_CREATED",
		}
	}

	ResponseMessage(ctx, "Metadata created", fasthttp.StatusCreated)
}

func (h handler) UploadAttachmentChunk(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Uanuthorized", fasthttp.StatusUnauthorized)
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	id := ctx.UserValue("id").(string)
	if id == "" {
		ResponseMessage(ctx, "Provide a message ID", fasthttp.StatusBadRequest)
		return
	}

	var isRoomMsg, isDirectMessage bool
	if selectRoomMsgStmt, err := conn.Conn().Prepare(rctx, "upload_attachment_chunk_select_room_message_stmt", "SELECT EXISTS(SELECT 1 FROM room_messages WHERE id = $1 AND author_id = $2)"); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		if err = conn.Conn().QueryRow(rctx, selectRoomMsgStmt.Name, id, uid).Scan(&isRoomMsg); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}
	if selectDirectMsgStmt, err := conn.Conn().Prepare(rctx, "upload_attachment_chunk_select_direct_message_stmt", "SELECT EXISTS(SELECT 1 FROM room_messages WHERE id = $1 AND author_id = $2)"); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		if err = conn.Conn().QueryRow(rctx, selectDirectMsgStmt.Name, id, uid).Scan(&isDirectMessage); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}

	if !isRoomMsg && !isDirectMessage {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusInternalServerError)
		return
	}

	uids := []string{}
	if isRoomMsg {
		var room_channel_id string
		if selectRoomChannelStmt, err := conn.Conn().Prepare(rctx, "upload_attachment_chunk_select_channel_stmt", "SELECT room_channel_id FROM room_messages WHERE id = $1"); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		} else {
			if err = conn.QueryRow(rctx, selectRoomChannelStmt.Name, id).Scan(&room_channel_id); err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
		}
		recvChan := make(chan map[string]struct{})
		h.SocketServer.GetSubscriptionUids <- socketServer.GetSubscriptionUids{
			SubName:  fmt.Sprintf("channel:%v", room_channel_id),
			RecvChan: recvChan,
		}
		uidsMap := <-recvChan
		for k := range uidsMap {
			uids = append(uids, k)
		}
	} else {
		var recipient_id string
		if selectRecipientStmt, err := conn.Conn().Prepare(rctx, "upload_attachment_chunk_select_recipient_stmt", "SELECT recipient_id FROM direct_messages WHERE id = $1"); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		} else {
			if err = conn.QueryRow(rctx, selectRecipientStmt.Name, id).Scan(&recipient_id); err != nil {
				ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
				return
			}
		}
		uids = append(uids, recipient_id)
		uids = append(uids, uid)
	}

	recvChan := make(chan bool)
	h.AttachmentServer.ChunkChan <- attachmentServer.InChunk{
		Uid:           uid,
		IsRoomMsg:     isRoomMsg,
		SendUpdatesTo: uids,
		Data:          ctx.Request.Body(),
		RecvChan:      recvChan,
	}
	wasErr := <-recvChan

	if wasErr {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
	} else {
		ResponseMessage(ctx, "Chunk created", fasthttp.StatusCreated)
	}
}

func (h handler) GetAttachmentMetadata(ctx *fasthttp.RequestCtx) {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	uid, _, err := authHelpers.GetUidAndSidFromCookie(h.RedisClient, ctx, rctx, h.DB)
	if err != nil {
		ResponseMessage(ctx, "Uanuthorized", fasthttp.StatusUnauthorized)
	}

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	}

	id := ctx.UserValue("id").(string)
	if id == "" {
		ResponseMessage(ctx, "Provide a message ID", fasthttp.StatusBadRequest)
		return
	}

	var isRoomMsg, isDirectMessage bool
	if selectRoomMsgStmt, err := conn.Conn().Prepare(rctx, "get_attachment_metadata_select_room_message_stmt", "SELECT EXISTS(SELECT 1 FROM room_messages WHERE id = $1 AND author_id = $2)"); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		if err = conn.Conn().QueryRow(rctx, selectRoomMsgStmt.Name, id, uid).Scan(&isRoomMsg); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}
	if selectDirectMsgStmt, err := conn.Conn().Prepare(rctx, "get_attachment_metadata_select_direct_message_stmt", "SELECT EXISTS(SELECT 1 FROM room_messages WHERE id = $1 AND author_id = $2)"); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		if err = conn.Conn().QueryRow(rctx, selectDirectMsgStmt.Name, id, uid).Scan(&isDirectMessage); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}

	if !isRoomMsg && !isDirectMessage {
		ResponseMessage(ctx, "Bad request", fasthttp.StatusInternalServerError)
		return
	}

	var tableName string
	if isRoomMsg {
		tableName = "room_messages_attachment_metadata"
	} else {
		tableName = "direct_messages_attachment_metadata"
	}

	var ratio float32
	var size int
	var name, meta string
	var failed bool
	if selectStmt, err := conn.Conn().Prepare(rctx, "get_attachment_metadata_select_stmt", `SELECT meta,name,size,ratio,failed FROM "$1" WHERE id = $2`); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		if err = conn.Conn().QueryRow(rctx, selectStmt.Name, tableName, id).Scan(&meta, &name, &size, &ratio, &failed); err != nil {
			ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
			return
		}
	}

	if outBytes, err := json.Marshal(responses.AttachmentMetadata{
		ID:     id,
		Size:   size,
		Name:   name,
		Meta:   meta,
		Failed: failed,
	}); err != nil {
		ResponseMessage(ctx, "Internal error", fasthttp.StatusInternalServerError)
		return
	} else {
		ctx.Response.Header.Add("Content-Type", "application/json")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.Write(outBytes)
	}
}
