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

I used a load of maps to avoid ranging through stuff.

"ConnectionsByWs" and "ConnectionsByID" are both maps
that contain pointers to all connections, so that
connections can be accessed by using the id of the user,
or the user id can be easily accessed by the connection,
just to avoid ranging through maps because I imagine it's
not as fast as just accessing the variable more directly
using a map.

I don't know if it performs better or worse than my
last projects.

Maybe using a sync.Map here would be better, but I would
have to rewrite everything and I've already 99% finished
the project.
*/

type SocketServer struct {
	ConnectionsByWs ConnectionsByWs
	ConnectionsByID ConnectionsByID

	// used to avoid ranging through maps. Keeps names of every subscription
	// a connection is registered to.
	ConnectionSubscriptions    ConnectionSubscriptions
	GetConnectionSubscriptions chan GetConnectionSubscriptions

	IsUserOnline chan IsUserOnline

	MessageLoop chan Message

	AttachmentServerRemoveUploaderChan chan string

	RegisterConn   chan ConnnectionData
	UnregisterConn chan *websocket.Conn

	CloseConnChan chan string

	SendDataToUser  chan UserMessageData
	SendDataToConn  chan ConnMessageData
	SendDataToUsers chan UsersMessageData
	SendDataToConns chan ConnsMessageData

	JoinSubscriptionByWs  chan RegisterUnregisterSubsConnWs
	JoinSubscriptionByID  chan RegisterUnregisterSubsConnID
	LeaveSubscriptionByWs chan RegisterUnregisterSubsConnWs
	LeaveSubscriptionByID chan RegisterUnregisterSubsConnID

	SendDataToSub  chan SubscriptionMessageData
	SendDataToSubs chan SubscriptionsMessageData
	// Send data to subscription, exclude connection(s)
	SendDataToSubExcludeByWss chan SubscriptionMessageDataExcludeByWss
	// Send data to subscription, exclude connection(s) by matching user ids
	SendDataToSubExcludeByIDs chan SubscriptionMessageDataExcludeByIDs
	// Send data to multiple subscriptions, exclude connection(s)
	SendDataToSubsExcludeByWss chan SubscriptionsMessageDataExcludeByWss
	// Send data to multiple subscriptions, exclude connection(s) by matching user ids
	SendDataToSubsExcludeByIDs chan SubscriptionsMessageDataExcludeByIDs

	Subscriptions       Subscriptions
	GetSubscriptionUids chan GetSubscriptionUids
}

/* ------ INTERNAL MUTEX PROTECTED MAPS ------ */

type ConnectionsByWs struct {
	data  map[*websocket.Conn]string
	mutex sync.Mutex
}

type ConnectionsByID struct {
	data  map[string]*websocket.Conn
	mutex sync.Mutex
}

type ConnectionSubscriptions struct {
	data  map[*websocket.Conn]map[string]struct{}
	mutex sync.Mutex
}

