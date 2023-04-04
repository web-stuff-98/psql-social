package attachmentserver

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	socketMessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	socketServer "github.com/web-stuff-98/psql-social/pkg/socketServer"
)

/*
AttachmentServer. This is cleaner than the version in my last project.
Chunks are 4mb each.
*/

type AttachmentServer struct {
	Uploaders Uploaders

	ChunkChan  chan InChunk
	DeleteChan chan Delete
}

func Init(ss *socketServer.SocketServer, db *pgxpool.Pool) *AttachmentServer {
	as := &AttachmentServer{
		Uploaders: Uploaders{
			data: make(map[string]map[string]Upload),
		},

		ChunkChan:  make(chan InChunk),
		DeleteChan: make(chan Delete),
	}
	runServer(as, ss, db)
	return as
}

type Uploaders struct {
	// Outer map key is UID, inner map key is MsgId
	data  map[string]map[string]Upload
	mutex sync.Mutex
}

type Upload struct {
	Index      uint16
	TotalBytes uint32
	IsRoomMsg  bool
	NextId     string
	MsgId      string
	LastChunk  time.Time
	// if timed out, the last chunk was received too long ago. upload has failed
	TimedOut bool
}

type InChunk struct {
	Uid           string
	MsgId         string
	IsRoomMsg     bool
	SendUpdatesTo []string
	Data          []byte
	RecvChan      chan<- bool
}

type Delete struct {
	MsgId string
	Uid   string
}

func runServer(as *AttachmentServer, ss *socketServer.SocketServer, db *pgxpool.Pool) {
	go handleChunks(as, ss, db)
	go deleteAttachment(as, ss, db)
	go socketDisconnect(as, ss, db)

	/* ------- Attachments fail when chunks haven't been received for a while. Keeps memory clear of stale uploads. ------- */
	cleanUpTicker := time.NewTicker(time.Second * 15)
	go func() {
		for {
			select {
			case <-cleanUpTicker.C:
				as.Uploaders.mutex.Lock()
				timedOut := make(map[string][]string)
				for uid, v := range as.Uploaders.data {
					for uploadId, u := range v {
						if u.LastChunk.Before(time.Now().Add(-time.Second * 15)) {
							as.Uploaders.data[uid][uploadId] = Upload{
								TimedOut:   true,
								IsRoomMsg:  u.IsRoomMsg,
								Index:      u.Index,
								TotalBytes: u.TotalBytes,
								NextId:     u.NextId,
								LastChunk:  u.LastChunk,
							}
							timedOut[uid] = append(timedOut[uid], u.MsgId)
						}
					}
				}
				as.Uploaders.mutex.Unlock()
				ctx, err := context.WithTimeout(context.Background(), time.Second*8)
				if err != nil {
					log.Println("Error acquiring context in attachment server cleanup loop:", err)
					continue
				}
				for uid, uploads := range timedOut {
					for _, id := range uploads {
						// don't use the delete channel because it also deletes the attachment metadata document
						// only the chunks and Upload struct should be removed
						deleteAttachmentChunks(id, uid, id, as, db)
						var table string
						as.Uploaders.mutex.Lock()
						if uploader, ok := as.Uploaders.data[uid]; ok {
							if upload, ok := uploader[id]; ok {
								if upload.IsRoomMsg {
									table = "room_messages_attachment_metadata"
								} else {
									table = "direct_messages_attachment_metadata"
								}
								if _, err := db.Exec(ctx, `UPDATE "$1" SET failed = TRUE where id = $2;`, table, id); err != nil {
									log.Println("Error updating failed field in attachment server cleanup loop:", err)
								}
							}
						}
						as.Uploaders.mutex.Unlock()
					}
				}
			}
		}
	}()
}

/* Spaghetti below, had to put an if statement around connection acquire or it would error for some reason */

