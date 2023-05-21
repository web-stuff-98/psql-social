package socketServer

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gofiber/websocket/v2"
	socketmessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
)

/*
It can only send JSON messages, in this form:
{
	event_type <- event name ("DIRECT_MESSAGE" for example)
	data       <- json message encoded from map[string]interface{}
}
*/

type SocketServer struct {
	ConnectionsByID ConnectionsByID
	ConnectionsByWs ConnectionsByWs
	Subscriptions   Subscriptions

	GetConnectionSubscriptions chan GetConnectionSubscriptions

	IsUserOnline chan IsUserOnline

	MessageLoop chan Message

	AttachmentServerRemoveUploaderChan chan string

	RegisterConn   chan ConnectionData
	UnregisterConn chan *websocket.Conn

	CloseConnChan chan string

	SendDataToUser  chan UserMessageData
	SendDataToUsers chan UsersMessageData

	JoinSubscriptionByWs  chan RegisterUnregisterSubsConnWs
	LeaveSubscriptionByWs chan RegisterUnregisterSubsConnWs

	SendDataToSub  chan SubscriptionMessageData
	SendDataToSubs chan SubscriptionsMessageData

	GetSubscriptionUids chan GetSubscriptionUids

	GetConnection chan GetConnection
}

/* ------ INTERNAL MUTEX PROTECTED MAPS ------ */

type ConnectionsByID struct {
	data  map[string]*websocket.Conn
	mutex sync.RWMutex
}

type ConnectionsByWs struct {
	data  map[*websocket.Conn]string
	mutex sync.RWMutex
}

type Subscriptions struct {
	data  map[string]map[string]struct{}
	mutex sync.RWMutex
}

/* ------ RECV CHAN STRUCTS ------ */

type GetConnectionSubscriptions struct {
	RecvChan chan map[string]struct{}
	Conn     *websocket.Conn
}

type GetSubscriptionUids struct {
	RecvChan chan map[string]struct{}
	SubName  string
}

type IsUserOnline struct {
	RecvChan chan bool
	Uid      string
}

type GetConnection struct {
	RecvChan chan *websocket.Conn
	Uid      string
}

/* ------ GENERAL STRUCTS USED INTERNALLY AND EXTERNALLY ------ */

type Message struct {
	Conn *websocket.Conn
	Data []byte
}

type ConnectionData struct {
	Uid  string
	Conn *websocket.Conn
}

type UserMessageData struct {
	Data        interface{}
	Uid         string
	MessageType string
}

type UsersMessageData struct {
	Data        interface{}
	Uids        []string
	MessageType string
}

type ConnMessageData struct {
	Data        interface{}
	Conn        *websocket.Conn
	MessageType string
}

type RegisterUnregisterSubsConnWs struct {
	Conn    *websocket.Conn
	SubName string
}

type SubscriptionMessageData struct {
	SubName     string
	MessageType string
	Data        interface{}
}

type SubscriptionsMessageData struct {
	SubNames    []string
	MessageType string
	Data        interface{}
}

func Init(csdc chan string, cRTCsdc chan string, udlcdc chan string, udludc chan string) *SocketServer {
	ss := &SocketServer{
		ConnectionsByID: ConnectionsByID{
			data: make(map[string]*websocket.Conn),
		},
		ConnectionsByWs: ConnectionsByWs{
			data: make(map[*websocket.Conn]string),
		},
		Subscriptions: Subscriptions{
			data: map[string]map[string]struct{}{},
		},

		GetConnectionSubscriptions: make(chan GetConnectionSubscriptions),

		IsUserOnline: make(chan IsUserOnline),

		MessageLoop: make(chan Message),

		AttachmentServerRemoveUploaderChan: make(chan string),

		RegisterConn:   make(chan ConnectionData),
		UnregisterConn: make(chan *websocket.Conn),

		CloseConnChan: make(chan string),

		SendDataToUser:  make(chan UserMessageData),
		SendDataToUsers: make(chan UsersMessageData),

		JoinSubscriptionByWs:  make(chan RegisterUnregisterSubsConnWs),
		LeaveSubscriptionByWs: make(chan RegisterUnregisterSubsConnWs),

		SendDataToSub:  make(chan SubscriptionMessageData),
		SendDataToSubs: make(chan SubscriptionsMessageData),

		GetSubscriptionUids: make(chan GetSubscriptionUids),

		GetConnection: make(chan GetConnection),
	}
	go runServer(ss, csdc, cRTCsdc, udlcdc, udludc)
	return ss
}

