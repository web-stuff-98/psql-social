package socketvalidation

// JOIN_ROOM/LEAVE_ROOM
type JoinLeaveRoomData struct {
	RoomID string `json:"room_id" validate:"required,lte=36"`
}