func handleChunks(as *AttachmentServer, ss *socketServer.SocketServer, db *pgxpool.Pool) {
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in attachment server chunk loop:", r)
			}
			go handleChunks(as, ss, db)
		}()
		chunk := <-as.ChunkChan
		ctx, err := context.WithTimeout(context.Background(), time.Second*8)
		if err != nil {
			log.Println("Error acquiring context in attachment server chunk loop:", err)
			continue
		}
		as.Uploaders.mutex.Lock()
		var size int
		var failed bool
		if conn, err := db.Acquire(ctx); err != nil {
			log.Println("Error acquiring connection in attachmentServer chunk loop:", err)
			as.Uploaders.mutex.Unlock()
			as.DeleteChan <- Delete{
				MsgId: chunk.MsgId,
				Uid:   chunk.Uid,
			}
			chunk.RecvChan <- false
			continue
		} else {
			selectStmt, err := conn.Conn().Prepare(ctx, "attachment_server_select_metadata_stmt", `SELECT size,failed FROM "$1" WHERE id = $2`)
			if err != nil {
				log.Println("Error preparing select metadata statement in attachmentServer chunk loop:", err)
				as.Uploaders.mutex.Unlock()
				as.DeleteChan <- Delete{
					MsgId: chunk.MsgId,
					Uid:   chunk.Uid,
				}
				chunk.RecvChan <- false
				continue
			}
			if err = conn.Conn().QueryRow(ctx, selectStmt.Name, chunk.MsgId).Scan(&size); err != nil {
				log.Println("Error selecting metadata in attachmentServer chunk loop:", err)
				as.Uploaders.mutex.Unlock()
				as.DeleteChan <- Delete{
					MsgId: chunk.MsgId,
					Uid:   chunk.Uid,
				}
				chunk.RecvChan <- false
				continue
			}
			if failed {
				chunk.RecvChan <- false
				continue
			}

			nextId := uuid.New().String()
			if _, ok := as.Uploaders.data[chunk.Uid]; !ok {
				// Create uploader data
				uploaderData := make(map[string]Upload)
				uploaderData[chunk.MsgId] = Upload{
					Index:      0,
					NextId:     nextId,
					TotalBytes: uint32(size),
					IsRoomMsg:  chunk.IsRoomMsg,
					LastChunk:  time.Now(),
					MsgId:      chunk.MsgId,
				}
				as.Uploaders.data[chunk.Uid] = uploaderData
			}
			lastChunk := len(chunk.Data) < 4*1024*1024
			var chunkId string
			if lastChunk {
				nextId = ""
			}
			if as.Uploaders.data[chunk.Uid][chunk.MsgId].Index == 0 {
				chunkId = chunk.MsgId
			} else {
				chunkId = as.Uploaders.data[chunk.Uid][chunk.MsgId].NextId
			}
			var metaTable string
			if as.Uploaders.data[chunk.Uid][chunk.MsgId].IsRoomMsg {
				metaTable = "room_messages_attachment_metadata"
			} else {
				metaTable = "direct_messages_attachment_metadata"
			}
			var chunkTable string
			if as.Uploaders.data[chunk.Uid][chunk.MsgId].IsRoomMsg {
				chunkTable = "room_messages_attachment_metadata"
			} else {
				chunkTable = "direct_messages_attachment_metadata"
			}
			// Write chunk
			insertStmt, err := conn.Conn().Prepare(ctx, "attachment_server_insert_chunk_stmt", `INSERT INTO "$1" (id,bytes,message_id,next_chunk) VALUES($1,$2,$3,$4,$5)`)
			if err != nil {
				log.Println("Error preparing insert chunk statement in attachmentServer chunk loop:", err)
				as.Uploaders.mutex.Unlock()
				as.DeleteChan <- Delete{
					MsgId: chunk.MsgId,
					Uid:   chunk.Uid,
				}
				chunk.RecvChan <- false
				continue
			} else {
				if _, err = conn.Conn().Exec(ctx, insertStmt.Name, chunkTable, chunkId, chunk.Data, chunk.MsgId, nextId); err != nil {
					log.Println("Error inserting chunk in attachmentServer chunk loop:", err)
					as.Uploaders.mutex.Unlock()
					as.DeleteChan <- Delete{
						MsgId: chunk.MsgId,
						Uid:   chunk.Uid,
					}
					chunk.RecvChan <- false
					continue
				}
			}
			if lastChunk {
				// Size less than 4mb, its the last chunk, upload is complete
				delete(as.Uploaders.data[chunk.Uid], chunk.MsgId)
				if len(as.Uploaders.data[chunk.Uid]) == 0 {
					delete(as.Uploaders.data, chunk.Uid)
				}
				// Send progress update
				updateStmt, err := conn.Conn().Prepare(ctx, "attachment_server_update_chunk_complete_stmt", `UPDATE "$1" SET ratio = 1 WHERE id = $2`)
				if err != nil {
					log.Println("Error in prepare update chunk metadata complete statement in attachmentServer chunk loop:", err)
					as.Uploaders.mutex.Unlock()
					as.DeleteChan <- Delete{
						MsgId: chunk.MsgId,
						Uid:   chunk.Uid,
					}
					chunk.RecvChan <- false
					continue
				}
				if _, err := conn.Conn().Exec(ctx, updateStmt.Name, metaTable, chunk.MsgId); err != nil {
					log.Println("Error in update chunk metadata complete statement in attachmentServer chunk loop:", err)
					as.Uploaders.mutex.Unlock()
					as.DeleteChan <- Delete{
						MsgId: chunk.MsgId,
						Uid:   chunk.Uid,
					}
					chunk.RecvChan <- false
					continue
				}
				ss.SendDataToUsers <- socketServer.UsersMessageData{
					Uids: chunk.SendUpdatesTo,
					Data: socketMessages.AttachmentProgress{
						Ratio:  1,
						Failed: false,
						MsgID:  chunk.MsgId,
					},
					MessageType: "ATTACHMENT_PROGRESS",
				}
			} else {
				if upload, ok := as.Uploaders.data[chunk.Uid][chunk.MsgId]; ok {
					if upload.TimedOut {
						chunk.RecvChan <- false
						as.Uploaders.mutex.Unlock()
						continue
					} else {
						// Send progress update
						ratio := (float32(upload.Index) * (4 * 1024 * 1024)) / float32(upload.TotalBytes)
						updateStmt, err := conn.Conn().Prepare(ctx, "attachment_server_update_ratio_stmt", `UPDATE "$1" SET ratio = $2 WHERE id = $3`)
						if err != nil {
							log.Println("Error in prepare update chunk metadata ratio statement in attachmentServer chunk loop:", err)
							as.Uploaders.mutex.Unlock()
							as.DeleteChan <- Delete{
								MsgId: chunk.MsgId,
								Uid:   chunk.Uid,
							}
							chunk.RecvChan <- false
							continue
						}
						if _, err = conn.Exec(ctx, updateStmt.Name, metaTable, ratio, chunk.MsgId); err != nil {
							log.Println("Error in update chunk metadata ratio statement in attachmentServer chunk loop:", err)
							as.Uploaders.mutex.Unlock()
							as.DeleteChan <- Delete{
								MsgId: chunk.MsgId,
								Uid:   chunk.Uid,
							}
							chunk.RecvChan <- false
							continue
						}
						ss.SendDataToUsers <- socketServer.UsersMessageData{
							Uids: chunk.SendUpdatesTo,
							Data: socketMessages.AttachmentProgress{
								Ratio:  ratio,
								Failed: false,
								MsgID:  chunk.MsgId,
							},
							MessageType: "ATTACHMENT_PROGRESS",
						}
						// Increment chunk index
						as.Uploaders.data[chunk.Uid][chunk.MsgId] = Upload{
							Index:      upload.Index + 1,
							TotalBytes: upload.TotalBytes,
							IsRoomMsg:  upload.IsRoomMsg,
							NextId:     nextId,
							LastChunk:  time.Now(),
							MsgId:      chunk.MsgId,
						}
					}
				}
			}
		}
		chunk.RecvChan <- true
		as.Uploaders.mutex.Unlock()
	}
}

