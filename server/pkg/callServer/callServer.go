package callserver

import (
	"log"
	"sync"
	"time"

	socketMessages "github.com/web-stuff-98/psql-social/pkg/socketMessages"
	"github.com/web-stuff-98/psql-social/pkg/socketServer"
)

/*
	For 1-1 calls, handles
	WebRTC events also
*/

type CallServer struct {
	// Calls that have not yet been answered
	CallsPending CallsPending
	// Channel for creating pending calls
	CallsPendingChan chan InCall
	// Channel for responding to pending calls
	ResponseToCallChan chan InCallResponse
	// Mutex protected map for active calls
	CallsActive CallsActive
	// Channel for closing active calls
	LeaveCallChan chan string
	// Channel for sending call recipient offer
	SendCallRecipientOffer chan CallerSignal
	// Channel for sending answer from called back to caller
	SendCalledAnswer chan CalledSignal
	// Channel for recipient requesting WebRTC re-initialization (necessary for changing/adding media devices)
	CallRecipientRequestedReInitialization chan string
	// Channel for updating media options
	UpdateMediaOptions chan UpdateMediaOptions
}

/* --------------- MUTEX PROTECTED MAPS --------------- */
type CallsPending struct {
	// outer map is caller ID, inner map is the user that was called ID
	data  map[string]string
	mutex sync.Mutex
}
type CallsActive struct {
	// outer map is caller ID, inner map is the user that was called ID
	data  map[string]string
	mutex sync.Mutex
}

/* --------------- STRUCTS --------------- */
type CallerSignal struct {
	Caller string
	Signal string

	UserMediaStreamID string
	UserMediaVid      bool
	DisplayMediaVid   bool
}
type CalledSignal struct {
	Called string
	Signal string

	UserMediaStreamID string
	UserMediaVid      bool
	DisplayMediaVid   bool
}
type InCall struct {
	Caller string
	Called string
}
type InCallResponse struct {
	Caller string
	Called string
	Accept bool
}
type UpdateMediaOptions struct {
	Uid string

	UserMediaStreamID string
	UserMediaVid      bool
	DisplayMediaVid   bool
}

func Init(ss *socketServer.SocketServer, dc chan string) *CallServer {
	cs := &CallServer{
		CallsPending: CallsPending{
			data: make(map[string]string),
		},
		CallsPendingChan:   make(chan InCall),
		ResponseToCallChan: make(chan InCallResponse),
		CallsActive: CallsActive{
			data: make(map[string]string),
		},
		LeaveCallChan:                          make(chan string),
		SendCallRecipientOffer:                 make(chan CallerSignal),
		SendCalledAnswer:                       make(chan CalledSignal),
		CallRecipientRequestedReInitialization: make(chan string),
		UpdateMediaOptions:                     make(chan UpdateMediaOptions),
	}
	runServer(ss, cs, dc)
	return cs
}

func runServer(ss *socketServer.SocketServer, cs *CallServer, dc chan string) {
	go callPending(ss, cs)
	go callResponse(ss, cs)
	go leaveCall(ss, cs)
	go sendCallRecipientOffer(ss, cs)
	go sendCallerAnswer(ss, cs)
	go callRecipientRequestReInitialization(ss, cs)
	go updateMediaOptions(ss, cs)
	go socketDisconnect(ss, cs, dc)
}

func callPending(ss *socketServer.SocketServer, cs *CallServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in call server pending calls loop:", r)
				if failCount < 10 {
					go callPending(ss, cs)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-cs.CallsPendingChan
		cs.CallsPending.mutex.Lock()
		if called, ok := cs.CallsPending.data[data.Caller]; ok {
			if called != data.Called {
				// pending call switching to different user. cancel previous pending call.
				Uids := []string{called, data.Caller}
				ss.SendDataToUsers <- socketServer.UsersMessageData{
					Uids: Uids,
					Data: socketMessages.CallResponse{
						Called: data.Called,
						Caller: data.Caller,
						Accept: false,
					},
					MessageType: "CALL_USER_RESPONSE",
				}
				cs.CallsPending.data[data.Caller] = data.Called
			}
		} else {
			cs.CallsPending.data[data.Caller] = data.Called
		}
		Uids := []string{data.Called, data.Caller}
		ss.SendDataToUsers <- socketServer.UsersMessageData{
			Uids: Uids,
			Data: socketMessages.CallAcknowledge{
				Caller: data.Caller,
				Called: data.Called,
			},
			MessageType: "CALL_USER_ACKNOWLEDGE",
		}

		log.Println("Sent CALL_USER_ACKNOWLEDGE event")

		cs.CallsPending.mutex.Unlock()
		go timeoutCall(ss, cs, data.Caller)
	}
}

func timeoutCall(ss *socketServer.SocketServer, cs *CallServer, uid string) {
	time.Sleep(time.Second * 15)
	cs.CallsPending.mutex.Lock()
	if callPending, ok := cs.CallsPending.data[uid]; ok {
		uids := []string{callPending, uid}
		ss.SendDataToUsers <- socketServer.UsersMessageData{
			Uids: uids,
			Data: socketMessages.CallResponse{
				Caller: uid,
				Called: callPending,
				Accept: false,
			},
			MessageType: "CALL_USER_RESPONSE",
		}
		delete(cs.CallsPending.data, uid)
	}
	cs.CallsPending.mutex.Unlock()
}

