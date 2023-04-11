package channelRTCserver

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	socketMessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	socketServer "github.com/web-stuff-98/psql-social/pkg/socketServer"
)

/*
	This is for WebRTC chat in room channels
*/

type ChannelRTCServer struct {
	// Mutex protected map for users joined in Channel WebRTC networks
	ChannelConnections ChannelConnections
	// Channel for joining WebRTC network
	JoinChannelRTC chan JoinChannel
	// Channel for leaving WebRTC network
	LeaveChannelRTC chan LeaveChannel
	// Channel for users sending WebRTC signal
	SignalRTC chan SignalRTC
	// Channel for users returning WebRTC signal
	ReturnSignalRTC chan ReturnSignalRTC
	// Channel for getting the IDs of users currently joined to a channel WebRTC network
	GetChannelUids chan GetChannelUids
	// Channel for updating media options
	UpdateMediaOptions chan UpdateMediaOptions
}

/* --------------- MUTEX PROTECTED MAPS --------------- */
type ChannelConnections struct {
	// Outer map is channel ID, inner map are UIDs of users in the channel
	data  map[string]map[string]Connection
	mutex sync.RWMutex
}

/* --------------- RECV CHANNEL STRUCTS --------------- */
type GetChannelUids struct {
	RecvChan  chan<- map[string]struct{}
	ChannelID string
}

/* --------------- OTHER MODELS --------------- */
type Connection struct {
	UserMediaStreamID string
	UserMediaVid      bool
	DisplayMediaVid   bool
}
type JoinChannel struct {
	Uid       string
	ChannelID string

	UserMediaStreamID string
	UserMediaVid      bool
	DisplayMediaVid   bool
}
type LeaveChannel struct {
	Uid       string
	ChannelID string
}
type SignalRTC struct {
	Signal string
	ToUid  string
	Uid    string

	UserMediaStreamID string
	UserMediaVid      bool
	DisplayMediaVid   bool
}
type ReturnSignalRTC struct {
	Signal   string
	CallerID string
	Uid      string

	UserMediaStreamID string
	UserMediaVid      bool
	DisplayMediaVid   bool
}
type UpdateMediaOptions struct {
	ChannelID string
	Uid       string

	UserMediaStreamID string
	UserMediaVid      bool
	DisplayMediaVid   bool
}

func Init(ss *socketServer.SocketServer, db *pgxpool.Pool, dc chan string) *ChannelRTCServer {
	cRTCs := &ChannelRTCServer{
		ChannelConnections: ChannelConnections{
			data: make(map[string]map[string]Connection),
		},
		JoinChannelRTC:     make(chan JoinChannel),
		LeaveChannelRTC:    make(chan LeaveChannel),
		SignalRTC:          make(chan SignalRTC),
		ReturnSignalRTC:    make(chan ReturnSignalRTC),
		GetChannelUids:     make(chan GetChannelUids),
		UpdateMediaOptions: make(chan UpdateMediaOptions),
	}
	runServer(ss, cRTCs, db, dc)
	return cRTCs
}

func runServer(ss *socketServer.SocketServer, cRTCs *ChannelRTCServer, db *pgxpool.Pool, dc chan string) {
	go joinWebRTCChannel(ss, cRTCs, db)
	go leaveWebRTCChannel(ss, cRTCs, db)
	go sendWebRTCSignals(ss, cRTCs, db)
	go returningWebRTCSignals(ss, cRTCs, db)
	go retrieveChannelUids(ss, cRTCs)
	go socketDisconnect(ss, cRTCs, db, dc)
	go updateMediaOptions(ss, cRTCs)
}

