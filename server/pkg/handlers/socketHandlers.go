package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/fasthttp/websocket"
	"github.com/go-playground/validator/v10"
	socketvalidation "github.com/web-stuff-98/psql-social/socketValidation"
)

func handleSocketEvent(data map[string]interface{}, event string, h handler, c *websocket.Conn) error {
	var err error

	switch event {
	case "JOIN_ROOM":
		err = joinRoom(data, h, c)
	case "LEAVE_ROOM":
		err = leaveRoom(data, h, c)
	default:
		return fmt.Errorf("Unrecognized event type")
	}

	return err
}

func joinRoom(inData map[string]interface{}, h handler, c *websocket.Conn) error {
	v := validator.New()
	if err := v.Struct(inData); err != nil {
		return fmt.Errorf("Bad request")
	}
	data := &socketvalidation.JoinLeaveRoomData{}
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("Bad request")
	}
	err = json.Unmarshal(b, &data)
	if err != nil {
		return fmt.Errorf("Bad request")
	}

	return nil
}

func leaveRoom(inData map[string]interface{}, h handler, c *websocket.Conn) error {
	v := validator.New()
	if err := v.Struct(inData); err != nil {
		return fmt.Errorf("Bad request")
	}
	data := &socketvalidation.JoinLeaveRoomData{}
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("Bad request")
	}
	err = json.Unmarshal(b, &data)
	if err != nil {
		return fmt.Errorf("Bad request")
	}

	return nil
}
