package socketServer

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gofiber/websocket/v2"
	socketmessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
)

/*
This works differently to my last 2 projects.

It can only send JSON messages, in this form:
{ event_type, data }

I ended up using only 1 mutex lock for all
data since I thought I was getting deadlocks
but it turned out to be something else
*/

type SocketServer struct {
	Server                     Server
	GetConnectionSubscriptions chan GetConnectionSubscriptions

	IsUserOnline chan IsUserOnline

	MessageLoop chan Message

	AttachmentServerRemoveUploaderChan chan string

	RegisterConn   chan ConnnectionData
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

type Server struct {
	data  ServerData
	mutex sync.RWMutex
}

type ServerData struct {
	ConnectionsByID map[string]*websocket.Conn
	ConnectionsByWs map[*websocket.Conn]string
	// outer map is subscription name, inner map is uids
	Subscriptions map[string]map[string]struct{}
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

type ConnnectionData struct {
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
		Server: Server{
			data: ServerData{
				ConnectionsByID: make(map[string]*websocket.Conn),
				ConnectionsByWs: make(map[*websocket.Conn]string),
				Subscriptions:   make(map[string]map[string]struct{}),
			},
		},
		GetConnectionSubscriptions: make(chan GetConnectionSubscriptions),

		IsUserOnline: make(chan IsUserOnline),

		MessageLoop: make(chan Message),

		AttachmentServerRemoveUploaderChan: make(chan string),

		RegisterConn:   make(chan ConnnectionData),
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

		ss.Server.mutex.RLock()

		if c, ok := ss.Server.data.ConnectionsByID[data.Uid]; ok {
			data.RecvChan <- c
		} else {
			data.RecvChan <- nil
		}

		ss.Server.mutex.RUnlock()
	}
}

func closeConn(ss *SocketServer) {
	for {
		uid := <-ss.CloseConnChan

		ss.Server.mutex.Lock()

		if conn, ok := ss.Server.data.ConnectionsByID[uid]; ok {
			ss.UnregisterConn <- conn
		}

		ss.Server.mutex.Unlock()
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

		ss.Server.mutex.Lock()

		ss.Server.data.ConnectionsByID[data.Uid] = data.Conn
		ss.Server.data.ConnectionsByWs[data.Conn] = data.Uid

		ss.Server.mutex.Unlock()

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

		ss.Server.mutex.Lock()

		uid := ss.Server.data.ConnectionsByWs[conn]

		delete(ss.Server.data.ConnectionsByID, uid)
		delete(ss.Server.data.ConnectionsByWs, conn)

		ss.Server.mutex.Unlock()

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

		ss.Server.mutex.RLock()

		_, ok := ss.Server.data.ConnectionsByID[data.Uid]

		data.RecvChan <- ok

		ss.Server.mutex.RUnlock()
	}
}

func messageLoop(ss *SocketServer) {
	for {
		msg := <-ss.MessageLoop

		// stupid way of avoiding datarace

		ss.Server.mutex.Lock()

		if key, ok := ss.Server.data.ConnectionsByWs[msg.Conn]; ok {
			if conn, ok := ss.Server.data.ConnectionsByID[key]; ok {
				conn.WriteMessage(1, msg.Data)
			}
		}

		ss.Server.mutex.Unlock()
	}
}

func sendUserData(ss *SocketServer) {
	for {
		data := <-ss.SendDataToUser

		ss.Server.mutex.Lock()

		if c, ok := ss.Server.data.ConnectionsByID[data.Uid]; ok {
			WriteMessage(data.MessageType, data.Data, c, ss)
		}

		ss.Server.mutex.Unlock()
	}
}

func sendUsersData(ss *SocketServer) {
	for {
		data := <-ss.SendDataToUsers

		ss.Server.mutex.Lock()

		for _, v := range data.Uids {
			WriteMessage(data.MessageType, data.Data, ss.Server.data.ConnectionsByID[v], ss)
		}

		ss.Server.mutex.Unlock()
	}
}

func joinSubsByWs(ss *SocketServer) {
	for {
		data := <-ss.JoinSubscriptionByWs

		ss.Server.mutex.Lock()

		if uid, ok := ss.Server.data.ConnectionsByWs[data.Conn]; ok {
			if _, ok := ss.Server.data.Subscriptions[data.SubName]; ok {
				ss.Server.data.Subscriptions[data.SubName][uid] = struct{}{}
			} else {
				uids := make(map[string]struct{})
				uids[uid] = struct{}{}
				ss.Server.data.Subscriptions[data.SubName] = uids
			}
		}

		ss.Server.mutex.Unlock()
	}
}

func leaveSubByWs(ss *SocketServer) {
	for {
		data := <-ss.LeaveSubscriptionByWs

		ss.Server.mutex.Lock()

		if uid, ok := ss.Server.data.ConnectionsByWs[data.Conn]; ok {
			if _, ok := ss.Server.data.Subscriptions[data.SubName]; ok {
				delete(ss.Server.data.Subscriptions[data.SubName], uid)
				if len(ss.Server.data.Subscriptions[data.SubName]) == 0 {
					delete(ss.Server.data.Subscriptions, data.SubName)
				}
			}
		}

		ss.Server.mutex.Unlock()
	}
}

func sendSubData(ss *SocketServer) {
	for {
		data := <-ss.SendDataToSub

		ss.Server.mutex.Lock()

		if uids, ok := ss.Server.data.Subscriptions[data.SubName]; ok {
			for uid := range uids {
				WriteMessage(data.MessageType, data.Data, ss.Server.data.ConnectionsByID[uid], ss)
			}
		}

		ss.Server.mutex.Unlock()
	}
}

func sendSubsData(ss *SocketServer) {
	for {
		data := <-ss.SendDataToSubs

		ss.Server.mutex.Lock()

		for _, subName := range data.SubNames {
			if uids, ok := ss.Server.data.Subscriptions[subName]; ok {
				for uid := range uids {
					WriteMessage(data.MessageType, data.Data, ss.Server.data.ConnectionsByID[uid], ss)
				}
			}
		}

		ss.Server.mutex.Unlock()
	}
}

func getConnSubscriptions(ss *SocketServer) {
	for {
		data := <-ss.GetConnectionSubscriptions

		ss.Server.mutex.RLock()

		if uid, ok := ss.Server.data.ConnectionsByWs[data.Conn]; ok {
			subs := make(map[string]struct{})

			for subName, uids := range ss.Server.data.Subscriptions {
				for k := range uids {
					if k == uid {
						subs[subName] = struct{}{}
					}
				}
			}

			data.RecvChan <- subs
		}

		ss.Server.mutex.RUnlock()
	}
}

func getSubscriptionUids(ss *SocketServer) {
	for {
		data := <-ss.GetSubscriptionUids

		ss.Server.mutex.RLock()

		out := make(map[string]struct{})

		if uids, ok := ss.Server.data.Subscriptions[data.SubName]; ok {
			out = uids
		}

		data.RecvChan <- out

		ss.Server.mutex.RUnlock()
	}
}
