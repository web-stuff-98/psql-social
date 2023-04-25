package socketlimiter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/redis/go-redis/v9"
)

/*
	Data for each connection is stored on redis as a
	map containing SocketConnectionLimiterData for
	each eventType name. The data is keyed as the
	connections remote address suffixed with
	"socket-limiter-data:". It expires after 2 minutes.
*/

type SocketLimiter struct {
	SocketEvent   chan SocketEvent
	Configuration map[string]EventLimiterConfiguration
}

type EventLimiterConfiguration struct {
	Window        time.Duration
	MaxReqs       uint16
	BlockDuration time.Duration
	Message       string
}

/* --------------- EVENT MODELS --------------- */
type SocketEvent struct {
	RecvChan chan error
	Type     string
	Conn     *websocket.Conn
}

/* --------------- REDIS JSON --------------- */
type SocketConnectionLimiterData struct {
	LastRequest      time.Time `json:"last_req"`
	RequestsInWindow uint16    `json:"reqs"`
}

func configure() map[string]EventLimiterConfiguration {
	config := make(map[string]EventLimiterConfiguration)

	watchEventConfig := EventLimiterConfiguration{
		Window:        time.Second,
		MaxReqs:       100,
		BlockDuration: time.Minute,
		Message:       "Too many requests",
	}

	messageEventConfig := EventLimiterConfiguration{
		Window:        time.Second * 5,
		MaxReqs:       5,
		BlockDuration: time.Minute * 2,
		Message:       "Too many messages. Wait 2 minutes.",
	}

	generalEventConfig := EventLimiterConfiguration{
		Window:        time.Second * 10,
		MaxReqs:       80,
		BlockDuration: time.Minute * 2,
		Message:       "Too many requests",
	}

	config["JOIN_ROOM"] = generalEventConfig
	config["LEAVE_ROOM"] = generalEventConfig

	config["JOIN_CHANNEL"] = generalEventConfig
	config["LEAVE_CHANNEL"] = generalEventConfig

	config["ROOM_MESSAGE"] = messageEventConfig
	config["ROOM_MESSAGE_UPDATE"] = messageEventConfig
	config["ROOM_MESSAGE_DELETE"] = messageEventConfig
	config["DIRECT_MESSAGE"] = messageEventConfig
	config["DIRECT_MESSAGE_UPDATE"] = messageEventConfig
	config["DIRECT_MESSAGE_DELETE"] = messageEventConfig

	config["START_WATCHING"] = watchEventConfig
	config["STOP_WATCHING"] = watchEventConfig

	config["FRIEND_REQUEST"] = generalEventConfig
	config["FRIEND_REQUEST_RESPONSE"] = generalEventConfig
	config["INVITATION"] = generalEventConfig
	config["INVITATION_RESPONSE"] = generalEventConfig

	config["BLOCK"] = generalEventConfig
	config["UNBLOCK"] = generalEventConfig

	config["BAN"] = generalEventConfig
	config["UNBAN"] = generalEventConfig

	config["CALL_USER"] = generalEventConfig
	config["CALL_USER_RESPONSE"] = generalEventConfig
	config["CALL_WEBRTC_OFFER"] = generalEventConfig
	config["CALL_WEBRTC_ANSWER"] = generalEventConfig
	config["CALL_WEBRTC_RECIPIENT_REQUEST_REINITIALIZATION"] = generalEventConfig
	config["CALL_UPDATE_MEDIA_OPTIONS"] = generalEventConfig

	config["CHANNEL_WEBRTC_UPDATE_MEDIA_OPTIONS"] = generalEventConfig
	config["CHANNEL_WEBRTC_SENDING_SIGNAL"] = generalEventConfig
	config["CHANNEL_WEBRTC_RETURNING_SIGNAL"] = generalEventConfig
	config["CHANNEL_WEBRTC_JOIN"] = generalEventConfig
	config["CHANNEL_WEBRTC_LEAVE"] = generalEventConfig

	return config
}

func Init(redisClient *redis.Client) *SocketLimiter {
	config := configure()
	sl := &SocketLimiter{
		SocketEvent:   make(chan SocketEvent),
		Configuration: config,
	}
	go runLimiter(redisClient, sl)
	return sl
}

func runLimiter(redisClient *redis.Client, sl *SocketLimiter) {
	go socketEventRegistration(redisClient, sl)
}

