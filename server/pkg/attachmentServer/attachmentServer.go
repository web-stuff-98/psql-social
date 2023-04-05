package attachmentserver

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	socketMessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
)

type AttachmentServer struct {
	Uploaders Uploaders

	ChunkChan  chan InChunk
	FailChan   chan string
	DeleteChan chan string
}

type Uploaders struct {
	data  map[string]map[string]Upload
	mutex sync.RWMutex
}

type Upload struct {
	MsgID      string
	Index      int
	LastUpdate time.Time
	BytesDone  int
}

type InChunk struct {
	Data     []byte
	ID       string
	Uid      string
	RecvChan chan bool
	Ctx      context.Context
}

func Init(ss *socketServer.SocketServer, db *pgxpool.Pool) *AttachmentServer {
	as := &AttachmentServer{
		Uploaders: Uploaders{
			data: map[string]map[string]Upload{},
		},

		ChunkChan:  make(chan InChunk),
		FailChan:   make(chan string),
		DeleteChan: make(chan string),
	}
	runServer(ss, as, db)
	return as
}

func runServer(ss *socketServer.SocketServer, as *AttachmentServer, db *pgxpool.Pool) {
	go processChunk(ss, as, db)
	go deleteAttachment(ss, as, db)
	go failAttachment(ss, as, db)
}

func processChunk(ss *socketServer.SocketServer, as *AttachmentServer, db *pgxpool.Pool) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if failCount < 10 {
				failCount++
				log.Println("Recovered from panic in attachment server handle chunk loop:", r)
			} else {
				log.Println("Attachment server panic count exceeded maximum retries")
			}
			failCount++
		}()

		data := <-as.ChunkChan

		errored := func(err error, conn *pgxpool.Conn) {
			log.Printf("Error in attachment server process chunk loop:%v\n", err)
			conn.Release()
			as.FailChan <- data.ID
			data.RecvChan <- false
		}

		conn, err := db.Acquire(data.Ctx)
		if err != nil {
			errored(err, conn)
			continue
		}

		metaTable, chunkTable, err := getTableNames(conn, data.Ctx, data.ID)
		if err != nil {
			errored(err, conn)
			continue
		}

		var i int = 0
		var size float32

		if selectSizeStmt, err := conn.Conn().Prepare(data.Ctx, "attachment_server_chunk_loop_select_meta_size_stmt", fmt.Sprintf("SELECT size FROM %v WHERE message_id = $1", metaTable)); err != nil {
			errored(err, conn)
			continue
		} else {
			if err = conn.Conn().QueryRow(data.Ctx, selectSizeStmt.Name, data.ID).Scan(&size); err != nil {
				errored(err, conn)
				continue
			}
		}

		as.Uploaders.mutex.Lock()

		if _, ok := as.Uploaders.data[data.Uid]; !ok {
			as.Uploaders.data[data.Uid] = make(map[string]Upload)
			as.Uploaders.data[data.Uid][data.ID] = Upload{
				Index:      0,
				MsgID:      data.ID,
				LastUpdate: time.Now(),
				BytesDone:  len(data.Data),
			}
		} else {
			if _, ok := as.Uploaders.data[data.Uid][data.ID]; !ok {
				as.Uploaders.data[data.Uid][data.ID] = Upload{
					Index:      0,
					MsgID:      data.ID,
					LastUpdate: time.Now(),
					BytesDone:  len(data.Data),
				}
			} else {
				i = as.Uploaders.data[data.Uid][data.ID].Index + 1
				as.Uploaders.data[data.Uid][data.ID] = Upload{
					Index:      i,
					MsgID:      data.ID,
					LastUpdate: time.Now(),
					BytesDone:  as.Uploaders.data[data.Uid][data.ID].BytesDone + len(data.Data),
				}
			}
		}

		ratio := (float32(as.Uploaders.data[data.Uid][data.ID].BytesDone)) / float32(size)

		as.Uploaders.mutex.Unlock()

		if insertStmt, err := conn.Conn().Prepare(data.Ctx, "attachment_server_chunk_loop_insert_stmt", fmt.Sprintf("INSERT INTO %v (bytes,message_id,chunk_index) VALUES($1,$2,$3)", chunkTable)); err != nil {
			errored(err, conn)
			continue
		} else {
			if _, err = conn.Conn().Exec(data.Ctx, insertStmt.Name, data.Data, data.ID, i); err != nil {
				errored(err, conn)
				continue
			}
		}

		if updateMetaStmt, err := conn.Conn().Prepare(data.Ctx, "attachment_server_chunk_loop_update_ratio_stmt", fmt.Sprintf("UPDATE %v SET ratio = $1 WHERE message_id = $2", metaTable)); err != nil {
			errored(err, conn)
			continue
		} else {
			if _, err = conn.Conn().Exec(data.Ctx, updateMetaStmt.Name, ratio, data.ID); err != nil {
				errored(err, conn)
				continue
			}
		}

		if strings.HasPrefix(metaTable, "room") {
			var room_channel_id string
			if selectChannelStmt, err := conn.Conn().Prepare(data.Ctx, "attachment_server_chunk_loop_select_channel_id_stmt", "SELECT room_channel_id FROM room_messages WHERE id = $1"); err != nil {
				errored(err, conn)
				continue
			} else {
				if err = conn.Conn().QueryRow(data.Ctx, selectChannelStmt.Name, data.ID).Scan(&room_channel_id); err != nil {
					errored(err, conn)
					continue
				}
				sub := fmt.Sprintf("channel:%v", room_channel_id)
				ss.SendDataToSub <- socketServer.SubscriptionMessageData{
					SubName: sub,
					Data: socketMessages.AttachmentProgress{
						Ratio:  ratio,
						Failed: false,
						MsgID:  data.ID,
					},
					MessageType: "ATTACHMENT_PROGRESS",
				}
			}
		}
		if strings.HasPrefix(metaTable, "direct") {
			var recipient_id string
			if selectRecipientStmt, err := conn.Conn().Prepare(data.Ctx, "attachment_server_chunk_loop_select_recipient_id_stmt", "SELECT recipient_id FROM direct_messages WHERE id = $1"); err != nil {
				errored(err, conn)
				continue
			} else {
				if err = conn.Conn().QueryRow(data.Ctx, selectRecipientStmt.Name, data.ID).Scan(&recipient_id); err != nil {
					errored(err, conn)
					continue
				}
				ss.SendDataToUsers <- socketServer.UsersMessageData{
					Uids: []string{data.Uid, recipient_id},
					Data: socketMessages.AttachmentProgress{
						Ratio:  ratio,
						Failed: false,
						MsgID:  data.ID,
					},
					MessageType: "ATTACHMENT_PROGRESS",
				}
			}
		}

		conn.Release()

		data.RecvChan <- true
	}
}