type Subscriptions struct {
	data  map[string]map[*websocket.Conn]struct{}
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

type ConnsMessageData struct {
	Data        interface{}
	Conns       []*websocket.Conn
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

type SubscriptionsMessageData struct {
	SubNames    []string
	MessageType string
	Data        interface{}
}

type SubscriptionMessageDataExcludeByWss struct {
	SubName      string
	MessageType  string
	Data         interface{}
	ExcludeConns map[*websocket.Conn]struct{}
}

type SubscriptionMessageDataExcludeByIDs struct {
	SubName     string
	MessageType string
	Data        interface{}
	ExcludeUids map[string]struct{}
}

type SubscriptionsMessageDataExcludeByWss struct {
	SubNames     []string
	MessageType  string
	Data         interface{}
	ExcludeConns map[*websocket.Conn]struct{}
}

type SubscriptionsMessageDataExcludeByIDs struct {
	SubNames    []string
	MessageType string
	Data        interface{}
	ExcludeUids map[string]struct{}
}

func Init(csdc chan string, cRTCsdc chan string) *SocketServer {
	ss := &SocketServer{
		ConnectionsByWs: ConnectionsByWs{
			data: map[*websocket.Conn]string{},
		},
		ConnectionsByID: ConnectionsByID{
			data: make(map[string]*websocket.Conn),
		},

		IsUserOnline: make(chan IsUserOnline),

		ConnectionSubscriptions: ConnectionSubscriptions{
			data: make(map[*websocket.Conn]map[string]struct{}),
		},
		GetConnectionSubscriptions: make(chan GetConnectionSubscriptions),

		MessageLoop: make(chan Message),

		AttachmentServerRemoveUploaderChan: make(chan string),

		RegisterConn:   make(chan ConnnectionData),
		UnregisterConn: make(chan *websocket.Conn),

		CloseConnChan: make(chan string),

		SendDataToUser:  make(chan UserMessageData),
		SendDataToConn:  make(chan ConnMessageData),
		SendDataToUsers: make(chan UsersMessageData),
		SendDataToConns: make(chan ConnsMessageData),

		SendDataToSubExcludeByWss:  make(chan SubscriptionMessageDataExcludeByWss),
		SendDataToSubExcludeByIDs:  make(chan SubscriptionMessageDataExcludeByIDs),
		SendDataToSubsExcludeByWss: make(chan SubscriptionsMessageDataExcludeByWss),
		SendDataToSubsExcludeByIDs: make(chan SubscriptionsMessageDataExcludeByIDs),

		JoinSubscriptionByWs:  make(chan RegisterUnregisterSubsConnWs),
		JoinSubscriptionByID:  make(chan RegisterUnregisterSubsConnID),
		LeaveSubscriptionByWs: make(chan RegisterUnregisterSubsConnWs),
		LeaveSubscriptionByID: make(chan RegisterUnregisterSubsConnID),

		SendDataToSub:  make(chan SubscriptionMessageData),
		SendDataToSubs: make(chan SubscriptionsMessageData),

		Subscriptions: Subscriptions{
			data: make(map[string]map[*websocket.Conn]struct{}),
		},
		GetSubscriptionUids: make(chan GetSubscriptionUids),
	}
	go runServer(ss, csdc, cRTCsdc)
	log.Println("Socket server initialized")
	return ss
}

func runServer(ss *SocketServer, csdc chan string, cRTCsdc chan string) {
	go connection(ss)
	go disconnect(ss, csdc, cRTCsdc)
	go checkUserOnline(ss)
	go closeConn(ss)
	go messageLoop(ss)
	go sendUserData(ss)
	go sendConnData(ss)
	go sendUsersData(ss)
	go sendConnsData(ss)
	go joinSubsByWs(ss)
	go joinSubsByID(ss)
	go leaveSubByWs(ss)
	go leaveSubByID(ss)
	go sendSubData(ss)
	go sendSubsData(ss)
	go sendDataToSubExcludeWss(ss)
	go sendDataToSubExcludeIDs(ss)
	go sendDataToSubsExcludeWss(ss)
	go sendDataToSubsExcludeIDs(ss)
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
		ss.ConnectionsByWs.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()
		ss.ConnectionsByWs.data[data.Conn] = data.Uid
		ss.ConnectionsByID.data[data.Uid] = data.Conn
		ss.ConnectionsByID.mutex.Unlock()
		ss.ConnectionsByWs.mutex.Unlock()

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

		ss.ConnectionsByWs.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()
		ss.Subscriptions.mutex.Lock()
		ss.ConnectionSubscriptions.mutex.Lock()
		uid, ok := ss.ConnectionsByWs.data[conn]
		if ok {
			csdc <- uid
			cRTCsdc <- uid
			ss.AttachmentServerRemoveUploaderChan <- uid

			delete(ss.ConnectionsByID.data, uid)

			if subs, ok := ss.ConnectionSubscriptions.data[conn]; ok {
				for sub := range subs {
					if _, ok := ss.Subscriptions.data[sub]; ok {
						delete(ss.Subscriptions.data[sub], conn)
					}
				}
			}
			delete(ss.ConnectionSubscriptions.data, conn)
		}
		delete(ss.ConnectionsByWs.data, conn)
		ss.ConnectionsByWs.mutex.Unlock()
		ss.ConnectionsByID.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
		ss.ConnectionSubscriptions.mutex.Unlock()

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
		data.RecvChan <- ok
		ss.ConnectionsByID.mutex.Unlock()
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
		// mutex lock all maps that contain connections
		ss.ConnectionsByID.mutex.Lock()
		ss.ConnectionsByWs.mutex.Lock()
		ss.Subscriptions.mutex.Lock()
		if conn, ok := ss.ConnectionsByID.data[data.Uid]; ok {
			WriteMessage(data.MessageType, data.Data, conn, ss)
		}
		ss.ConnectionsByID.mutex.Unlock()
		ss.ConnectionsByWs.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
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
		// mutex lock all maps that contain connections
		ss.ConnectionsByID.mutex.Lock()
		ss.ConnectionsByWs.mutex.Lock()
		ss.Subscriptions.mutex.Lock()
		for _, v := range data.Uids {
			if conn, ok := ss.ConnectionsByID.data[v]; ok {
				WriteMessage(data.MessageType, data.Data, conn, ss)
			}
		}
		ss.ConnectionsByID.mutex.Unlock()
		ss.ConnectionsByWs.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func sendConnData(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws send conn data loop:", r)
				if failCount < 10 {
					go sendConnData(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.SendDataToConn
		// mutex lock all maps that contain connections
		ss.ConnectionsByID.mutex.Lock()
		ss.ConnectionsByWs.mutex.Lock()
		ss.Subscriptions.mutex.Lock()
		if _, ok := ss.ConnectionsByWs.data[data.Conn]; ok {
			WriteMessage(data.MessageType, data.Data, data.Conn, ss)
		}
		ss.ConnectionsByID.mutex.Unlock()
		ss.ConnectionsByWs.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func sendConnsData(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws send connections data loop:", r)
				if failCount < 10 {
					go sendConnsData(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.SendDataToConns
		// mutex lock all maps that contain connections
		ss.ConnectionsByID.mutex.Lock()
		ss.ConnectionsByWs.mutex.Lock()
		ss.Subscriptions.mutex.Lock()
		for _, conn := range data.Conns {
			if _, ok := ss.ConnectionsByWs.data[conn]; ok {
				WriteMessage(data.MessageType, data.Data, conn, ss)
			}
		}
		ss.ConnectionsByID.mutex.Unlock()
		ss.ConnectionsByWs.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
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
		ss.ConnectionSubscriptions.mutex.Lock()
		if _, ok := ss.Subscriptions.data[data.SubName]; ok {
			ss.Subscriptions.data[data.SubName][data.Conn] = struct{}{}
		} else {
			conns := make(map[*websocket.Conn]struct{})
			conns[data.Conn] = struct{}{}
			ss.Subscriptions.data[data.SubName] = conns
		}

		if _, ok := ss.ConnectionSubscriptions.data[data.Conn]; ok {
			ss.ConnectionSubscriptions.data[data.Conn][data.SubName] = struct{}{}
		} else {
			subs := make(map[string]struct{})
			subs[data.SubName] = struct{}{}
			ss.ConnectionSubscriptions.data[data.Conn] = subs
		}
		ss.Subscriptions.mutex.Unlock()
		ss.ConnectionSubscriptions.mutex.Unlock()
	}
}

func joinSubsByID(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws register subscription by uid loop:", r)
				if failCount < 10 {
					go joinSubsByID(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.JoinSubscriptionByID
		ss.Subscriptions.mutex.Lock()
		ss.ConnectionSubscriptions.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()
		if conn, ok := ss.ConnectionsByID.data[data.Uid]; ok {
			if _, ok := ss.Subscriptions.data[data.SubName]; ok {
				ss.Subscriptions.data[data.SubName][conn] = struct{}{}
			} else {
				conns := make(map[*websocket.Conn]struct{})
				conns[conn] = struct{}{}
				ss.Subscriptions.data[data.SubName] = conns
			}

			if _, ok := ss.ConnectionSubscriptions.data[conn]; ok {
				ss.ConnectionSubscriptions.data[conn][data.SubName] = struct{}{}
			} else {
				subs := make(map[string]struct{})
				subs[data.SubName] = struct{}{}
				ss.ConnectionSubscriptions.data[conn] = subs
			}
		} else {
			log.Println("Could not register user ID to subscription - connection information not found in memory")
		}
		ss.Subscriptions.mutex.Unlock()
		ss.ConnectionSubscriptions.mutex.Unlock()
		ss.ConnectionsByID.mutex.Unlock()
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
		ss.ConnectionSubscriptions.mutex.Lock()
		if _, ok := ss.Subscriptions.data[data.SubName]; ok {
			delete(ss.Subscriptions.data[data.SubName], data.Conn)
			if len(ss.Subscriptions.data[data.SubName]) == 0 {
				delete(ss.Subscriptions.data, data.SubName)
			}
		}

		delete(ss.ConnectionSubscriptions.data[data.Conn], data.SubName)
		ss.Subscriptions.mutex.Unlock()
		ss.ConnectionSubscriptions.mutex.Unlock()
	}
}

func leaveSubByID(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws leave subscription by uid loop:", r)
				if failCount < 10 {
					go leaveSubByID(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.LeaveSubscriptionByID
		ss.Subscriptions.mutex.Lock()
		ss.ConnectionsByID.mutex.Lock()
		ss.ConnectionSubscriptions.mutex.Lock()
		ss.Subscriptions.mutex.Lock()
		conn, connRegistered := ss.ConnectionsByID.data[data.Uid]
		if !connRegistered {
			ss.Subscriptions.mutex.Unlock()
			ss.ConnectionsByID.mutex.Unlock()
			ss.ConnectionSubscriptions.mutex.Unlock()
			ss.Subscriptions.mutex.Unlock()
			continue
		}
		if _, ok := ss.Subscriptions.data[data.SubName]; ok {
			delete(ss.Subscriptions.data[data.SubName], conn)
			if len(ss.Subscriptions.data[data.SubName]) == 0 {
				delete(ss.Subscriptions.data, data.SubName)
			}
		}

		delete(ss.ConnectionSubscriptions.data[conn], data.SubName)
		ss.Subscriptions.mutex.Unlock()
		ss.ConnectionsByID.mutex.Unlock()
		ss.ConnectionSubscriptions.mutex.Unlock()
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
		// mutex lock all maps that contain connections
		ss.ConnectionsByID.mutex.Lock()
		ss.ConnectionsByWs.mutex.Lock()
		ss.Subscriptions.mutex.Lock()
		if conns, ok := ss.Subscriptions.data[data.SubName]; ok {
			for c := range conns {
				WriteMessage(data.MessageType, data.Data, c, ss)
			}
		}
		ss.ConnectionsByID.mutex.Unlock()
		ss.ConnectionsByWs.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func sendSubsData(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws send data to subscriptions loop:", r)
				if failCount < 10 {
					go sendSubsData(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.SendDataToSubs
		// mutex lock all maps that contain connections
		ss.ConnectionsByID.mutex.Lock()
		ss.ConnectionsByWs.mutex.Lock()
		ss.Subscriptions.mutex.Lock()
		for _, v := range data.SubNames {
			if _, ok := ss.Subscriptions.data[v]; ok {
				if conns, ok := ss.Subscriptions.data[v]; ok {
					for c := range conns {
						WriteMessage(data.MessageType, data.Data, c, ss)
					}
				}
			}
		}
		ss.ConnectionsByID.mutex.Unlock()
		ss.ConnectionsByWs.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func sendDataToSubExcludeWss(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws send data to subscription excluding connections loop:", r)
				if failCount < 10 {
					go sendDataToSubExcludeWss(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.SendDataToSubExcludeByWss
		// mutex lock all maps that contain connections
		ss.ConnectionsByID.mutex.Lock()
		ss.ConnectionsByWs.mutex.Lock()
		ss.Subscriptions.mutex.Lock()
		if conns, ok := ss.Subscriptions.data[data.SubName]; ok {
			for c := range conns {
				if _, ok := data.ExcludeConns[c]; !ok {
					WriteMessage(data.MessageType, data.Data, c, ss)
				}
			}
		}
		ss.ConnectionsByID.mutex.Unlock()
		ss.ConnectionsByWs.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func sendDataToSubExcludeIDs(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws send data to subscription excluding user ids loop:", r)
				if failCount < 10 {
					go sendDataToSubExcludeIDs(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.SendDataToSubExcludeByIDs
		// mutex lock all maps that contain connections
		ss.ConnectionsByID.mutex.Lock()
		ss.ConnectionsByWs.mutex.Lock()
		ss.Subscriptions.mutex.Lock()
		if conns, ok := ss.Subscriptions.data[data.SubName]; ok {
			for c := range conns {
				if id, ok := ss.ConnectionsByWs.data[c]; ok {
					if _, ok := data.ExcludeUids[id]; !ok {
						WriteMessage(data.MessageType, data.Data, c, ss)
					}
				}
			}
		}
		ss.ConnectionsByID.mutex.Unlock()
		ss.ConnectionsByWs.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func sendDataToSubsExcludeIDs(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws send data to subscriptions excluding user ids loop:", r)
				if failCount < 10 {
					go sendDataToSubsExcludeIDs(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.SendDataToSubsExcludeByIDs
		// mutex lock all maps that contain connections
		ss.ConnectionsByID.mutex.Lock()
		ss.ConnectionsByWs.mutex.Lock()
		ss.Subscriptions.mutex.Lock()
		for _, subName := range data.SubNames {
			if conns, ok := ss.Subscriptions.data[subName]; ok {
				for c := range conns {
					if id, ok := ss.ConnectionsByWs.data[c]; ok {
						if _, ok := data.ExcludeUids[id]; !ok {
							WriteMessage(data.MessageType, data.Data, c, ss)
						}
					}
				}
			}
		}
		ss.ConnectionsByID.mutex.Unlock()
		ss.ConnectionsByWs.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func sendDataToSubsExcludeWss(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws send data to subscriptions excluding connections loop:", r)
				if failCount < 10 {
					go sendDataToSubsExcludeWss(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-ss.SendDataToSubsExcludeByWss
		// mutex lock all maps that contain connections
		ss.ConnectionsByID.mutex.Lock()
		ss.ConnectionsByWs.mutex.Lock()
		ss.Subscriptions.mutex.Lock()
		for _, subName := range data.SubNames {
			if conns, ok := ss.Subscriptions.data[subName]; ok {
				for c := range conns {
					if _, ok := data.ExcludeConns[c]; !ok {
						WriteMessage(data.MessageType, data.Data, c, ss)
					}
				}
			}
		}
		ss.ConnectionsByID.mutex.Unlock()
		ss.ConnectionsByWs.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
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
		ss.ConnectionSubscriptions.mutex.Lock()
		if subs, ok := ss.ConnectionSubscriptions.data[data.Conn]; ok {
			data.RecvChan <- subs
		} else {
			empty := make(map[string]struct{})
			data.RecvChan <- empty
		}
		ss.ConnectionSubscriptions.mutex.Unlock()
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
		ss.ConnectionsByWs.mutex.Lock()
		uids := make(map[string]struct{})
		conns, ok := ss.Subscriptions.data[data.SubName]
		if ok {
			for c := range conns {
				if uid, ok := ss.ConnectionsByWs.data[c]; ok {
					uids[uid] = struct{}{}
				}
			}
			data.RecvChan <- uids
		} else {
			data.RecvChan <- uids
		}
		ss.ConnectionsByWs.mutex.Unlock()
		ss.Subscriptions.mutex.Unlock()
	}
}
