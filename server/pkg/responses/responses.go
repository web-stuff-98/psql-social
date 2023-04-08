package responses

/* ----------------- HTTP RESPONSES ----------------- */

type User struct {
	ID       string `json:"ID"`
	Username string `json:"username"`
	Online   bool   `json:"online"`
	// "ADMIN" | "USER"
	Role string `json:"role"`
}

type Room struct {
	ID        string `json:"ID"`
	Name      string `json:"name"`
	AuthorID  string `json:"author_id"`
	Private   bool   `json:"is_private"`
	CreatedAt string `json:"created_at"`
}

type RoomChannelBase struct {
	ID   string `json:"ID"`
	Name string `json:"name"`
	Main bool   `json:"main"`
}

type RoomMessage struct {
	ID            string `json:"ID"`
	Content       string `json:"content"`
	AuthorID      string `json:"author_id"`
	CreatedAt     string `json:"created_at"`
	HasAttachment bool   `json:"has_attachment"`
}

type RoomChannel struct {
	Messages      []RoomMessage `json:"messages"`
	UsersInWebRTC []string      `json:"users_in_webrtc"`
}

type DirectMessage struct {
	ID            string `json:"ID"`
	Content       string `json:"content"`
	AuthorID      string `json:"author_id"`
	RecipientID   string `json:"recipient_id"`
	CreatedAt     string `json:"created_at"`
	HasAttachment bool   `json:"has_attachment"`
}

type Invitation struct {
	Inviter   string `json:"inviter"`
	Invited   string `json:"invited"`
	CreatedAt string `json:"created_at"`
	RoomID    string `json:"room_id"`
}

type FriendRequest struct {
	Friender  string `json:"friender"`
	Friended  string `json:"friended"`
	CreatedAt string `json:"created_at"`
}

type Conversation struct {
	DirectMessages []DirectMessage `json:"direct_messages"`
	Invitations    []Invitation    `json:"invitations"`
	FriendRequests []FriendRequest `json:"friend_requests"`
}

type AttachmentMetadata struct {
	ID     string  `json:"ID"`
	Meta   string  `json:"meta"`
	Name   string  `json:"name"`
	Size   int     `json:"size"`
	Ratio  float32 `json:"ratio"`
	Failed bool    `json:"failed"`
}
