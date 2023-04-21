package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/pgtype"
	"github.com/jackc/pgx/v5"
	attachmentServer "github.com/web-stuff-98/psql-social/pkg/attachmentServer"
	attachmentHelpers "github.com/web-stuff-98/psql-social/pkg/helpers/attachmentHelpers"
	"github.com/web-stuff-98/psql-social/pkg/helpers/authHelpers"
	"github.com/web-stuff-98/psql-social/pkg/responses"
	socketMessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
	"github.com/web-stuff-98/psql-social/pkg/validation"
)

func (h handler) CreateAttachmentMetadata(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
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

	v := validator.New()
	body := &validation.CreateAttachmentMetadata{}
	if err = json.Unmarshal(ctx.Body(), body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}
	if err = v.Struct(body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	var isRoomMsg, isDirectMessage bool
	if selectRoomMsgStmt, err := conn.Conn().Prepare(rctx, "create_attachment_metadata_select_room_message_stmt", `
	SELECT EXISTS(SELECT 1 FROM room_messages WHERE id = $1 AND author_id = $2);
	`); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		if err = conn.Conn().QueryRow(rctx, selectRoomMsgStmt.Name, body.ID, uid).Scan(&isRoomMsg); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}
	if selectDirectMsgStmt, err := conn.Conn().Prepare(rctx, "create_attachment_metadata_select_direct_message_stmt", `
	SELECT EXISTS(SELECT 1 FROM room_messages WHERE id = $1 AND author_id = $2);
	`); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		if err = conn.Conn().QueryRow(rctx, selectDirectMsgStmt.Name, body.ID, uid).Scan(&isDirectMessage); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}

	if !isRoomMsg && !isDirectMessage {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	var tableName string
	if isRoomMsg {
		tableName = "room_message_attachment_metadata"
	} else {
		tableName = "direct_message_attachment_metadata"
	}

	if selectStmt, err := conn.Conn().Prepare(rctx, "create_attachment_metadata_select_stmt", fmt.Sprintf(`
	SELECT EXISTS(SELECT 1 FROM %v WHERE id = $1);
	`, tableName)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		var exists bool
		if err = conn.Conn().QueryRow(rctx, selectStmt.Name, body.ID).Scan(&exists); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
		if exists {
			return fiber.NewError(fiber.StatusBadRequest, "Bad request")
		}
	}

	var messagesTableName string
	if isRoomMsg {
		messagesTableName = "room_messages"
	} else {
		messagesTableName = "direct_messages"
	}

	var author_id string
	if selectAuthorStmt, err := conn.Conn().Prepare(rctx, "create_attachment_metadata_select_author_stmt", fmt.Sprintf(`
	SELECT author_id FROM %v WHERE id = $1;
	`, messagesTableName)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		if err = conn.Conn().QueryRow(rctx, selectAuthorStmt.Name, body.ID).Scan(&author_id); err != nil {
			if err != pgx.ErrNoRows {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			} else {
				return fiber.NewError(fiber.StatusNotFound, "Message not found")
			}
		}
	}

	if author_id != uid {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	if insertStmt, err := conn.Conn().Prepare(rctx, "create_attachment_metadata_insert_stmt", fmt.Sprintf(`
	INSERT INTO %v (meta,name,size,failed,ratio,message_id) VALUES($1,$2,$3,$4,$5,$6);
	`, tableName)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		if _, err = conn.Conn().Exec(rctx, insertStmt.Name, body.Mime, body.Name, body.Size, false, 0, body.ID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}

	if isRoomMsg {
		var room_channel_id string
		if selectRoomChannelStmt, err := conn.Conn().Prepare(rctx, "create_attachment_metadata_select_room_stmt", `
		SELECT room_channel_id FROM room_messages WHERE id = $1;
		`); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			if err = conn.QueryRow(rctx, selectRoomChannelStmt.Name, body.ID).Scan(&room_channel_id); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
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
		if selectRecipientStmt, err := conn.Conn().Prepare(rctx, "create_attachment_metadata_select_recipient_stmt", `
		SELECT recipient_id FROM direct_messages WHERE id = $1;
		`); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			if err = conn.QueryRow(rctx, selectRecipientStmt.Name, body.ID).Scan(&recipient_id); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
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

	return nil
}

func (h handler) UploadAttachmentChunk(ctx *fiber.Ctx) error {
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
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

	id := ctx.Params("id")
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	if len(ctx.Body()) > 4*1024*1024 {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	var isRoomMsg, isDirectMessage bool
	if selectRoomMsgStmt, err := conn.Conn().Prepare(rctx, "upload_attachment_chunk_select_room_message_stmt", `
	SELECT EXISTS(SELECT 1 FROM room_messages WHERE id = $1 AND author_id = $2);
	`); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		if err = conn.Conn().QueryRow(rctx, selectRoomMsgStmt.Name, id, uid).Scan(&isRoomMsg); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}
	if selectDirectMsgStmt, err := conn.Conn().Prepare(rctx, "upload_attachment_chunk_select_direct_message_stmt", `
	SELECT EXISTS(SELECT 1 FROM room_messages WHERE id = $1 AND author_id = $2);
	`); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		if err = conn.Conn().QueryRow(rctx, selectDirectMsgStmt.Name, id, uid).Scan(&isDirectMessage); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}

	if !isRoomMsg && !isDirectMessage {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	uids := []string{}
	if isRoomMsg {
		var room_channel_id string
		if selectRoomChannelStmt, err := conn.Conn().Prepare(rctx, "upload_attachment_chunk_select_channel_stmt", `
		SELECT room_channel_id FROM room_messages WHERE id = $1;
		`); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			if err = conn.QueryRow(rctx, selectRoomChannelStmt.Name, id).Scan(&room_channel_id); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
		}
		recvChan := make(chan map[string]struct{}, 1)
		h.SocketServer.GetSubscriptionUids <- socketServer.GetSubscriptionUids{
			SubName:  fmt.Sprintf("channel:%v", room_channel_id),
			RecvChan: recvChan,
		}
		uidsMap := <-recvChan

		close(recvChan)

		for k := range uidsMap {
			uids = append(uids, k)
		}
	} else {
		var recipient_id string
		if selectRecipientStmt, err := conn.Conn().Prepare(rctx, "upload_attachment_chunk_select_recipient_stmt", `
		SELECT recipient_id FROM direct_messages WHERE id = $1;
		`); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			if err = conn.QueryRow(rctx, selectRecipientStmt.Name, id).Scan(&recipient_id); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			}
		}
		uids = []string{uid, recipient_id}
	}

	recvChan := make(chan bool, 1)
	h.AttachmentServer.ChunkChan <- attachmentServer.InChunk{
		Uid:      uid,
		Data:     ctx.Body(),
		RecvChan: recvChan,
		ID:       id,
		Ctx:      rctx,
	}
	complete := <-recvChan

	close(recvChan)

	if !complete {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	return nil
}

func (h handler) GetAttachmentMetadata(ctx *fiber.Ctx) error {
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

	id := ctx.Params("id")
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	var isRoomMsg, isDirectMessage bool
	if selectRoomMsgStmt, err := conn.Conn().Prepare(rctx, "get_attachment_metadata_select_room_message_stmt", `
	SELECT EXISTS(SELECT 1 FROM room_messages WHERE id = $1 AND author_id = $2);
	`); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		if err = conn.Conn().QueryRow(rctx, selectRoomMsgStmt.Name, id, uid).Scan(&isRoomMsg); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}
	if selectDirectMsgStmt, err := conn.Conn().Prepare(rctx, "get_attachment_metadata_select_direct_message_stmt", `
	SELECT EXISTS(SELECT 1 FROM room_messages WHERE id = $1 AND author_id = $2);
	`); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		if err = conn.Conn().QueryRow(rctx, selectDirectMsgStmt.Name, id, uid).Scan(&isDirectMessage); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}

	if !isRoomMsg && !isDirectMessage {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	var tableName string
	if isRoomMsg {
		tableName = "room_message_attachment_metadata"
	} else {
		tableName = "direct_message_attachment_metadata"
	}

	var ratio float32
	var size int
	var name, meta string
	var failed bool
	if selectStmt, err := conn.Conn().Prepare(rctx, "get_attachment_metadata_select_stmt", fmt.Sprintf(`
	SELECT meta,name,size,ratio,failed FROM %v WHERE id = $1;
	`, tableName)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		if err = conn.Conn().QueryRow(rctx, selectStmt.Name, id).Scan(&meta, &name, &size, &ratio, &failed); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}

	if outBytes, err := json.Marshal(responses.AttachmentMetadata{
		ID:     id,
		Size:   size,
		Name:   name,
		Meta:   meta,
		Failed: failed,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		ctx.Response().Header.Add("Content-Type", "application/json")
		ctx.Write(outBytes)
	}

	return nil
}

func (h handler) DownloadAttachment(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	rctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	metaTable, chunkTable, err := attachmentHelpers.GetTableNames(conn, rctx, id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var size int
	var name, meta string
	var failed bool
	var ratio float32
	if selectMetadataStmt, err := conn.Conn().Prepare(rctx, "download_attachment_select_metadata_stmt", fmt.Sprintf(`
	SELECT size,name,meta,failed,ratio FROM %v WHERE message_id = $1;
	`, metaTable)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		if err = conn.Conn().QueryRow(rctx, selectMetadataStmt.Name, id).Scan(&size, &name, &meta, &failed, &ratio); err != nil {
			if err != pgx.ErrNoRows {
				return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
			} else {
				return fiber.NewError(fiber.StatusNotFound, "Metadata not found")
			}
		}
	}
	if failed {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot download a failed attachment")
	}
	if ratio != 1 {
		return fiber.NewError(fiber.StatusBadRequest, "Attachment upload incomplete")
	}

	var index int = 0
	var bytesDone int = 0

	ctx.Response().Header.SetContentType("application/octet-stream")
	ctx.Response().Header.SetContentLength(size)
	ctx.Response().Header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%v"`, url.PathEscape(name)))

	// i used a goto here because it's useful
	var chunkBytes pgtype.Bytea
	recursivelyWriteAttachmentChunksToResponse := func() error {
	WRITE:
		if err = conn.QueryRow(rctx, fmt.Sprintf(`
		SELECT bytes FROM %v WHERE message_id = $1 AND chunk_index = $2;
		`, chunkTable), id, index).Scan(&chunkBytes); err != nil {
			if err == pgx.ErrNoRows {
				rctx.Done()
				return nil
			}
			return err
		} else {
			index++
			bytesDone += len(chunkBytes.Bytes)
			if _, err = ctx.Write(chunkBytes.Bytes); err != nil {
				return err
			}
		}
		goto WRITE
	}

	if err = recursivelyWriteAttachmentChunksToResponse(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	rctx.Done()
	return nil
}

// Doesn't work, cant be asked to fix, I wasted days on this last time.
// It does work for videos smaller than the chunk size but that's completely useless
func (h handler) GetAttachmentVideoPartialContent(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}

	rctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	conn, err := h.DB.Acquire(rctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}
	defer conn.Release()

	metaTable, chunkTable, err := attachmentHelpers.GetTableNames(conn, rctx, id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	var size int
	var meta string
	var failed bool
	var ratio float32

	if selectMetaStmt, err := conn.Conn().Prepare(rctx, "attachment_get_video_stream_select_metadata_stmt", fmt.Sprintf(`
	SELECT size,meta,failed,ratio FROM %v WHERE message_id = $1;
	`, metaTable)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		if err = conn.QueryRow(rctx, selectMetaStmt.Name, id).Scan(&size, &meta, &failed, &ratio); err != nil {
			if err == pgx.ErrNoRows {
				return fiber.NewError(fiber.StatusNotFound, "Attachment metadata not found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		}
	}

	if failed || ratio != 1 {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request")
	}
	// Process the range header
	var maxLength int64
	var start, end int64
	if rangeHeader := ctx.Get("Range"); rangeHeader != "" {
		_, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &start)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid range header")
		}
		maxLength = 4 * 1024 * 1024
		if start+maxLength > int64(size) {
			maxLength = int64(size) - start
		}
		// check if end is present in the range header
		if i := strings.Index(rangeHeader, "-"); i != -1 {
			if end, err = strconv.ParseInt(rangeHeader[i+1:], 10, 64); err != nil {
				// if end is absent, set it
				end = start + maxLength
			}
		} else {
			// if end is absent, set it
			end = start + maxLength
		}
	}

	// Calculate the start and end chunk indexes
	startChunkIndex := int(start / 4 * 1024 * 1024)
	endChunkIndex := int(math.Ceil(float64(end+1)/float64(4*1024*1024))) - 1
	var nums []string
	for i := startChunkIndex; i <= endChunkIndex; i++ {
		nums = append(nums, strconv.Itoa(i))
	}
	// Retrieve the data from relevant chunks
	var chunkBytes []byte
	if selectChunksStmt, err := conn.Conn().Prepare(rctx, "attachment_get_video_stream_select_chunks_stmt", fmt.Sprintf(`
	SELECT bytes FROM %v WHERE message_id = $1 AND chunk_index IN (%v);
	`, chunkTable, strings.Join(nums, ","))); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	} else {
		if rows, err := conn.Query(rctx, selectChunksStmt.Name, id); err != nil {
			if err == pgx.ErrNoRows {
				return fiber.NewError(fiber.StatusNotFound, "Attachment chunk data not found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
		} else {
			i := 0
			defer rows.Close()
			for rows.Next() {
				var bytes pgtype.Bytea
				if err = rows.Scan(&bytes); err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Internal error")
				}
				if i == 0 {
					chunkStart := start - int64(startChunkIndex)*4*1024*1024
					chunkEnd := chunkStart + int64(len(bytes.Bytes))
					if chunkEnd > end {
						chunkEnd = end
					}
					chunkBytes = append(chunkBytes, bytes.Bytes[chunkStart:chunkEnd]...)
				} else if i == endChunkIndex-startChunkIndex {
					chunkEnd := end - int64(endChunkIndex)*4*1024*1024
					chunkBytes = append(chunkBytes, bytes.Bytes[:chunkEnd]...)
				} else {
					chunkBytes = append(chunkBytes, bytes.Bytes...)
				}
				i++
			}
		}
	}

	ctx.Response().Header.Add("Accept-Ranges", "bytes")
	ctx.Response().Header.Add("Content-Length", fmt.Sprint(maxLength))
	ctx.Response().Header.Add("Content-Range", fmt.Sprint(start)+"-"+fmt.Sprint(end)+"/"+fmt.Sprint(size))
	ctx.Response().Header.Add("Content-Type", "video/mp4")

	ctx.Write(chunkBytes[:maxLength])

	return nil
}
