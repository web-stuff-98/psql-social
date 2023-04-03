package attachmentserver

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
)

/*
	Takes in bytes for attachments from the websocket connection.
	The first 36 bytes of every message will be the message ID
	the attachment is associated with.

	The binary data will be buffered until it reaches the chunk
	size, then the chunk will be saved to the database.
*/

var chunkSize = 2 * 1024 * 1024

type AttachmentServer struct {
	Uploaders Uploaders

	// first 36 bytes are msg id
	ChunkChan chan ChunkData
	FailChan  chan string
}

/* --------- MUTEX PROTECTED --------- */
type Uploaders struct {
	// outer map is uid, inner map is msgId
	data  map[string]map[string]Upload
	mutex sync.RWMutex
}

/* --------- OTHER --------- */
type Upload struct {
	ChunksDone  int
	CurrentData []byte
	NextUUID    string
	IsRoomMsg   bool
}

type ChunkData struct {
	// first 36 bytes are the msg id
	Data []byte
	Uid  string
}

func Init(ss *socketServer.SocketServer, db *pgxpool.Pool) *AttachmentServer {
	as := &AttachmentServer{
		Uploaders: Uploaders{
			data: make(map[string]map[string]Upload),
		},

		ChunkChan: make(chan ChunkData),
		FailChan:  make(chan string),
	}
	go runServer(as, ss, db)
	return as
}

func runServer(as *AttachmentServer, ss *socketServer.SocketServer, db *pgxpool.Pool) {
	go processChunks(as, ss, db)
}

func processChunks(as *AttachmentServer, ss *socketServer.SocketServer, db *pgxpool.Pool) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				if failCount < 10 {
					log.Println("Recovered from panic in attachment server process chunks loop:", r)
				} else {
					log.Println("Panic recovery count in attachment server loop exceeded maximum")
					return
				}
				failCount++
			}
		}()

		c := <-as.ChunkChan

		msgId := string(c.Data[:36])
		log.Printf("Receiving chunks for msg:%v\n", msgId)

		checkAuthOk := func() (bool, error) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
			defer cancel()
			conn, err := db.Acquire(ctx)
			if err != nil {
				return false, fmt.Errorf("Failed to acquire connection in attachmentServer chunk loop:%v", err)
			}
			defer ctx.Done()
			selectRoomMsg, err := conn.Conn().Prepare(ctx, "attachment_chunk_get_msg_type_stmt", "SELECT EXISTS(SELECT 1 FROM room_messages WHERE id = $1)")
			var isRoomMsg bool
			if err = conn.Conn().QueryRow(ctx, selectRoomMsg.Name, msgId).Scan(&isRoomMsg); err != nil {
				return false, fmt.Errorf("Error in attachment chunk loop get room message exists statement:%v", err)
			}
			selectDirectMsg, err := conn.Conn().Prepare(ctx, "attachment_chunk_get_msg_type_stmt", "SELECT EXISTS(SELECT 1 FROM direct_messages WHERE id = $1)")
			var isDirectMessage bool
			if err = conn.Conn().QueryRow(ctx, selectDirectMsg.Name, msgId).Scan(&isDirectMessage); err != nil {
				return false, fmt.Errorf("Error in attachment chunk loop get direct message exists statement:%v", err)
			}
			if !isDirectMessage && !isRoomMsg {
				return false, fmt.Errorf("Error in attachment chunk loop, message could not be found")
			}
			// now no need to prepare statement. The ID is definitely clean, because it matched an existing uuid
			var author_id string
			if isDirectMessage {
				if err = db.QueryRow(ctx, "SELECT author_id FROM direct_messages WHERE id = $1;", msgId).Scan(&author_id); err != nil {
					return false, fmt.Errorf("Error in attachment chunk loop retrieving author_id:%v", err)
				}
			}
			if isRoomMsg {
				if err = db.QueryRow(ctx, "SELECT author_id FROM room_messages WHERE id = $1;", msgId).Scan(&author_id); err != nil {
					return false, fmt.Errorf("Error in attachment chunk loop retrieving author_id:%v", err)
				}
			}
			if author_id != c.Uid {
				return false, fmt.Errorf("A user is trying to upload data using another users ID: (culprit:%v, target:%v)\n", c.Uid, author_id)
			}
			return isRoomMsg, nil
		}

		as.Uploaders.mutex.Lock()
		if _, ok := as.Uploaders.data[c.Uid]; ok {
			if data, ok := as.Uploaders.data[c.Uid][msgId]; ok {
				var new = append(data.CurrentData, c.Data[36:]...)
				if len(new) >= chunkSize {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
					defer cancel()
					fit := new[:chunkSize]
					remaining := new[chunkSize:]
					nextChunkId := uuid.New().String()
					var table string
					if data.IsRoomMsg {
						table = "room_message_attachment_chunks"
					} else {
						table = "direct_message_attachment_chunks"
					}
					// no need to prepare statement, msg id was predetermined to be clean
					if _, err := db.Exec(ctx, `INSERT INTO "$1" (bytes,message_id,next_chunk,id) VALUES($2,$3,$4,$5)`, table, fit, msgId, nextChunkId, data.NextUUID); err != nil {
						as.FailChan <- msgId
						as.Uploaders.mutex.Unlock()
						continue
					}
					as.Uploaders.data[c.Uid][msgId] = Upload{
						ChunksDone:  data.ChunksDone + 1,
						CurrentData: remaining,
						NextUUID:    nextChunkId,
						IsRoomMsg:   data.IsRoomMsg,
					}
					ctx.Done()
				} else {
					if isRoomMsg, err := checkAuthOk(); err != nil {
						log.Println(err)
						as.FailChan <- msgId
						as.Uploaders.mutex.Unlock()
						continue
					} else {
						as.Uploaders.data[c.Uid][msgId] = Upload{
							ChunksDone:  data.ChunksDone + 1,
							CurrentData: new,
							NextUUID:    "",
							IsRoomMsg:   isRoomMsg,
						}
					}
				}
			}
		} else {
			if isRoomMsg, err := checkAuthOk(); err != nil {
				log.Println(err)
				as.FailChan <- msgId
				as.Uploaders.mutex.Unlock()
				continue
			} else {
				uploads := make(map[string]Upload)
				uploads[msgId] = Upload{
					ChunksDone:  0,
					CurrentData: c.Data[36:],
					NextUUID:    "",
					IsRoomMsg:   isRoomMsg,
				}
				as.Uploaders.data[c.Uid] = uploads
			}
		}
		as.Uploaders.mutex.Unlock()
	}
}
