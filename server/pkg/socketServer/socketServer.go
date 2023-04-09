package socketServer

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
	socketmessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
)

/*
This works differently to my last 2 projects.

It can only send JSON messages, in this form:
{ event_type, data }

I will end up using sync.Map next time instead
of mutex locks
*/

type SocketServer struct {
	ConnectionsByID            ConnectionsByID
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

	Subscriptions       Subscriptions
	GetSubscriptionUids chan GetSubscriptionUids
}

/* ------ INTERNAL MUTEX PROTECTED MAPS ------ */

type ConnectionsByID struct {
	data  map[string]*websocket.Conn
	mutex sync.Mutex
}

type Subscriptions struct {
	// outer map is subscription name, inner map is uids
	data  map[string]map[string]struct{}
	mutex sync.Mutex
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

func Init(csdc chan string, cRTCsdc chan string) *SocketServer {
	ss := &SocketServer{
		ConnectionsByID: ConnectionsByID{
			data: make(map[string]*websocket.Conn),
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

		Subscriptions: Subscriptions{
			data: make(map[string]map[string]struct{}),
		},
		GetSubscriptionUids: make(chan GetSubscriptionUids),
	}
	go runServer(ss, csdc, cRTCsdc)
	return ss
}

func runServer(ss *SocketServer, csdc chan string, cRTCsdc chan string) {
	go connection(ss)
	go disconnect(ss, csdc, cRTCsdc)
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
}

func closeConn(ss *SocketServer) {
	for {
		uid := <-ss.CloseConnChan

		ss.ConnectionsByID.mutex.Lock()
		if conn, ok := ss.ConnectionsByID.data[uid]; ok {
			ss.ConnectionsByID.mutex.Unlock()
			ss.UnregisterConn <- conn
		} else {
			ss.ConnectionsByID.mutex.Unlock()
		}
	}
}

func WriteMessage(t string, m interface{}, c *websocket.Conn, ss *SocketServer) {
	withType := make(map[string]interface{})
	withType["event_type"] = t
	withType["data"] = m
	if b, err := json.Marshal(withType); err == nil {
		ss.MessageLoop <- Message{
			Conn: c,
			Data: b,
		}
	} else {
		log.Println("Error marshalling message:", err)
	}
}

func connection(ss *SocketServer) {
	for {
		data := <-ss.RegisterConn

		log.Println("Connection registration")

		ss.ConnectionsByID.mutex.Lock()

		ss.ConnectionsByID.data[data.Uid] = data.Conn

		ss.ConnectionsByID.mutex.Unlock()

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

func disconnect(ss *SocketServer, csdc chan string, cRTCsdc chan string) {
	for {
		conn := <-ss.UnregisterConn

		log.Println("Disconnect registration")

		ss.ConnectionsByID.mutex.Lock()
		ss.Subscriptions.mutex.Lock()

		var uid string
		for k, c := range ss.ConnectionsByID.data {
			if c == conn {
				uid = k
				break
			}
		}

		csdc <- uid
		cRTCsdc <- uid
		ss.AttachmentServerRemoveUploaderChan <- uid
		delete(ss.ConnectionsByID.data, uid)

		ss.Subscriptions.mutex.Unlock()
		ss.ConnectionsByID.mutex.Unlock()

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

		ss.ConnectionsByID.mutex.Lock()

		_, ok := ss.ConnectionsByID.data[data.Uid]

		ss.ConnectionsByID.mutex.Unlock()

		data.RecvChan <- ok
	}
}

func messageLoop(ss *SocketServer) {
	for {
		msg := <-ss.MessageLoop
		msg.Conn.WriteMessage(1, msg.Data)
	}
}

func sendUserData(ss *SocketServer) {
	for {
		data := <-ss.SendDataToUser

		ss.Subscriptions.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()

		if c, ok := ss.ConnectionsByID.data[data.Uid]; ok {
			WriteMessage(data.MessageType, data.Data, c, ss)
		}

		ss.ConnectionsByID.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func sendUsersData(ss *SocketServer) {
	for {
		data := <-ss.SendDataToUsers

		log.Println("Send")

		ss.Subscriptions.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()

		conns := []*websocket.Conn{}

		for k, c := range ss.ConnectionsByID.data {
			for _, v := range data.Uids {
				if v == k {
					log.Println("Append")
					conns = append(conns, c)
				}
			}
		}

		for _, c := range conns {
			log.Println("Supposed to have written message")
			WriteMessage(data.MessageType, data.Data, c, ss)
		}

		ss.ConnectionsByID.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func joinSubsByWs(ss *SocketServer) {
	for {
		data := <-ss.JoinSubscriptionByWs

		ss.Subscriptions.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()

		var uid string

		for k, c := range ss.ConnectionsByID.data {
			if c == data.Conn {
				uid = k
				break
			}
		}

		if _, ok := ss.Subscriptions.data[data.SubName]; ok {
			ss.Subscriptions.data[data.SubName][uid] = struct{}{}
		} else {
			uids := make(map[string]struct{})
			uids[uid] = struct{}{}
			ss.Subscriptions.data[data.SubName] = uids
		}

		ss.ConnectionsByID.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func leaveSubByWs(ss *SocketServer) {
	for {
		data := <-ss.LeaveSubscriptionByWs

		ss.Subscriptions.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()

		var uid string

		for k, c := range ss.ConnectionsByID.data {
			if c == data.Conn {
				uid = k
				break
			}
		}

		if _, ok := ss.Subscriptions.data[data.SubName]; ok {
			delete(ss.Subscriptions.data[data.SubName], uid)
			if len(ss.Subscriptions.data[data.SubName]) == 0 {
				delete(ss.Subscriptions.data, data.SubName)
			}
		}

		ss.ConnectionsByID.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func sendSubData(ss *SocketServer) {
	for {
		data := <-ss.SendDataToSub

		ss.Subscriptions.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()

		if uids, ok := ss.Subscriptions.data[data.SubName]; ok {
			for uid := range uids {
				for k, c := range ss.ConnectionsByID.data {
					if k == uid {
						WriteMessage(data.MessageType, data.Data, c, ss)
					}
				}
			}
		}

		ss.ConnectionsByID.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func sendSubsData(ss *SocketServer) {
	for {
		data := <-ss.SendDataToSubs

		ss.Subscriptions.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()

		for _, subName := range data.SubNames {
			if uids, ok := ss.Subscriptions.data[subName]; ok {
				for uid := range uids {
					for k, c := range ss.ConnectionsByID.data {
						if k == uid {
							WriteMessage(data.MessageType, data.Data, c, ss)
						}
					}
				}
			}
		}

		ss.ConnectionsByID.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func getConnSubscriptions(ss *SocketServer) {
	for {
		data := <-ss.GetConnectionSubscriptions

		ss.Subscriptions.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()

		var uid string

		for k, c := range ss.ConnectionsByID.data {
			if c == data.Conn {
				uid = k
			}
		}

		subs := make(map[string]struct{})

		for subName, uids := range ss.Subscriptions.data {
			for k := range uids {
				if k == uid {
					subs[subName] = struct{}{}
				}
			}
		}

		ss.ConnectionsByID.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()

		data.RecvChan <- subs
	}
}

func getSubscriptionUids(ss *SocketServer) {
	for {
		data := <-ss.GetSubscriptionUids

		ss.Subscriptions.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()

		out := make(map[string]struct{})

		if uids, ok := ss.Subscriptions.data[data.SubName]; ok {
			out = uids
		}

		ss.ConnectionsByID.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()

		data.RecvChan <- out
	}
}
