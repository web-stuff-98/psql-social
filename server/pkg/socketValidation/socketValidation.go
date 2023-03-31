package socketvalidation

// JOIN_ROOM/LEAVE_ROOM
type JoinLeaveRoomData struct {
	RoomID string `json:"room_id" validate:"required,lte=36"`
}

// ROOM_MESSAGE
type RoomMessage struct {
	Content   string `json:"content" validate:"required,lte=200"`
	ChannelID string `json:"channel_id" validate:"required,lte=36"`
}

// ROOM_MESSAGE_UPDATE
type RoomMessageUpdate struct {
	Content string `json:"content" validate:"required,lte=200"`
	MsgID   string `json:"msg_id" validate:"required,lte=36"`
}

// ROOM_MESSAGE_DELETE
type RoomMessageDelete struct {
	MsgID string `json:"msg_id" validate:"required,lte=36"`
}

// START_WATCHING/STOP_WATCHING
type StartStopWatching struct {
	ID     string `json:"id" validate:"required,lte=36"`
	Entity string `json:"entity" validate:"required,lte=4"`
}