func joinWebRTCChannel(ss *socketServer.SocketServer, cRTCs *ChannelRTCServer, db *pgxpool.Pool) {
	for {
		data := <-cRTCs.JoinChannelRTC

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
		defer cancel()

		conn, err := db.Acquire(ctx)
		if err != nil {
			log.Println("Error acquiring pgxpool connection:", err)
			continue
		}

		cRTCs.ChannelConnections.mutex.Lock()

		connectionInfo := &Connection{
			UserMediaStreamID: data.UserMediaStreamID,
			UserMediaVid:      data.UserMediaVid,
			DisplayMediaVid:   data.DisplayMediaVid,
		}

		if channelUsers, ok := cRTCs.ChannelConnections.data[data.ChannelID]; ok {
			// Send back uids of other users in the channel WebRTC chat
			users := []socketMessages.ChannelWebRTCOutUser{}
			for id, user := range channelUsers {
				if id != data.Uid {
					users = append(users, socketMessages.ChannelWebRTCOutUser{
						Uid:               id,
						UserMediaStreamID: user.UserMediaStreamID,
						UserMediaVid:      user.UserMediaVid,
						DisplayMediaVid:   user.DisplayMediaVid,
					})
				}
			}
			if _, ok := channelUsers[data.Uid]; !ok {
				// Add the user to the channel map
				cRTCs.ChannelConnections.data[data.ChannelID][data.Uid] = *connectionInfo

				selectChannelStmt, err := conn.Conn().Prepare(ctx, "cRTCs_join_channel_select_stmt", "SELECT room_id FROM room_channels WHERE id = $1;")
				if err != nil {
					log.Println("Error in join WebRTC select channel prepare statement:", err)
					cRTCs.ChannelConnections.mutex.Unlock()
					conn.Release()
					continue
				}
				var room_id string
				if err = conn.QueryRow(ctx, selectChannelStmt.Name, data.ChannelID).Scan(&room_id); err != nil {
					log.Println("Error in join WebRTC select channel statement:", err)
				}

				ss.SendDataToSub <- socketServer.SubscriptionMessageData{
					SubName: fmt.Sprintf("channel:%v", data.ChannelID),
					Data: socketMessages.RoomChannelWebRTCUserJoinedLeft{
						ChannelID: data.ChannelID,
						Uid:       data.Uid,
					},
					MessageType: "ROOM_CHANNEL_WEBRTC_USER_JOINED",
				}
			}
			ss.SendDataToUser <- socketServer.UserMessageData{
				Uid: data.Uid,
				Data: socketMessages.ChannelWebRTCAllUsers{
					Users: users,
				},
				MessageType: "CHANNEL_WEBRTC_ALL_USERS",
			}
		} else {
			// Create the channel data and add the user
			cRTCs.ChannelConnections.data[data.ChannelID] = make(map[string]Connection)
			cRTCs.ChannelConnections.data[data.ChannelID][data.Uid] = *connectionInfo
			// Send back empty list of uids, since the user is the only one in the channel
			ss.SendDataToUser <- socketServer.UserMessageData{
				MessageType: "CHANNEL_WEBRTC_ALL_USERS",
				Data: socketMessages.ChannelWebRTCAllUsers{
					Users: []socketMessages.ChannelWebRTCOutUser{},
				},
				Uid: data.Uid,
			}
		}

		conn.Release()

		cRTCs.ChannelConnections.mutex.Unlock()
	}
}

func leaveWebRTCChannel(ss *socketServer.SocketServer, cRTCs *ChannelRTCServer, db *pgxpool.Pool) {
	for {
		data := <-cRTCs.LeaveChannelRTC

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
		defer cancel()

		conn, err := db.Acquire(ctx)
		if err != nil {
			log.Println("Error acquiring pgxpool connection:", err)
			continue
		}

		cRTCs.ChannelConnections.mutex.Lock()

		if channelUids, ok := cRTCs.ChannelConnections.data[data.ChannelID]; ok {
			if _, ok := channelUids[data.Uid]; ok {
				uids := []string{}
				for uid := range cRTCs.ChannelConnections.data[data.ChannelID] {
					if uid != data.Uid {
						uids = append(uids, uid)
					}
				}
				delete(cRTCs.ChannelConnections.data[data.ChannelID], data.Uid)

				selectChannelStmt, err := conn.Conn().Prepare(ctx, "cRTCs_leave_channel_select_stmt", "SELECT room_id FROM room_channels WHERE id = $1;")
				if err != nil {
					log.Println("Error in leave WebRTC select channel prepare statement:", err)

					cRTCs.ChannelConnections.mutex.Unlock()

					conn.Release()
					continue
				}
				var room_id string
				if err = conn.QueryRow(ctx, selectChannelStmt.Name, data.ChannelID).Scan(&room_id); err != nil {
					log.Println("Error in leave WebRTC select channel statement:", err)
				}

				conn.Release()

				cRTCs.ChannelConnections.mutex.Unlock()

				ss.SendDataToSub <- socketServer.SubscriptionMessageData{
					SubName: fmt.Sprintf("channel:%v", data.ChannelID),
					Data: socketMessages.RoomChannelWebRTCUserJoinedLeft{
						ChannelID: data.ChannelID,
						Uid:       data.Uid,
					},
					MessageType: "ROOM_CHANNEL_WEBRTC_USER_LEFT",
				}
				ss.SendDataToUsers <- socketServer.UsersMessageData{
					Uids:        uids,
					MessageType: "CHANNEL_WEBRTC_LEFT",
					Data: socketMessages.ChannelWebRTCUserLeft{
						Uid: data.Uid,
					},
				}
				continue
			}
		}

		cRTCs.ChannelConnections.mutex.Unlock()

		conn.Release()
	}
}

func sendWebRTCSignals(ss *socketServer.SocketServer, cRTCs *ChannelRTCServer, db *pgxpool.Pool) {
	for {
		data := <-cRTCs.SignalRTC

		ss.SendDataToUser <- socketServer.UserMessageData{
			MessageType: "CHANNEL_WEBRTC_JOINED",
			Uid:         data.ToUid,
			Data: socketMessages.ChannelWebRTCUserJoined{
				CallerID:          data.Uid,
				Signal:            data.Signal,
				UserMediaStreamID: data.UserMediaStreamID,
				UserMediaVid:      data.UserMediaVid,
				DisplayMediaVid:   data.DisplayMediaVid,
			},
		}
	}
}

