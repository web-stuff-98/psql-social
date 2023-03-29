package socketServer

import (
	"log"
	"sync"

	"github.com/fasthttp/websocket"
)

/*
This works differently to my last 2 projects, mainly because users can
establish a socket connection without being logged in. Also using
unidirectional channels for performance.

There are loads more channels because I am trying to avoid ranging
through maps as often this time. It means more code and memory usage
but should improve performance since ranging through a map is slower
than accessing by key.

I haven't tested performance but it's probably better than the previous
versions I used in my other projects.
*/

type SocketServer struct {
	ConnectionsByWs ConnectionsByWs
	ConnectionsByID ConnectionsByID

	// used to avoid ranging through maps. Keeps names of every subscription
	// a connection is registered to.
	ConnectionSubscriptions ConnectionSubscriptions

	RegisterConn   <-chan ConnnectionData
	UnregisterConn <-chan *websocket.Conn

	SendDataToUser  <-chan UserMessageData
	SendDataToConn  <-chan ConnMessageData
	SendDataToUsers <-chan UsersMessageData
	SendDataToConns <-chan ConnsMessageData

	JoinSubscriptionByWs  <-chan RegisterUnregisterSubsConnWs
	JoinSubscriptionByID  <-chan RegisterUnregisterSubsConnID
	LeaveSubscriptionByWs <-chan RegisterUnregisterSubsConnWs
	LeaveSubscriptionByID <-chan RegisterUnregisterSubsConnID

	SendDataToSub  <-chan SubscriptionMessageData
	SendDataToSubs <-chan SubscriptionsMessageData
	// Send data to subscription, exclude connection(s)
	SendDataToSubExcludeByWss <-chan SubscriptionMessageDataExcludeByWss
	// Send data to subscription, exclude connection(s) by matching user ids
	SendDataToSubExcludeByIDs <-chan SubscriptionMessageDataExcludeByIDs
	// Send data to multiple subscriptions, exclude connection(s)
	SendDataToSubsExcludeByWss <-chan SubscriptionsMessageDataExcludeByWss
	// Send data to multiple subscriptions, exclude connection(s) by matching user ids
	SendDataToSubsExcludeByIDs <-chan SubscriptionsMessageDataExcludeByIDs

	Subscriptions Subscriptions
}

/* ------ INTERNAL MUTEX PROTECTED MAPS ------ */

type ConnectionsByWs struct {
	data  map[*websocket.Conn]string
	mutex sync.RWMutex
}

type ConnectionsByID struct {
	data  map[string]*websocket.Conn
	mutex sync.RWMutex
}

type ConnectionSubscriptions struct {
	data  map[*websocket.Conn]map[string]struct{}
	mutex sync.Mutex
}

type Subscriptions struct {
	data  map[string]map[*websocket.Conn]struct{}
	mutex sync.RWMutex
}

/* ------ GENERAL STRUCTS USED INTERNALLY AND EXTERNALLY ------ */

type ConnnectionData struct {
	Uid  string
	Conn *websocket.Conn
}

type UserMessageData struct {
	Bytes       []byte
	Uid         string
	MessageType int
}

type UsersMessageData struct {
	Bytes       []byte
	Uids        []string
	MessageType int
}

type ConnMessageData struct {
	Bytes       []byte
	Conn        *websocket.Conn
	MessageType int
}

type ConnsMessageData struct {
	Bytes       []byte
	Conns       []*websocket.Conn
	MessageType int
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
	MessageType int
	Bytes       []byte
}

type SubscriptionsMessageData struct {
	SubNames    []string
	MessageType int
	Bytes       []byte
}

type SubscriptionMessageDataExcludeByWss struct {
	SubName      string
	MessageType  int
	Bytes        []byte
	ExcludeConns map[*websocket.Conn]struct{}
}

type SubscriptionMessageDataExcludeByIDs struct {
	SubName     string
	MessageType int
	Bytes       []byte
	ExcludeUids map[string]struct{}
}

type SubscriptionsMessageDataExcludeByWss struct {
	SubNames     []string
	MessageType  int
	Bytes        []byte
	ExcludeConns map[*websocket.Conn]struct{}
}

type SubscriptionsMessageDataExcludeByIDs struct {
	SubNames    []string
	MessageType int
	Bytes       []byte
	ExcludeUids map[string]struct{}
}