func runServer(ss *SocketServer, csdc chan string, cRTCsdc chan string, udlcdc chan string, udludc chan string) {
	go connection(ss, udlcdc)
	go disconnect(ss, csdc, cRTCsdc, udludc)
	go checkUserOnline(ss)
	go closeConn(ss)
	go messageLoop(ss)
	go sendUserData(ss)
	go sendUsersData(ss)
	go joinSubsByWs(ss)
	go leaveSubByWs(ss)
	go sendSubData(ss)
	go sendSubsData(ss)
	go getConnSubscriptions(ss)
	go getSubscriptionUids(ss)
	go getConnection(ss)
}

func getConnection(ss *SocketServer) {
	for {
		data := <-ss.GetConnection

		ss.ConnectionsByID.mutex.RLock()

		if c, ok := ss.ConnectionsByID.data[data.Uid]; ok {
			ss.ConnectionsByID.mutex.RUnlock()

			data.RecvChan <- c
		} else {
			ss.ConnectionsByID.mutex.RUnlock()

			data.RecvChan <- nil
		}
	}
}

func closeConn(ss *SocketServer) {
	for {
		uid := <-ss.CloseConnChan

		ss.ConnectionsByID.mutex.RLock()

		if conn, ok := ss.ConnectionsByID.data[uid]; ok {
			ss.ConnectionsByID.mutex.RUnlock()

			ss.UnregisterConn <- conn
		} else {
			ss.ConnectionsByID.mutex.RUnlock()
		}
	}
}

func WriteMessage(t string, m interface{}, c *websocket.Conn, ss *SocketServer) {
	withType := make(map[string]interface{})
	withType["event_type"] = t
	withType["data"] = m

	if c == nil {
		return
	}

	if b, err := json.Marshal(withType); err == nil {
		ss.MessageLoop <- Message{
			Conn: c,
			Data: b,
		}
	}
}

func connection(ss *SocketServer, udlcdc chan string) {
	for {
		data := <-ss.RegisterConn

		ss.ConnectionsByID.mutex.Lock()
		ss.ConnectionsByWs.mutex.Lock()

		ss.ConnectionsByID.data[data.Uid] = data.Conn
		ss.ConnectionsByWs.data[data.Conn] = data.Uid

		ss.ConnectionsByWs.mutex.Unlock()
		ss.ConnectionsByID.mutex.Unlock()

		udlcdc <- data.Uid

		changeData := make(map[string]interface{})
		changeData["ID"] = data.Uid
		changeData["online"] = true
		ss.SendDataToSub <- SubscriptionMessageData{
			SubName: fmt.Sprintf("user:%v", data.Uid),
			Data: socketmessages.ChangeEvent{
				Type: "UPDATE",
				Data: changeData,
			},
			MessageType: "CHANGE",
		}
	}
}

func disconnect(ss *SocketServer, csdc chan string, cRTCsdc chan string, udludc chan string) {
	for {
		conn := <-ss.UnregisterConn

		ss.ConnectionsByWs.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()

		uid := ss.ConnectionsByWs.data[conn]

		delete(ss.ConnectionsByID.data, uid)
		delete(ss.ConnectionsByWs.data, conn)

		ss.ConnectionsByID.mutex.Unlock()
		ss.ConnectionsByWs.mutex.Unlock()

		csdc <- uid
		cRTCsdc <- uid
		udludc <- uid
		ss.AttachmentServerRemoveUploaderChan <- uid

		if conn != nil {
			conn.Close()
		}

		changeData := make(map[string]interface{})
		changeData["ID"] = uid
		changeData["online"] = false
		ss.SendDataToSub <- SubscriptionMessageData{
			SubName: fmt.Sprintf("user:%v", uid),
			Data: socketmessages.ChangeEvent{
				Type: "UPDATE",
				Data: changeData,
			},
			MessageType: "CHANGE",
		}
	}
}

func checkUserOnline(ss *SocketServer) {
	for {
		data := <-ss.IsUserOnline

		ss.ConnectionsByID.mutex.RLock()

		_, ok := ss.ConnectionsByID.data[data.Uid]

		ss.ConnectionsByID.mutex.RUnlock()

		data.RecvChan <- ok
	}
}

func messageLoop(ss *SocketServer) {
	for {
		data := <-ss.MessageLoop

		data.Conn.WriteMessage(1, data.Data)
	}
}

