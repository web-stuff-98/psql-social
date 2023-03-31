package socketmessages

/* This is for Outbound messages*/

// TYPE: ROOM_MESSAGE
type RoomMessage struct {
	ID        string `json:"ID"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	AuthorID  string `json:"author_id"`
}
