package socketserver

import (
	"log"
	"sync"

	"github.com/fasthttp/websocket"
)

type SocketServer struct {
	Connections Connections

	RegisterConn   <-chan ConnnectionData
	UnregisterConn <-chan *websocket.Conn
}

/* ------ MUTEX PROTECTED ------ */

type Connections struct {
	data  map[*websocket.Conn]string
	mutex sync.RWMutex
}

/* ------ GENERAL STRUCTS ------ */

type ConnnectionData struct {
	Uid  string
	Conn *websocket.Conn
}

func Init() *SocketServer {
	ss := &SocketServer{
		Connections: Connections{
			data: map[*websocket.Conn]string{},
		},

		RegisterConn:   make(<-chan ConnnectionData),
		UnregisterConn: make(<-chan *websocket.Conn),
	}
	go runServer(ss)
	return ss
}

func runServer(ss *SocketServer) {
	// Connection registration loop
	go connectionLoop(ss)
	// Disconnect registration loop
	go disconnectLoop(ss)
}

func connectionLoop(ss *SocketServer) {
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws connection loop:", r)
				go connectionLoop(ss)
			}
		}()
		data := <-ss.RegisterConn
		ss.Connections.mutex.Lock()
		ss.Connections.data[data.Conn] = data.Uid
		ss.Connections.mutex.Unlock()
	}
}

func disconnectLoop(ss *SocketServer) {
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in ws disconnect loop:", r)
				go disconnectLoop(ss)
			}
		}()
		conn := <-ss.UnregisterConn
		ss.Connections.mutex.Lock()
		delete(ss.Connections.data, conn)
		ss.Connections.mutex.Unlock()
	}
}
