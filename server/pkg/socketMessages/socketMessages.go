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
	ID          string `json:"ID"`
	Content     string `json:"content"`
	CreatedAt   string `json:"created_at"`
	AuthorID    string `json:"author_id"`
	RecipientID string `json:"recipient_id"`
}

// TYPE: DIRECT_MESSAGE_UPDATE
type DirectMessageUpdate struct {
	ID          string `json:"ID"`
	Content     string `json:"content"`
	AuthorID    string `json:"author_id"`
	RecipientID string `json:"recipient_id"`
}

// TYPE: DIRECT_MESSAGE_DELETE
type DirectMessageDelete struct {
	ID          string `json:"ID"`
	AuthorID    string `json:"author_id"`
	RecipientID string `json:"recipient_id"`
}

// TYPE: FRIEND_REQUEST
type FriendRequest struct {
	Friender  string `json:"friender"`
	Friended  string `json:"friended"`
	CreatedAt string `json:"created_at"`
}

// TYPE: FRIEND_REQUEST_RESPONSE
type FriendRequestResponse struct {
	Accepted bool   `json:"accepted"`
	Friender string `json:"friender"`
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

// TYPE: INVITATION
type Invitation struct {
	Inviter   string `json:"inviter"`
	Invited   string `json:"invited"`
	RoomID    string `json:"room_id"`
	CreatedAt string `json:"created_at"`
}

// TYPE: INVITATION_RESPONSE
type InvitationResponse struct {
	Invited  string `json:"invited"`
	Inviter  string `json:"inviter"`
	Accepted bool   `json:"accepted"`
	RoomID   string `json:"room_id"`
}
