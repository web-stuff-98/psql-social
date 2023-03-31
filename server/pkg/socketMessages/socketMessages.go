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

// TYPE: CHANGE_EVENT
type ChangeEvent struct {
	// UPDATE/DELETE/INSERT/UPDATE_IMAGE
	Type   string `json:"change_type"`
	Entity string `json:"entity"`
	// "ID" should be included in Data
	Data map[string]interface{} `json:"data"`
}