func deleteAttachment(ss *socketServer.SocketServer, as *AttachmentServer, db *pgxpool.Pool) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if failCount < 10 {
				failCount++
				log.Println("Recovered from panic in attachment server handle chunk loop:", r)
			} else {
				log.Println("Attachment server panic count exceeded maximum retries")
			}
			failCount++
		}()

		id := <-as.DeleteChan

		errored := func(err error, conn *pgxpool.Conn) {
			log.Printf("Error in attachment server fail attachment loop:%v\n", err)
			conn.Release()
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		conn, err := db.Acquire(ctx)
		if err != nil {
			errored(err, conn)
			continue
		}

		_, chunkTable, err := getTableNames(conn, ctx, id)
		if err != nil {
			errored(err, conn)
			continue
		}

		if deleteStmt, err := conn.Conn().Prepare(ctx, "attachment_server_delete_chunk_loop_stmt", fmt.Sprintf("DELETE FROM %v WHERE message_id = $1", chunkTable)); err != nil {
			errored(err, conn)
			continue
		} else {
			if _, err = conn.Conn().Exec(ctx, deleteStmt.Name, id); err != nil {
				errored(err, conn)
				continue
			}
		}

		conn.Release()
	}
}

func failAttachment(ss *socketServer.SocketServer, as *AttachmentServer, db *pgxpool.Pool) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if failCount < 10 {
				failCount++
				log.Println("Recovered from panic in attachment server handle chunk loop:", r)
			} else {
				log.Println("Attachment server panic count exceeded maximum retries")
			}
			failCount++
		}()

		id := <-as.FailChan

		errored := func(err error, conn *pgxpool.Conn) {
			log.Printf("Error in attachment server fail attachment loop:%v\n", err)
			conn.Release()
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		conn, err := db.Acquire(ctx)
		if err != nil {
			errored(err, conn)
			continue
		}

		metaTable, _, err := getTableNames(conn, ctx, id)
		if err != nil {
			errored(err, conn)
			continue
		}

		if updateStmt, err := conn.Conn().Prepare(ctx, "attachment_server_fail_attachment_metadata_update_stmt", fmt.Sprintf("UPDATE %v SET failed = TRUE WHERE message_id = $1", metaTable)); err != nil {
			errored(err, conn)
			continue
		} else {
			if _, err := conn.Exec(ctx, updateStmt.Name, id); err != nil {
				errored(err, conn)
				continue
			}
		}

		conn.Release()

		as.DeleteChan <- id
	}
}

func getTableNames(conn *pgxpool.Conn, ctx context.Context, id string) (metaTable string, chunkTable string, err error) {
	var isDirectMessage, isRoomMsg bool
	if selectDirectMessage, err := conn.Conn().Prepare(ctx, "attachment_server_chunk_loop_select_direct_messages_stmt", "SELECT EXISTS(SELECT 1 FROM direct_messages WHERE id = $1)"); err != nil {
		return "", "", err
	} else {
		if err = conn.Conn().QueryRow(ctx, selectDirectMessage.Name, id).Scan(&isDirectMessage); err != nil {
			return "", "", err
		}
	}
	if selectRoomMessage, err := conn.Conn().Prepare(ctx, "attachment_server_chunk_loop_select_room_messages_stmt", "SELECT EXISTS(SELECT 1 FROM room_messages WHERE id = $1)"); err != nil {
		return "", "", err
	} else {
		if err = conn.Conn().QueryRow(ctx, selectRoomMessage.Name, id).Scan(&isRoomMsg); err != nil {
			return "", "", err
		}
	}
	if isDirectMessage {
		return "direct_message_attachment_metadata", "direct_message_attachment_chunks", nil
	}
	if isRoomMsg {
		return "room_message_attachment_metadata", "room_message_attachment_chunks", nil
	}
	return "", "", fmt.Errorf("Message not found in either table")
}
