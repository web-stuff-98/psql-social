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

// DIRECT_MESSAGE
type DirectMessage struct {
	Content string `json:"content" validate:"required,lte=200"`
	Uid     string `json:"uid" validate:"required,lte=36"`
}

// DIRECT_MESSAGE_UPDATE
type DirectMessageUpdate struct {
	Content string `json:"content" validate:"required,lte=200"`
	MsgID   string `json:"msg_id" validate:"required,lte=36"`
}

// DIRECT_MESSAGE_DELETE
type DirectMessageDelete struct {
	MsgID string `json:"msg_id" validate:"required,lte=36"`
}

// FRIEND_REQUEST
type FriendRequest struct {
	Uid string `json:"uid" validate:"required,lte=36"`
}

// FRIEND_REQUEST_RESPONSE
type FriendRequestResponse struct {
	Friender string `json:"friender" validate:"required,lte=36"`
	Accepted bool   `json:"accepted"`
}

// INVITATION
type Invitation struct {
	RoomID string `json:"room_id" validation:"required,lte=36"`
	Uid    string `json:"uid" validation:"required,lte=36"`
}

// INVITATION_RESPONSE
type InvitationResponse struct {
	Inviter  string `json:"inviter" validation:"required,lte=36"`
	RoomID   string `json:"room_id" validation:"required,lte=36"`
	Accepted bool   `json:"accepted"`
}
