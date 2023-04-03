package channelRTCserver

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	socketmessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
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
	RecvChan  chan map[string]struct{}
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
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in cRTCs join webRTC loop:", r)
				if failCount < 10 {
					go joinWebRTCChannel(ss, cRTCs, db)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-cRTCs.JoinChannelRTC

		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		conn, err := db.Acquire(ctx)
		if err != nil {
			log.Println("Error acquiring pgxpool connection:", err)
			continue
		}
		defer conn.Release()

		cRTCs.ChannelConnections.mutex.RLock()
		connectionInfo := &Connection{
			UserMediaStreamID: data.UserMediaStreamID,
			UserMediaVid:      data.UserMediaVid,
			DisplayMediaVid:   data.DisplayMediaVid,
		}
		if channelUsers, ok := cRTCs.ChannelConnections.data[data.ChannelID]; ok {
			// Send back uids of other users in the channel WebRTC chat
			users := []socketmessages.ChannelWebRTCOutUser{}
			for id, user := range channelUsers {
				if id != data.Uid {
					users = append(users, socketmessages.ChannelWebRTCOutUser{
						Uid:               id,
						UserMediaStreamID: user.UserMediaStreamID,
						UserMediaVid:      user.UserMediaVid,
						DisplayMediaVid:   user.DisplayMediaVid,
					})
				}
			}
			if _, ok := channelUsers[data.Uid]; !ok {
				cRTCs.ChannelConnections.mutex.RUnlock()
				// Add the user to the channel map
				cRTCs.ChannelConnections.mutex.Lock()
				cRTCs.ChannelConnections.data[data.ChannelID][data.Uid] = *connectionInfo
				cRTCs.ChannelConnections.mutex.Unlock()

				selectChannelStmt, err := conn.Conn().Prepare(ctx, "cRTCs_join_channel_select_stmt", "SELECT room_id FROM room_channels WHERE id = $1")
				if err != nil {
					log.Println("Error in join WebRTC select channel prepare statement:", err)
					continue
				}
				var room_id string
				if err = conn.QueryRow(ctx, selectChannelStmt.Name, data.ChannelID).Scan(&room_id); err != nil {
					log.Println("Error in join WebRTC select channel statement:", err)
				}

				ss.SendDataToSub <- socketServer.SubscriptionMessageData{
					SubName: fmt.Sprintf("channel:%v", data.ChannelID),
					Data: socketmessages.RoomChannelWebRTCUserJoinedLeft{
						ChannelID: data.ChannelID,
						Uid:       data.Uid,
					},
					MessageType: "ROOM_CHANNEL_WEBRTC_USER_JOINED",
				}
			} else {
				cRTCs.ChannelConnections.mutex.RUnlock()
			}
			ss.SendDataToUser <- socketServer.UserMessageData{
				Uid: data.Uid,
				Data: socketmessages.ChannelWebRTCAllUsers{
					Users: users,
				},
				MessageType: "CHANNEL_WEBRTC_ALL_USERS",
			}
		} else {
			// Create the channel data and add the user
			cRTCs.ChannelConnections.mutex.RUnlock()
			cRTCs.ChannelConnections.mutex.Lock()
			cRTCs.ChannelConnections.data[data.ChannelID] = make(map[string]Connection)
			cRTCs.ChannelConnections.data[data.ChannelID][data.Uid] = *connectionInfo
			cRTCs.ChannelConnections.mutex.Unlock()
			// Send back empty list of uids, since the user is the only one in the channel
			ss.SendDataToUser <- socketServer.UserMessageData{
				MessageType: "CHANNEL_WEBRTC_ALL_USERS",
				Data: socketmessages.ChannelWebRTCAllUsers{
					Users: []socketmessages.ChannelWebRTCOutUser{},
				},
				Uid: data.Uid,
			}
		}
	}
}

func leaveWebRTCChannel(ss *socketServer.SocketServer, cRTCs *ChannelRTCServer, db *pgxpool.Pool) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in cRTCs leave webRTC loop:", r)
				if failCount < 10 {
					go leaveWebRTCChannel(ss, cRTCs, db)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-cRTCs.LeaveChannelRTC

		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		conn, err := db.Acquire(ctx)
		if err != nil {
			log.Println("Error acquiring pgxpool connection:", err)
			continue
		}
		defer conn.Release()

		cRTCs.ChannelConnections.mutex.RLock()
		if channelUids, ok := cRTCs.ChannelConnections.data[data.ChannelID]; ok {
			if _, ok := channelUids[data.Uid]; ok {
				uids := []string{}
				for uid := range cRTCs.ChannelConnections.data[data.ChannelID] {
					uids = append(uids, uid)
				}
				cRTCs.ChannelConnections.mutex.RUnlock()
				cRTCs.ChannelConnections.mutex.Lock()
				delete(cRTCs.ChannelConnections.data[data.ChannelID], data.Uid)
				cRTCs.ChannelConnections.mutex.Unlock()

				selectChannelStmt, err := conn.Conn().Prepare(ctx, "cRTCs_leave_channel_select_stmt", "SELECT room_id FROM room_channels WHERE id = $1")
				if err != nil {
					log.Println("Error in leave WebRTC select channel prepare statement:", err)
					continue
				}
				var room_id string
				if err = conn.QueryRow(ctx, selectChannelStmt.Name, data.ChannelID).Scan(&room_id); err != nil {
					log.Println("Error in leave WebRTC select channel statement:", err)
				}

				ss.SendDataToSub <- socketServer.SubscriptionMessageData{
					SubName: fmt.Sprintf("channel:%v", data.ChannelID),
					Data: socketmessages.RoomChannelWebRTCUserJoinedLeft{
						ChannelID: data.ChannelID,
						Uid:       data.Uid,
					},
					MessageType: "ROOM_CHANNEL_WEBRTC_USER_LEFT",
				}
				ss.SendDataToUsers <- socketServer.UsersMessageData{
					Uids:        uids,
					MessageType: "CHANNEL_WEBRTC_LEFT",
					Data: socketmessages.ChannelWebRTCUserLeft{
						Uid: data.Uid,
					},
				}
				continue
			} else {
				cRTCs.ChannelConnections.mutex.RUnlock()
			}
		} else {
			cRTCs.ChannelConnections.mutex.RUnlock()
		}
	}
}

