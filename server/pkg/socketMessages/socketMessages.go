package socketmessages

/* This is for Outbound messages*/

// TYPE: ROOM_MESSAGE
type RoomMessage struct {
	ID        string `json:"ID"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	AuthorID  string `json:"author_id"`
}

// TYPE: ROOM_MESSAGE_UPDATE
type RoomMessageUpdate struct {
	ID      string `json:"ID"`
	Content string `json:"content"`
}

// TYPE: ROOM_MESSAGE_DELETE
type RoomMessageDelete struct {
	ID string `json:"ID"`
}

// TYPE: DIRECT_MESSAGE
type DirectMessage struct {
	ID        string `json:"ID"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	AuthorID  string `json:"author_id"`
}

// TYPE: DIRECT_MESSAGE_UPDATE
type DirectMessageUpdate struct {
	ID      string `json:"ID"`
	Content string `json:"content"`
}

// TYPE: DIRECT_MESSAGE_DELETE
type DirectMessageDelete struct {
	ID string `json:"ID"`
}

// TYPE: FRIEND_REQUEST
type FriendRequest struct {
	Friender string `json:"friender"`
	Friended string `json:"friended"`
}

// TYPE: FRIEND_REQUEST_RESPONSE
type FriendRequestResponse struct {
	Accepted bool   `json:"accepted"`
	Friended string `json:"friended"`
}

// TYPE: CHANGE_EVENT
type ChangeEvent struct {
	// UPDATE/DELETE/INSERT/UPDATE_IMAGE
	Type   string `json:"change_type"`
	Entity string `json:"entity"`
	// "ID" should be included in Data
	Data map[string]interface{} `json:"data"`
}