func sendUserData(ss *SocketServer) {
	for {
		data := <-ss.SendDataToUser

		ss.ConnectionsByID.mutex.Lock()

		if c, ok := ss.ConnectionsByID.data[data.Uid]; ok {
			ss.ConnectionsByID.mutex.Unlock()

			WriteMessage(data.MessageType, data.Data, c, ss)
		} else {
			ss.ConnectionsByID.mutex.Unlock()
		}
	}
}

func sendUsersData(ss *SocketServer) {
	for {
		data := <-ss.SendDataToUsers

		for _, uid := range data.Uids {
			ss.SendDataToUser <- UserMessageData{
				Data:        data.Data,
				MessageType: data.MessageType,
				Uid:         uid,
			}
		}
	}
}

func joinSubsByWs(ss *SocketServer) {
	for {
		data := <-ss.JoinSubscriptionByWs

		ss.ConnectionsByWs.mutex.RLock()

		if uid, ok := ss.ConnectionsByWs.data[data.Conn]; ok {
			ss.Subscriptions.mutex.Lock()

			if _, ok := ss.Subscriptions.data[data.SubName]; ok {
				ss.Subscriptions.data[data.SubName][uid] = struct{}{}
			} else {
				uids := make(map[string]struct{})
				uids[uid] = struct{}{}
				ss.Subscriptions.data[data.SubName] = uids
			}

			ss.Subscriptions.mutex.Unlock()
		}

		ss.ConnectionsByWs.mutex.RUnlock()
	}
}

func leaveSubByWs(ss *SocketServer) {
	for {
		data := <-ss.LeaveSubscriptionByWs

		ss.ConnectionsByWs.mutex.RLock()

		if uid, ok := ss.ConnectionsByWs.data[data.Conn]; ok {
			ss.Subscriptions.mutex.Lock()

			if _, ok := ss.Subscriptions.data[data.SubName]; ok {
				delete(ss.Subscriptions.data[data.SubName], uid)
				if len(ss.Subscriptions.data[data.SubName]) == 0 {
					delete(ss.Subscriptions.data, data.SubName)
				}
			}

			ss.Subscriptions.mutex.Unlock()
		}

		ss.ConnectionsByWs.mutex.RUnlock()
	}
}

func sendSubData(ss *SocketServer) {
	for {
		data := <-ss.SendDataToSub

		ss.Subscriptions.mutex.RLock()

		if uids, ok := ss.Subscriptions.data[data.SubName]; ok {
			ss.ConnectionsByID.mutex.RLock()

			for uid := range uids {
				WriteMessage(data.MessageType, data.Data, ss.ConnectionsByID.data[uid], ss)
			}

			ss.ConnectionsByID.mutex.RUnlock()
		}

		ss.Subscriptions.mutex.RUnlock()
	}
}

func sendSubsData(ss *SocketServer) {
	for {
		data := <-ss.SendDataToSubs

		ss.Subscriptions.mutex.RLock()

		for _, subName := range data.SubNames {
			if uids, ok := ss.Subscriptions.data[subName]; ok {
				ss.ConnectionsByID.mutex.RLock()

				for uid := range uids {
					WriteMessage(data.MessageType, data.Data, ss.ConnectionsByID.data[uid], ss)
				}

				ss.ConnectionsByID.mutex.RUnlock()
			}
		}

		ss.Subscriptions.mutex.RUnlock()
	}
}

func getConnSubscriptions(ss *SocketServer) {
	for {
		data := <-ss.GetConnectionSubscriptions

		ss.ConnectionsByWs.mutex.RLock()

		if uid, ok := ss.ConnectionsByWs.data[data.Conn]; ok {
			subs := make(map[string]struct{})

			ss.Subscriptions.mutex.RLock()

			for subName, uids := range ss.Subscriptions.data {
				for k := range uids {
					if k == uid {
						subs[subName] = struct{}{}
					}
				}
			}

			ss.Subscriptions.mutex.RUnlock()

			data.RecvChan <- subs
		}

		ss.ConnectionsByWs.mutex.RUnlock()
	}
}

func getSubscriptionUids(ss *SocketServer) {
	for {
		data := <-ss.GetSubscriptionUids

		ss.Subscriptions.mutex.RLock()

		out := make(map[string]struct{})

		if uids, ok := ss.Subscriptions.data[data.SubName]; ok {
			out = uids
		}

		ss.Subscriptions.mutex.RUnlock()

		data.RecvChan <- out
	}
}