func deleteAttachment(as *AttachmentServer, ss *socketServer.SocketServer, db *pgxpool.Pool) {
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in attachment server delete loop:", r)
			}
			go deleteAttachment(as, ss, db)
		}()
		deleteData := <-as.DeleteChan
		ctx, err := context.WithTimeout(context.Background(), time.Second*8)
		if err != nil {
			log.Println("Error acquiring context in attachment server chunk loop:", err)
			continue
		}
		as.Uploaders.mutex.Lock()
		if conn, err := db.Acquire(ctx); err != nil {
			log.Println("Error acquiring connection in delete attachment chunk loop:", err)
			continue
		} else {
			errored := func() {
				delete(as.Uploaders.data[deleteData.Uid], deleteData.MsgId)
				if len(as.Uploaders.data[deleteData.Uid]) == 0 {
					delete(as.Uploaders.data, deleteData.Uid)
				}
				as.Uploaders.mutex.Unlock()
			}
			deleteStmt, err := conn.Conn().Prepare(ctx, "attachment_server_delete_attachment_stmt", `DELETE FROM "$1" WHERE id = $2`)
			if err != nil {
				log.Println("Error preparing delete attachment statement in delete attachment loop:", err)
				errored()
				continue
			}
			var isRoomMsg bool
			if err = db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM room_messages_attachment_metadata);").Scan(&isRoomMsg); err != nil {
				log.Println("Error in select room message attachment data in delete attachment metadata loop:", err)
				errored()
				continue
			}
			var isDirectMessage bool
			if !isRoomMsg {
				if err = db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM direct_messages_attachment_metadata);").Scan(&isDirectMessage); err != nil {
					log.Println("Error in select direct message attachment data in delete attachment metadata loop:", err)
					errored()
					continue
				}
			}
			if !isRoomMsg && !isDirectMessage {
				log.Println("Error in delete attachment loop, message metadata could not be found in either table")
				errored()
				continue
			}
			var metaTable string
			if isRoomMsg {
				metaTable = "room_messages_attachment_metadata"
			} else {
				metaTable = "direct_messages_attachment_metadata"
			}
			if _, err = conn.Conn().Exec(ctx, deleteStmt.Name, metaTable, deleteData.MsgId); err != nil {
				log.Println("Error in delete attachment statement in delete attachment loop:", err)
				errored()
				continue
			}
		}
		deleteAttachmentChunks(deleteData.MsgId, deleteData.Uid, deleteData.MsgId, as, db)
		as.Uploaders.mutex.Unlock()
	}
}

