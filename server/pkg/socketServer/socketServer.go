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

I will end up using sync.Map next time instead
of mutex locks.

I ended up using only 1 mutex lock for all
data since I was getting loads of random
deadlocks and couldn't be asked with it
anymore, even that still didn't fix it, I
cannot run with -race flag because fasthttp
doesn't work properly with CGO... not fun

I also removed panic recovery because it's
kind of pointless. There shouldn't be any
panics anyway
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
}

/* ------ INTERNAL MUTEX PROTECTED MAPS ------ */

type Server struct {
	data  ServerData
	mutex sync.RWMutex
}

type ServerData struct {
	ConnectionsByID map[string]*websocket.Conn
	// outer map is subscription name, inner map is uids
	Subscriptions map[string]map[string]struct{}
}

/* ------ RECV CHAN STRUCTS ------ */

type GetConnectionSubscriptions struct {
	RecvChan chan<- map[string]struct{}
	Conn     *websocket.Conn
}

type GetSubscriptionUids struct {
	RecvChan chan<- map[string]struct{}
	SubName  string
}

type IsUserOnline struct {
	RecvChan chan<- bool
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
		Server: Server{
			data: ServerData{
				ConnectionsByID: make(map[string]*websocket.Conn),
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

		ss.Server.mutex.Lock()

		if conn, ok := ss.Server.data.ConnectionsByID[uid]; ok {
			ss.UnregisterConn <- conn

			ss.Server.mutex.Unlock()
		} else {
			ss.Server.mutex.Unlock()
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
	}
}

func connection(ss *SocketServer) {
	for {
		data := <-ss.RegisterConn

		ss.Server.mutex.Lock()

		ss.Server.data.ConnectionsByID[data.Uid] = data.Conn

		ss.Server.mutex.Unlock()

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

		ss.Server.mutex.Lock()

		var uid string
		for k, c := range ss.Server.data.ConnectionsByID {
			if c == conn {
				uid = k
				break
			}
		}

		delete(ss.Server.data.ConnectionsByID, uid)

		ss.Server.mutex.Unlock()

		csdc <- uid
		cRTCsdc <- uid
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
	var wg sync.WaitGroup

	for {
		msg := <-ss.MessageLoop

		wg.Wait()
		wg.Add(1)

		msg.Conn.WriteMessage(1, msg.Data)

		wg.Done()
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

		conns := []*websocket.Conn{}

		for k, c := range ss.Server.data.ConnectionsByID {
			for _, v := range data.Uids {
				if v == k {
					conns = append(conns, c)
				}
			}
		}

		for _, c := range conns {
			WriteMessage(data.MessageType, data.Data, c, ss)
		}

		ss.Server.mutex.Unlock()
	}
}

func joinSubsByWs(ss *SocketServer) {
	for {
		data := <-ss.JoinSubscriptionByWs

		ss.Server.mutex.Lock()

		var uid string

		for k, c := range ss.Server.data.ConnectionsByID {
			if c == data.Conn {
				uid = k
				break
			}
		}

		if _, ok := ss.Server.data.Subscriptions[data.SubName]; ok {
			ss.Server.data.Subscriptions[data.SubName][uid] = struct{}{}
		} else {
			uids := make(map[string]struct{})
			uids[uid] = struct{}{}
			ss.Server.data.Subscriptions[data.SubName] = uids
		}

		ss.Server.mutex.Unlock()
	}
}

func leaveSubByWs(ss *SocketServer) {
	for {
		data := <-ss.LeaveSubscriptionByWs

		ss.Server.mutex.Lock()

		var uid string

		for k, c := range ss.Server.data.ConnectionsByID {
			if c == data.Conn {
				uid = k
				break
			}
		}

		if _, ok := ss.Server.data.Subscriptions[data.SubName]; ok {
			delete(ss.Server.data.Subscriptions[data.SubName], uid)
			if len(ss.Server.data.Subscriptions[data.SubName]) == 0 {
				delete(ss.Server.data.Subscriptions, data.SubName)
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
				for k, c := range ss.Server.data.ConnectionsByID {
					if k == uid {
						WriteMessage(data.MessageType, data.Data, c, ss)
					}
				}
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
					for k, c := range ss.Server.data.ConnectionsByID {
						if k == uid {
							WriteMessage(data.MessageType, data.Data, c, ss)
						}
					}
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

		var uid string

		for k, c := range ss.Server.data.ConnectionsByID {
			if c == data.Conn {
				uid = k
				break
			}
		}

		subs := make(map[string]struct{})

		for subName, uids := range ss.Server.data.Subscriptions {
			for k := range uids {
				if k == uid {
					subs[subName] = struct{}{}
				}
			}
		}

		data.RecvChan <- subs

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