// shorthand function for setting redis key, could be moved into redis helper package if needed somewhere else, using interface instead of map
func set(redisClient *redis.Client, address string, value map[string]SocketConnectionLimiterData) error {
	if bytes, err := json.Marshal(value); err != nil {
		return err
	} else {
		if _, err := redisClient.SetEx(context.Background(), "socket-limiter-data:"+address, string(bytes), time.Minute*2).Result(); err != nil {
			log.Println("Redis internal error handling socket limiter:", err)
			return err
		}
	}
	return nil
}

func socketEventRegistration(redisClient *redis.Client, sl *SocketLimiter) {
	for {
		eventData := <-sl.SocketEvent

		// bypass limiter for development mode
		if os.Getenv("ENVIRONMENT") != "PRODUCTION" {
			eventData.RecvChan <- nil
			continue
		}

		keyVal := make(map[string]SocketConnectionLimiterData)
		if rawVal, err := redisClient.Get(context.Background(), "socket-limiter-data:"+eventData.Conn.RemoteAddr().String()).Result(); err != nil {
			if err != redis.Nil {
				log.Println("Redis internal error handling socket limiter:", err)
				eventData.RecvChan <- fmt.Errorf("Internal error")
				continue
			}
			// data wasn't found on redis. Create it and continue. Connection will not be limited, since it's only sent one event
			keyVal[eventData.Type] = SocketConnectionLimiterData{
				LastRequest:      time.Now(),
				RequestsInWindow: 1,
			}
			if err := set(redisClient, eventData.Conn.RemoteAddr().String(), keyVal); err != nil {
				eventData.RecvChan <- err
			} else {
				eventData.RecvChan <- nil
			}
			continue
		} else {
			// data was found on redis. Unmarshal it.
			if err := json.Unmarshal([]byte(rawVal), &keyVal); err != nil {
				log.Println("Error unmarshalling connection socket limiter data:", err)
				eventData.RecvChan <- fmt.Errorf("Internal error")
				continue
			}
		}
		if data, ok := keyVal[eventData.Type]; !ok {
			keyVal[eventData.Type] = SocketConnectionLimiterData{
				LastRequest:      time.Now(),
				RequestsInWindow: 1,
			}
			if err := set(redisClient, eventData.Conn.RemoteAddr().String(), keyVal); err != nil {
				eventData.RecvChan <- err
			} else {
				eventData.RecvChan <- nil
			}
		} else {
			if config, ok := sl.Configuration[eventData.Type]; !ok {
				log.Println("Limiter configuration not found for socket event type ", eventData.Type, ". Configuration needs to be added")
				eventData.RecvChan <- fmt.Errorf("Limiter configuration not found for socket event type")
				continue
			} else {
				// check if connection has already exceeded the rate limiter
				if data.RequestsInWindow > config.MaxReqs && data.LastRequest.Add(config.BlockDuration).Before(time.Now()) {
					// need to set the value again so that the key doesn't expire
					if err := set(redisClient, eventData.Conn.RemoteAddr().String(), keyVal); err != nil {
						eventData.RecvChan <- err
					} else {
						eventData.RecvChan <- fmt.Errorf(config.Message)
					}
					continue
				}
				// connection has not exceeded the rate limiter already
				if data.LastRequest.Before(time.Now().Add(-config.Window)) {
					keyVal[eventData.Type] = SocketConnectionLimiterData{
						LastRequest:      time.Now(),
						RequestsInWindow: 1,
					}
					eventData.RecvChan <- nil
				} else {
					keyVal[eventData.Type] = SocketConnectionLimiterData{
						LastRequest:      time.Now(),
						RequestsInWindow: data.RequestsInWindow + 1,
					}
					if keyVal[eventData.Type].RequestsInWindow > config.MaxReqs {
						// need to set the value again so that the key doesn't expire
						if err := set(redisClient, eventData.Conn.RemoteAddr().String(), keyVal); err != nil {
							eventData.RecvChan <- err
						} else {
							eventData.RecvChan <- fmt.Errorf(config.Message)
						}
					} else {
						eventData.RecvChan <- nil
					}
				}
				if err := set(redisClient, eventData.Conn.RemoteAddr().String(), keyVal); err != nil {
					eventData.RecvChan <- err
				}
				continue
			}
		}
	}
}