func socketDisconnect(as *AttachmentServer, ss *socketServer.SocketServer, db *pgxpool.Pool) {
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in attachment server disconnect loop:", r)
			}
			go socketDisconnect(as, ss, db)
		}()
		uid := <-ss.AttachmentServerRemoveUploaderChan
		as.Uploaders.mutex.Lock()
		for msgId := range as.Uploaders.data[uid] {
			deleteAttachmentChunks(msgId, uid, msgId, as, db)
		}
		delete(as.Uploaders.data, uid)
		as.Uploaders.mutex.Unlock()
	}
}

func deleteAttachmentChunks(chunkId string, uid string, msgId string, as *AttachmentServer, db *pgxpool.Pool) {
	errored := func() {
		delete(as.Uploaders.data[uid], msgId)
		if len(as.Uploaders.data[uid]) == 0 {
			delete(as.Uploaders.data, uid)
		}
	}
	ctx, err := context.WithTimeout(context.Background(), time.Second*8)
	if err != nil {
		log.Println("Error acquiring context in attachment server chunk loop:", err)
		errored()
	}
	var isRoomMsg bool
	if err := db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM room_messages_attachment_metadata);").Scan(&isRoomMsg); err != nil {
		log.Println("Error in select room message attachment data in delete attachment metadata loop:", err)
		errored()
		return
	}
	var isDirectMessage bool
	if !isRoomMsg {
		if err := db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM direct_messages_attachment_metadata);").Scan(&isDirectMessage); err != nil {
			log.Println("Error in select direct message attachment data in delete attachment metadata loop:", err)
			errored()
			return
		}
	}
	if !isRoomMsg && !isDirectMessage {
		log.Println("Error in delete attachment loop, message metadata could not be found in either table")
		errored()
		return
	}
	var metaTable string
	if isRoomMsg {
		metaTable = "room_messages_attachment_metadata"
	} else {
		metaTable = "direct_messages_attachment_metadata"
	}
	var nextChunkId string
	if err := db.QueryRow(ctx, `SELECT next_chunk FROM "$1" WHERE id = $2;`, metaTable, chunkId).Scan(&nextChunkId); err != nil {
		log.Println("Error in delete attachment loop select next chunk id statement:", err)
		errored()
		return
	}
	if _, err := db.Exec(ctx, `DELETE FROM "$1" WHERE id = $1;`, chunkId); err != nil {
		log.Println("Error in delete attachment loop delete statement:", err)
		errored()
		return
	}
	if nextChunkId == "" {
		return
	}
	deleteAttachmentChunks(nextChunkId, uid, msgId, as, db)
}