func Init() *SocketServer {
	ss := &SocketServer{
		ConnectionsByWs: ConnectionsByWs{
			data: map[*websocket.Conn]string{},
		},
		ConnectionsByID: ConnectionsByID{
			data: make(map[string]*websocket.Conn),
		},

		ConnectionSubscriptions: ConnectionSubscriptions{
			data: make(map[*websocket.Conn]map[string]struct{}),
		},

		RegisterConn:   make(<-chan ConnnectionData),
		UnregisterConn: make(<-chan *websocket.Conn),

		SendDataToUser:  make(<-chan UserMessageData),
		SendDataToConn:  make(<-chan ConnMessageData),
		SendDataToUsers: make(<-chan UsersMessageData),
		SendDataToConns: make(<-chan ConnsMessageData),

		SendDataToSubExcludeByWss:  make(<-chan SubscriptionMessageDataExcludeByWss),
		SendDataToSubExcludeByIDs:  make(<-chan SubscriptionMessageDataExcludeByIDs),
		SendDataToSubsExcludeByWss: make(<-chan SubscriptionsMessageDataExcludeByWss),
		SendDataToSubsExcludeByIDs: make(<-chan SubscriptionsMessageDataExcludeByIDs),

		JoinSubscriptionByWs:  make(<-chan RegisterUnregisterSubsConnWs),
		JoinSubscriptionByID:  make(<-chan RegisterUnregisterSubsConnID),
		LeaveSubscriptionByWs: make(<-chan RegisterUnregisterSubsConnWs),
		LeaveSubscriptionByID: make(<-chan RegisterUnregisterSubsConnID),

		SendDataToSub:  make(<-chan SubscriptionMessageData),
		SendDataToSubs: make(<-chan SubscriptionsMessageData),

		Subscriptions: Subscriptions{
			data: make(map[string]map[*websocket.Conn]struct{}),
		},
	}
	go runServer(ss)
	log.Println("Socket server initialized")
	return ss
}

func runServer(ss *SocketServer) {
	// Connection registration
	go connection(ss)
	// Disconnect registration
	go disconnect(ss)
	// Send data to user loop
	go sendUserData(ss)
	// Send data to a specific connection
	go sendConnData(ss)
	// Send data to users loop
	go sendUsersData(ss)
	// Send data to connections loop
	go sendConnsData(ss)
	// Join connection to subscription by matching websocket
	go joinSubsByWs(ss)
	// Join connection subscription by matching user id
	go joinSubsByID(ss)
	// Disconnect connection from subscription by matching websocket
	go leaveSubByWs(ss)
	// Disconnect connection from subscription by matching user id
	go leaveSubByID(ss)
	// Send data to subscription
	go sendSubData(ss)
	// Send data to subscriptions
	go sendSubsData(ss)
	// Send data to subscription excluding connection(s)
	go sendDataToSubExcludeWss(ss)
	// Send data to subscription excluding connection(s) by matching user ids
	go sendDataToSubExcludeIDs(ss)
	// Send data to multiple subscriptions excluding connection(s)
	go sendDataToSubsExcludeWss(ss)
	// Send data to multiple subscriptions excluding connection(s) by matching user ids
	go sendDataToSubsExcludeIDs(ss)
}