func callResponse(ss *socketServer.SocketServer, cs *CallServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in call server call response loop:", r)
				if failCount < 10 {
					go callResponse(ss, cs)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-cs.ResponseToCallChan
		cs.CallsPending.mutex.Lock()
		cs.CallsActive.mutex.Lock()
		delete(cs.CallsPending.data, data.Caller)

		if data.Accept {
			// Close any call that either user is currently in.
			// Clients can only be in a single call.
			// Confusing variable names here.
			closedCallerCall := false
			closedCalledCall := false
			if callerCalled, ok := cs.CallsActive.data[data.Caller]; ok {
				closedCallerCall = true
				Uids := []string{callerCalled, data.Caller}
				ss.SendDataToUsers <- socketServer.UsersMessageData{
					Uids:        Uids,
					Data:        socketMessages.CallLeft{},
					MessageType: "CALL_LEFT",
				}
				delete(cs.CallsActive.data, data.Caller)
			}
			if calledCalled, ok := cs.CallsActive.data[data.Called]; ok {
				closedCalledCall = true
				Uids := []string{calledCalled, data.Called}
				ss.SendDataToUsers <- socketServer.UsersMessageData{
					Uids:        Uids,
					Data:        socketMessages.CallLeft{},
					MessageType: "CALL_LEFT",
				}
				delete(cs.CallsActive.data, data.Called)
			}
			// make sure that the caller is not in a call. If they are exit the call they are already in
			if !closedCallerCall {
				for caller, called := range cs.CallsActive.data {
					if data.Caller == called {
						Uids := []string{called, caller}
						ss.SendDataToUsers <- socketServer.UsersMessageData{
							Data:        socketMessages.CallLeft{},
							Uids:        Uids,
							MessageType: "CALL_LEFT",
						}
						delete(cs.CallsActive.data, caller)
						break
					}
				}
			}
			// make sure that the called user is not in a call. If they are exit the call they are already in
			if !closedCalledCall {
				for caller, called := range cs.CallsActive.data {
					if data.Called == called {
						Uids := []string{caller, called}
						ss.SendDataToUsers <- socketServer.UsersMessageData{
							Data:        socketMessages.CallLeft{},
							Uids:        Uids,
							MessageType: "CALL_LEFT",
						}
						delete(cs.CallsActive.data, caller)
						break
					}
				}
			}

			// Any active calls that either user in have now been closed. Proceed.
			cs.CallsActive.data[data.Caller] = data.Called
		}

		// Send the response to both clients
		Uids := []string{data.Called, data.Caller}
		ss.SendDataToUsers <- socketServer.UsersMessageData{
			Uids: Uids,
			Data: socketMessages.CallResponse{
				Caller: data.Caller,
				Called: data.Called,
				Accept: data.Accept,
			},
			MessageType: "CALL_USER_RESPONSE",
		}

		cs.CallsActive.mutex.Unlock()
		cs.CallsPending.mutex.Unlock()
	}
}

func leaveCall(ss *socketServer.SocketServer, cs *CallServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in call server leave calls loop:", r)
				if failCount < 10 {
					go leaveCall(ss, cs)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		uid := <-cs.LeaveCallChan
		cs.CallsActive.mutex.Lock()
		if called, ok := cs.CallsActive.data[uid]; ok {
			ss.SendDataToUser <- socketServer.UserMessageData{
				MessageType: "CALL_LEFT",
				Data:        socketMessages.CallLeft{},
				Uid:         called,
			}
			delete(cs.CallsActive.data, uid)
		} else {
			for caller, called := range cs.CallsActive.data {
				if called == uid {
					ss.SendDataToUser <- socketServer.UserMessageData{
						MessageType: "CALL_LEFT",
						Data:        socketMessages.CallLeft{},
						Uid:         caller,
					}
					delete(cs.CallsActive.data, caller)
					break
				}
			}
		}
		cs.CallsActive.mutex.Unlock()
	}
}

func sendCallRecipientOffer(ss *socketServer.SocketServer, cs *CallServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in call server send call recipient offer loop:", r)
				if failCount < 10 {
					go sendCallRecipientOffer(ss, cs)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-cs.SendCallRecipientOffer
		cs.CallsActive.mutex.Lock()
		if called, ok := cs.CallsActive.data[data.Caller]; ok {
			ss.SendDataToUser <- socketServer.UserMessageData{
				Uid:         called,
				MessageType: "CALL_WEBRTC_OFFER_FROM_INITIATOR",
				Data: socketMessages.CallWebRTCOfferFromInitiator{
					Signal: data.Signal,

					UserMediaStreamID: data.UserMediaStreamID,
					UserMediaVid:      data.UserMediaVid,
					DisplayMediaVid:   data.DisplayMediaVid,
				},
			}
		}
		cs.CallsActive.mutex.Unlock()
	}
}

