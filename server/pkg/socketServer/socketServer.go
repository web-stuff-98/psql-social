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

I will end up using sync.Map next time, this
had a lot more methods but I had to remove them
since the deadlocks were becoming unmanagable.
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

	SendDataToSub chan SubscriptionMessageData

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

type RegisterUnregisterSubsConnID struct {
	Uid     string
	SubName string
}

type SubscriptionMessageData struct {
	SubName     string
	MessageType string
	Data        interface{}
}

func Init(csdc chan string, cRTCsdc chan string) *SocketServer {
	ss := &SocketServer{
		ConnectionsByID: ConnectionsByID{
			data: make(map[string]*websocket.Conn),
		},
		GetConnectionSubscriptions: make(chan GetConnectionSubscriptions),

		MessageLoop: make(chan Message),

		IsUserOnline: make(chan IsUserOnline),

		AttachmentServerRemoveUploaderChan: make(chan string),

		RegisterConn:   make(chan ConnnectionData),
		UnregisterConn: make(chan *websocket.Conn),

		CloseConnChan: make(chan string),

		SendDataToUser:  make(chan UserMessageData),
		SendDataToUsers: make(chan UsersMessageData),

		SendDataToSub: make(chan SubscriptionMessageData),

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
	go getConnSubscriptions(ss)
	go getSubscriptionUids(ss)
}

func closeConn(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws close connection loop:", r)
				if failCount < 10 {
					go connection(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

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
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws connection loop:", r)
				if failCount < 10 {
					go connection(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.RegisterConn
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
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws disconnect loop:", r)
				if failCount < 10 {
					go disconnect(ss, csdc, cRTCsdc)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		conn := <-ss.UnregisterConn

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
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws checkUserOnline loop:", r)
				if failCount < 10 {
					go checkUserOnline(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.IsUserOnline

		ss.ConnectionsByID.mutex.Lock()
		_, ok := ss.ConnectionsByID.data[data.Uid]
		ss.ConnectionsByID.mutex.Unlock()
		data.RecvChan <- ok
	}
}

func messageLoop(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws message loop:", r)
				if failCount < 10 {
					go sendUserData(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		msg := <-ss.MessageLoop
		msg.Conn.WriteMessage(1, msg.Data)
	}
}

func sendUserData(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws send user data loop:", r)
				if failCount < 10 {
					go sendUserData(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.SendDataToUser

		ss.ConnectionsByID.mutex.Lock()
		ss.Subscriptions.mutex.Lock()
		if conn, ok := ss.ConnectionsByID.data[data.Uid]; ok {
			WriteMessage(data.MessageType, data.Data, conn, ss)
		}
		ss.Subscriptions.mutex.Unlock()
		ss.ConnectionsByID.mutex.Unlock()
	}
}

func sendUsersData(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws send users data loop:", r)
				if failCount < 10 {
					go sendUsersData(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.SendDataToUsers

		ss.ConnectionsByID.mutex.Lock()
		ss.Subscriptions.mutex.Lock()

		conns := []*websocket.Conn{}

		for k, c := range ss.ConnectionsByID.data {
			for _, v := range data.Uids {
				if v == k {
					conns = append(conns, c)
				}
			}
		}

		for _, c := range conns {
			WriteMessage(data.MessageType, data.Data, c, ss)
		}

		ss.Subscriptions.mutex.Unlock()
		ss.ConnectionsByID.mutex.Unlock()
	}
}

func joinSubsByWs(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws register subscription by ws connection loop:", r)
				if failCount < 10 {
					go joinSubsByWs(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

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
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws leave subscription by ws conn loop:", r)
				if failCount < 10 {
					go leaveSubByWs(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

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
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws send data to subscription loop:", r)
				if failCount < 10 {
					go sendSubData(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.SendDataToSub

		ss.ConnectionsByID.mutex.Lock()
		ss.Subscriptions.mutex.Lock()

		if uids, ok := ss.Subscriptions.data[data.SubName]; ok {
			for uid := range uids {
				for k, c := range ss.ConnectionsByID.data {
					if k == uid {
						WriteMessage(data.MessageType, data.Data, c, ss)
					}
				}
			}
		}

		ss.Subscriptions.mutex.Unlock()
		ss.ConnectionsByID.mutex.Unlock()
	}
}

func getConnSubscriptions(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws get connection subscriptions loop:", r)
				if failCount < 10 {
					go getConnSubscriptions(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

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

		data.RecvChan <- subs

		ss.ConnectionsByID.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func getSubscriptionUids(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws get connection subscriptions loop:", r)
				if failCount < 10 {
					go getSubscriptionUids(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.GetSubscriptionUids

		ss.Subscriptions.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()

		if uids, ok := ss.Subscriptions.data[data.SubName]; ok {
			data.RecvChan <- uids
		} else {
			data.RecvChan <- make(map[string]struct{})
		}

		ss.ConnectionsByID.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}