func sendWebRTCSignals(ss *socketServer.SocketServer, cRTCs *ChannelRTCServer, db *pgxpool.Pool) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in cRTCs send webRTC signal loop:", r)
				if failCount < 10 {
					go sendWebRTCSignals(ss, cRTCs, db)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-cRTCs.SignalRTC

		ss.SendDataToUser <- socketServer.UserMessageData{
			MessageType: "CHANNEL_WEBRTC_JOINED",
			Uid:         data.ToUid,
			Data: socketmessages.ChannelWebRTCUserJoined{
				CallerUID:         data.Uid,
				Signal:            data.Signal,
				UserMediaStreamID: data.UserMediaStreamID,
				UserMediaVid:      data.UserMediaVid,
				DisplayMediaVid:   data.DisplayMediaVid,
			},
		}
	}
}

func returningWebRTCSignals(ss *socketServer.SocketServer, cRTCs *ChannelRTCServer, db *pgxpool.Pool) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in cRTCs returning webRTC signal loop:", r)
				if failCount < 10 {
					go returningWebRTCSignals(ss, cRTCs, db)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-cRTCs.ReturnSignalRTC

		ss.SendDataToUser <- socketServer.UserMessageData{
			MessageType: "CHANNEL_WEBRTC_RETURN_SIGNAL_OUT",
			Uid:         data.CallerID,
			Data: socketmessages.ChannelWebRTCReturnSignal{
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
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in cRTCs retrieve uids loop:", r)
				if failCount < 10 {
					go retrieveChannelUids(ss, cRTCs)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-cRTCs.GetChannelUids
		cRTCs.ChannelConnections.mutex.RLock()
		if channelConnections, ok := cRTCs.ChannelConnections.data[data.ChannelID]; ok {
			channelUids := make(map[string]struct{})
			for oi := range channelConnections {
				channelUids[oi] = struct{}{}
			}
			cRTCs.ChannelConnections.mutex.RUnlock()
			data.RecvChan <- channelUids
		} else {
			cRTCs.ChannelConnections.mutex.RUnlock()
			data.RecvChan <- make(map[string]struct{})
		}
	}
}

func updateMediaOptions(ss *socketServer.SocketServer, cRTCs *ChannelRTCServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in cRTCs update media options loop:", r)
				if failCount < 10 {
					go updateMediaOptions(ss, cRTCs)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

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
				Data: socketmessages.UpdateMediaOptions{
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
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in cRTCs disconnect loop:", r)
				if failCount < 10 {
					go socketDisconnect(ss, cRTCs, db, dc)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		uid := <-dc

		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		conn, err := db.Acquire(ctx)
		if err != nil {
			log.Println("Error acquiring pgxpool connection:", err)
			continue
		}
		defer conn.Release()

		cRTCs.ChannelConnections.mutex.Lock()
		for channelId, uids := range cRTCs.ChannelConnections.data {
			for oi := range uids {
				if oi == uid {
					delete(cRTCs.ChannelConnections.data[channelId], uid)
					uidsInWebRTC := []string{}
					for uid := range cRTCs.ChannelConnections.data[channelId] {
						uidsInWebRTC = append(uidsInWebRTC, uid)
					}

					selectChannelStmt, err := conn.Conn().Prepare(ctx, "cRTCs_socket_disconnect_select_stmt", "SELECT room_id FROM room_channels WHERE id = $1")
					if err != nil {
						log.Println("Error in cRTCs socket disconnect event select channel prepare statement:", err)
						cRTCs.ChannelConnections.mutex.Unlock()
						continue
					}
					var room_id string
					if err = conn.QueryRow(ctx, selectChannelStmt.Name, channelId).Scan(&room_id); err != nil {
						log.Println("Error in cRTCs socket disconnect event select channel statement:", err)
						cRTCs.ChannelConnections.mutex.Unlock()
						continue
					}

					recvChan := make(chan map[string]struct{})
					ss.GetSubscriptionUids <- socketServer.GetSubscriptionUids{
						RecvChan: recvChan,
						SubName:  fmt.Sprintf("channel:%v", channelId),
					}
					uidsMap := <-recvChan

					uids := []string{}
					for k := range uidsMap {
						uids = append(uids, k)
					}

					ss.SendDataToUsers <- socketServer.UsersMessageData{
						Uids: uids,
						Data: socketmessages.RoomChannelWebRTCUserJoinedLeft{
							Uid:       uid,
							ChannelID: channelId,
						},
						MessageType: "ROOM_CHANNEL_WEBRTC_LEFT",
					}
					ss.SendDataToUsers <- socketServer.UsersMessageData{
						Uids: uidsInWebRTC,
						Data: socketmessages.ChannelWebRTCUserLeft{
							Uid: uid,
						},
					}
					break
				}
			}
		}
		cRTCs.ChannelConnections.mutex.Unlock()
	}
}