func sendCallerAnswer(ss *socketServer.SocketServer, cs *CallServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in call server send caller answer loop:", r)
				if failCount < 10 {
					go sendCallerAnswer(ss, cs)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-cs.SendCalledAnswer
		cs.CallsActive.mutex.Lock()
		for caller, oi2 := range cs.CallsActive.data {
			if oi2 == data.Called {
				ss.SendDataToUser <- socketServer.UserMessageData{
					Uid:         caller,
					MessageType: "CALL_WEBRTC_ANSWER_FROM_RECIPIENT",
					Data: socketMessages.CallWebRTCOfferAnswer{
						Signal: data.Signal,

						UserMediaStreamID: data.UserMediaStreamID,
						UserMediaVid:      data.UserMediaVid,
						DisplayMediaVid:   data.DisplayMediaVid,
					},
				}
				break
			}
		}
		cs.CallsActive.mutex.Unlock()
	}
}

func callRecipientRequestReInitialization(ss *socketServer.SocketServer, cs *CallServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in call server request WebRTC reinitialization loop:", r)
				if failCount < 10 {
					go callRecipientRequestReInitialization(ss, cs)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		callerId := <-cs.CallRecipientRequestedReInitialization
		cs.CallsActive.mutex.Lock()
		for caller, uid := range cs.CallsActive.data {
			if uid == callerId {
				ss.SendDataToUser <- socketServer.UserMessageData{
					Uid:         caller,
					MessageType: "CALL_WEBRTC_REQUESTED_REINITIALIZATION",
					Data:        socketMessages.CallWebRTCRequestedReInitialization{},
				}
				break
			}
		}
		cs.CallsActive.mutex.Unlock()
	}
}

func updateMediaOptions(ss *socketServer.SocketServer, cs *CallServer) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in call server update media options loop:", r)
				if failCount < 10 {
					go updateMediaOptions(ss, cs)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		data := <-cs.UpdateMediaOptions
		cs.CallsActive.mutex.Lock()
		if recipient, ok := cs.CallsActive.data[data.Uid]; ok {
			ss.SendDataToUser <- socketServer.UserMessageData{
				MessageType: "UPDATE_MEDIA_OPTIONS_OUT",
				Uid:         recipient,
				Data: socketMessages.UpdateMediaOptions{
					UserMediaVid:      data.UserMediaVid,
					DisplayMediaVid:   data.DisplayMediaVid,
					UserMediaStreamID: data.UserMediaStreamID,
				},
			}
		} else {
			for caller, called := range cs.CallsActive.data {
				if called == data.Uid {
					ss.SendDataToUser <- socketServer.UserMessageData{
						MessageType: "UPDATE_MEDIA_OPTIONS_OUT",
						Uid:         caller,
						Data: socketMessages.UpdateMediaOptions{
							UserMediaVid:      data.UserMediaVid,
							DisplayMediaVid:   data.DisplayMediaVid,
							UserMediaStreamID: data.UserMediaStreamID,
						},
					}
					break
				}
			}
		}
		cs.CallsActive.mutex.Unlock()
	}
}

func socketDisconnect(ss *socketServer.SocketServer, cs *CallServer, dc chan string) {
	var failCount uint8
	for {
		defer func() {
			r := recover()
			if r != nil {
				log.Println("Recovered from panic in call server socket disconnect loop:", r)
				if failCount < 10 {
					go socketDisconnect(ss, cs, dc)
				} else {
					log.Println("Panic recovery count in ws loop exceeded maximum. Loop will not recover.")
				}
				failCount++
			}
		}()

		uid := <-dc
		cs.CallsPending.mutex.Lock()
		cs.CallsActive.mutex.Lock()
		if callPending, ok := cs.CallsPending.data[uid]; ok {
			ss.SendDataToUser <- socketServer.UserMessageData{
				Uid:         callPending,
				MessageType: "CALL_USER_RESPONSE",
				Data: socketMessages.CallResponse{
					Caller: uid,
					Called: callPending,
					Accept: false,
				},
			}
			delete(cs.CallsActive.data, uid)
		}
		for caller, called := range cs.CallsPending.data {
			if called == uid {
				ss.SendDataToUser <- socketServer.UserMessageData{
					Uid:         caller,
					MessageType: "CALL_USER_RESPONSE",
					Data: socketMessages.CallResponse{
						Caller: caller,
						Called: uid,
						Accept: false,
					},
				}
				delete(cs.CallsPending.data, caller)
			}
		}

		if called, ok := cs.CallsActive.data[uid]; ok {
			ss.SendDataToUser <- socketServer.UserMessageData{
				Uid:         called,
				MessageType: "CALL_LEFT",
				Data:        socketMessages.CallLeft{},
			}
			delete(cs.CallsActive.data, uid)
		} else {
			for caller, called := range cs.CallsActive.data {
				if called == uid {
					ss.SendDataToUser <- socketServer.UserMessageData{
						MessageType: "CALL_LEFT",
						Uid:         caller,
						Data:        socketMessages.CallLeft{},
					}
					delete(cs.CallsActive.data, caller)
					break
				}
			}
		}
		cs.CallsActive.mutex.Unlock()
		cs.CallsPending.mutex.Unlock()
	}
}