func returningWebRTCSignals(ss *socketServer.SocketServer, cRTCs *ChannelRTCServer, db *pgxpool.Pool) {
	for {
		data := <-cRTCs.ReturnSignalRTC

		ss.SendDataToUser <- socketServer.UserMessageData{
			MessageType: "CHANNEL_WEBRTC_RETURN_SIGNAL_OUT",
			Uid:         data.CallerID,
			Data: socketMessages.ChannelWebRTCReturnSignal{
				Signal:            data.Signal,
				Uid:               data.Uid,
				UserMediaStreamID: data.UserMediaStreamID,
				UserMediaVid:      data.UserMediaVid,
				DisplayMediaVid:   data.DisplayMediaVid,
			},
		}
	}
}

func retrieveChannelUids(ss *socketServer.SocketServer, cRTCs *ChannelRTCServer) {
	for {
		data := <-cRTCs.GetChannelUids

		cRTCs.ChannelConnections.mutex.RLock()

		if channelConnections, ok := cRTCs.ChannelConnections.data[data.ChannelID]; ok {
			channelUids := make(map[string]struct{})
			for oi := range channelConnections {
				channelUids[oi] = struct{}{}
			}
			data.RecvChan <- channelUids

			cRTCs.ChannelConnections.mutex.RUnlock()
		} else {
			data.RecvChan <- make(map[string]struct{})

			cRTCs.ChannelConnections.mutex.RUnlock()
		}
	}
}

func updateMediaOptions(ss *socketServer.SocketServer, cRTCs *ChannelRTCServer) {
	for {
		data := <-cRTCs.UpdateMediaOptions

		cRTCs.ChannelConnections.mutex.RLock()

		if channelUids, ok := cRTCs.ChannelConnections.data[data.ChannelID]; ok {
			uids := []string{}
			for uid := range channelUids {
				if uid != data.Uid {
					uids = append(uids, uid)
				}
			}

			ss.SendDataToUsers <- socketServer.UsersMessageData{
				MessageType: "UPDATE_MEDIA_OPTIONS_OUT",
				Uids:        uids,
				Data: socketMessages.UpdateMediaOptions{
					UserMediaVid:      data.UserMediaVid,
					DisplayMediaVid:   data.DisplayMediaVid,
					UserMediaStreamID: data.UserMediaStreamID,
					Uid:               data.Uid,
				},
			}
		}

		cRTCs.ChannelConnections.mutex.RUnlock()
	}
}

func socketDisconnect(ss *socketServer.SocketServer, cRTCs *ChannelRTCServer, db *pgxpool.Pool, dc chan string) {
	for {
		uid := <-dc

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
		defer cancel()

		conn, err := db.Acquire(ctx)
		if err != nil {
			log.Println("Error acquiring pgxpool connection:", err)
			continue
		}

		cRTCs.ChannelConnections.mutex.Lock()

		for channelId, uids := range cRTCs.ChannelConnections.data {
			for oi := range uids {
				if oi == uid {
					delete(cRTCs.ChannelConnections.data[channelId], uid)
					uidsInWebRTC := []string{}
					for uid := range cRTCs.ChannelConnections.data[channelId] {
						uidsInWebRTC = append(uidsInWebRTC, uid)
					}

					selectChannelStmt, err := conn.Conn().Prepare(ctx, "cRTCs_socket_disconnect_select_stmt", "SELECT room_id FROM room_channels WHERE id = $1;")
					if err != nil {
						log.Println("Error in cRTCs socket disconnect event select channel prepare statement:", err)
						cRTCs.ChannelConnections.mutex.Unlock()
						conn.Release()
						continue
					}
					var room_id string
					if err = conn.QueryRow(ctx, selectChannelStmt.Name, channelId).Scan(&room_id); err != nil {
						log.Println("Error in cRTCs socket disconnect event select channel statement:", err)
						cRTCs.ChannelConnections.mutex.Unlock()
						conn.Release()
						continue
					}

					recvChan := make(chan map[string]struct{})
					ss.GetSubscriptionUids <- socketServer.GetSubscriptionUids{
						RecvChan: recvChan,
						SubName:  fmt.Sprintf("channel:%v", channelId),
					}
					uidsMap := <-recvChan

					close(recvChan)

					uids := []string{}
					for k := range uidsMap {
						uids = append(uids, k)
					}

					ss.SendDataToUsers <- socketServer.UsersMessageData{
						Uids: uids,
						Data: socketMessages.RoomChannelWebRTCUserJoinedLeft{
							Uid:       uid,
							ChannelID: channelId,
						},
						MessageType: "ROOM_CHANNEL_WEBRTC_LEFT",
					}
					ss.SendDataToUsers <- socketServer.UsersMessageData{
						Uids: uidsInWebRTC,
						Data: socketMessages.ChannelWebRTCUserLeft{
							Uid: uid,
						},
					}
					break
				}
			}
		}

		conn.Release()

		cRTCs.ChannelConnections.mutex.Unlock()
	}
}