func connection(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws connection loop:", r)
				if failCount < 10 {
					go connection(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.RegisterConn
		ss.ConnectionsByWs.mutex.Lock()
		ss.ConnectionsByWs.data[data.Conn] = data.Uid
		if data.Uid != "" {
			ss.ConnectionsByID.mutex.Lock()
			ss.ConnectionsByID.data[data.Uid] = data.Conn
			ss.ConnectionsByID.mutex.Unlock()
		}
		ss.ConnectionsByWs.mutex.Unlock()

		ss.ConnectionSubscriptions.mutex.Lock()
		ss.ConnectionSubscriptions.data[data.Conn] = make(map[string]struct{})
		ss.ConnectionSubscriptions.mutex.Unlock()
	}
}

func disconnect(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws disconnect loop:", r)
				if failCount < 10 {
					go disconnect(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		conn := <-ss.UnregisterConn
		ss.ConnectionsByWs.mutex.Lock()
		if uid, ok := ss.ConnectionsByWs.data[conn]; ok {
			ss.ConnectionsByID.mutex.Lock()
			delete(ss.ConnectionsByID.data, uid)
			ss.ConnectionsByID.mutex.Unlock()

			ss.ConnectionSubscriptions.mutex.Lock()
			delete(ss.ConnectionSubscriptions.data, conn)
			ss.ConnectionSubscriptions.mutex.Unlock()
		}
		delete(ss.ConnectionsByWs.data, conn)
		ss.ConnectionsByWs.mutex.Unlock()
	}
}

func sendUserData(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws send user data loop:", r)
				if failCount < 10 {
					go sendUserData(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.SendDataToUser
		ss.ConnectionsByID.mutex.RLock()
		// Lock mutex is used to prevent multiple messages being sent to the same connection concurrently
		if conn, ok := ss.ConnectionsByID.data[data.Uid]; ok {
			conn.WriteMessage(data.MessageType, data.Bytes)
		}
		ss.ConnectionsByID.mutex.RUnlock()
	}
}

func sendUsersData(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws send users data loop:", r)
				if failCount < 10 {
					go sendUsersData(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.SendDataToUsers
		ss.ConnectionsByID.mutex.RLock()
		for _, v := range data.Uids {
			if conn, ok := ss.ConnectionsByID.data[v]; ok {
				conn.WriteMessage(data.MessageType, data.Bytes)
			}
		}
		ss.ConnectionsByID.mutex.RUnlock()
	}
}

func sendConnData(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws send conn data loop:", r)
				if failCount < 10 {
					go sendConnData(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.SendDataToConn
		// Lock mutex is used to prevent multiple messages being sent to the same connection concurrently
		ss.ConnectionsByWs.mutex.Lock()
		if _, ok := ss.ConnectionsByWs.data[data.Conn]; ok {
			data.Conn.WriteMessage(data.MessageType, data.Bytes)
		}
		ss.ConnectionsByWs.mutex.Unlock()
	}
}

func sendConnsData(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws send connections data loop:", r)
				if failCount < 10 {
					go sendConnsData(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.SendDataToConns
		ss.ConnectionsByWs.mutex.RLock()
		for _, conn := range data.Conns {
			if _, ok := ss.ConnectionsByWs.data[conn]; ok {
				conn.WriteMessage(data.MessageType, data.Bytes)
			}
		}
		ss.ConnectionsByWs.mutex.RUnlock()
	}
}

func joinSubsByWs(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws register subscription by ws connection loop:", r)
				if failCount < 10 {
					go joinSubsByWs(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.JoinSubscriptionByWs
		ss.Subscriptions.mutex.Lock()
		if _, ok := ss.Subscriptions.data[data.SubName]; ok {
			ss.Subscriptions.data[data.SubName][data.Conn] = struct{}{}
		} else {
			conns := make(map[*websocket.Conn]struct{})
			conns[data.Conn] = struct{}{}
			ss.Subscriptions.data[data.SubName] = conns
		}
		ss.Subscriptions.mutex.Unlock()

		ss.ConnectionSubscriptions.mutex.Lock()
		ss.ConnectionSubscriptions.data[data.Conn][data.SubName] = struct{}{}
		ss.ConnectionSubscriptions.mutex.Unlock()
	}
}

func joinSubsByID(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws register subscription by uid loop:", r)
				if failCount < 10 {
					go joinSubsByID(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.JoinSubscriptionByID
		ss.Subscriptions.mutex.Lock()
		ss.ConnectionsByID.mutex.RLock()
		if conn, ok := ss.ConnectionsByID.data[data.Uid]; ok {
			if _, ok := ss.Subscriptions.data[data.SubName]; ok {
				ss.Subscriptions.data[data.SubName][conn] = struct{}{}
			} else {
				conns := make(map[*websocket.Conn]struct{})
				conns[conn] = struct{}{}
				ss.Subscriptions.data[data.SubName] = conns
			}

			ss.ConnectionSubscriptions.mutex.Lock()
			ss.ConnectionSubscriptions.data[conn][data.SubName] = struct{}{}
			ss.ConnectionSubscriptions.mutex.Unlock()
		} else {
			log.Println("Could not register user ID to subscription - connection information not found in memory")
		}
		ss.ConnectionsByID.mutex.RUnlock()
		ss.Subscriptions.mutex.Unlock()
	}
}

func leaveSubByWs(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws leave subscription by ws conn loop:", r)
				if failCount < 10 {
					go leaveSubByWs(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.LeaveSubscriptionByWs
		ss.Subscriptions.mutex.RLock()
		if _, ok := ss.Subscriptions.data[data.SubName]; ok {
			ss.Subscriptions.mutex.Lock()
			delete(ss.Subscriptions.data[data.SubName], data.Conn)
			if len(ss.Subscriptions.data[data.SubName]) == 0 {
				delete(ss.Subscriptions.data, data.SubName)
			}
			ss.Subscriptions.mutex.Unlock()
		}
		ss.Subscriptions.mutex.RUnlock()

		ss.ConnectionSubscriptions.mutex.Lock()
		delete(ss.ConnectionSubscriptions.data[data.Conn], data.SubName)
		ss.ConnectionSubscriptions.mutex.Unlock()
	}
}

func leaveSubByID(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws leave subscription by uid loop:", r)
				if failCount < 10 {
					go leaveSubByWs(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.LeaveSubscriptionByID
		ss.Subscriptions.mutex.RLock()
		ss.ConnectionsByID.mutex.RLock()
		conn, connRegistered := ss.ConnectionsByID.data[data.Uid]
		if !connRegistered {
			ss.Subscriptions.mutex.RUnlock()
			ss.ConnectionsByID.mutex.RUnlock()
			continue
		}
		ss.ConnectionsByID.mutex.RUnlock()
		if _, ok := ss.Subscriptions.data[data.SubName]; ok {
			ss.Subscriptions.mutex.Lock()
			delete(ss.Subscriptions.data[data.SubName], conn)
			if len(ss.Subscriptions.data[data.SubName]) == 0 {
				delete(ss.Subscriptions.data, data.SubName)
			}
			ss.Subscriptions.mutex.Unlock()
		}
		ss.Subscriptions.mutex.RUnlock()

		ss.ConnectionSubscriptions.mutex.Lock()
		delete(ss.ConnectionSubscriptions.data[conn], data.SubName)
		ss.ConnectionSubscriptions.mutex.Unlock()
	}
}

func sendSubData(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws send data to subscription loop:", r)
				if failCount < 10 {
					go sendSubData(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.SendDataToSub
		ss.Subscriptions.mutex.RLock()
		if conns, ok := ss.Subscriptions.data[data.SubName]; ok {
			for c := range conns {
				if err := c.WriteMessage(data.MessageType, data.Bytes); err != nil {
					log.Println(err)
				}
			}
		}
		ss.Subscriptions.mutex.RUnlock()
	}
}

func sendSubsData(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws send data to subscriptions loop:", r)
				if failCount < 10 {
					go sendSubsData(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.SendDataToSubs
		ss.Subscriptions.mutex.RLock()
		for _, v := range data.SubNames {
			if _, ok := ss.Subscriptions.data[v]; ok {
				if conns, ok := ss.Subscriptions.data[v]; ok {
					for c := range conns {
						if err := c.WriteMessage(data.MessageType, data.Bytes); err != nil {
							log.Println(err)
						}
					}
				}
			}
		}
		ss.Subscriptions.mutex.RUnlock()
	}
}

func sendDataToSubExcludeWss(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws send data to subscription excluding connections loop:", r)
				if failCount < 10 {
					go sendDataToSubExcludeWss(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.SendDataToSubExcludeByWss
		ss.Subscriptions.mutex.RLock()
		if conns, ok := ss.Subscriptions.data[data.SubName]; ok {
			for c := range conns {
				if _, ok := data.ExcludeConns[c]; !ok {
					if err := c.WriteMessage(data.MessageType, data.Bytes); err != nil {
						log.Println(err)
					}
				}
			}
		}
		ss.Subscriptions.mutex.RUnlock()
	}
}

func sendDataToSubExcludeIDs(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws send data to subscription excluding user ids loop:", r)
				if failCount < 10 {
					go sendDataToSubExcludeIDs(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.SendDataToSubExcludeByIDs
		ss.Subscriptions.mutex.RLock()
		ss.ConnectionsByWs.mutex.RLock()
		if conns, ok := ss.Subscriptions.data[data.SubName]; ok {
			for c := range conns {
				if id, ok := ss.ConnectionsByWs.data[c]; ok {
					if _, ok := data.ExcludeUids[id]; !ok {
						if err := c.WriteMessage(data.MessageType, data.Bytes); err != nil {
							log.Println(err)
						}
					}
				}
			}
		}
		ss.ConnectionsByWs.mutex.RUnlock()
		ss.Subscriptions.mutex.RUnlock()
	}
}

func sendDataToSubsExcludeIDs(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws send data to subscriptions excluding user ids loop:", r)
				if failCount < 10 {
					go sendDataToSubsExcludeIDs(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.SendDataToSubsExcludeByIDs
		ss.Subscriptions.mutex.RLock()
		ss.ConnectionsByWs.mutex.RLock()
		for _, subName := range data.SubNames {
			if conns, ok := ss.Subscriptions.data[subName]; ok {
				for c := range conns {
					if id, ok := ss.ConnectionsByWs.data[c]; ok {
						if _, ok := data.ExcludeUids[id]; !ok {
							if err := c.WriteMessage(data.MessageType, data.Bytes); err != nil {
								log.Println(err)
							}
						}
					}
				}
			}
		}
		ss.ConnectionsByWs.mutex.RUnlock()
		ss.Subscriptions.mutex.RUnlock()
	}
}

func sendDataToSubsExcludeWss(ss *SocketServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				failCount++
				log.Println("Recovered from panic in ws send data to subscriptions excluding connections loop:", r)
				if failCount < 10 {
					go sendDataToSubsExcludeWss(ss)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
			}
		}()

		data := <-ss.SendDataToSubsExcludeByWss
		ss.Subscriptions.mutex.RLock()
		for _, subName := range data.SubNames {
			if conns, ok := ss.Subscriptions.data[subName]; ok {
				for c := range conns {
					if _, ok := data.ExcludeConns[c]; !ok {
						if err := c.WriteMessage(data.MessageType, data.Bytes); err != nil {
							log.Println(err)
						}
					}
				}
			}
		}
		ss.Subscriptions.mutex.RUnlock()
	}
}