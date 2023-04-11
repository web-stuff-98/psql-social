package socketmessages

/* This is for Outbound messages, not well organized */

// TYPE: ROOM_MESSAGE_NOTIFY (unused, cba)
type RoomMessageNotify struct {
	RoomID    string `json:"room_id"`
	ChannelID string `json:"channel_id"`
}

// TYPE: ROOM_MESSAGE
type RoomMessage struct {
	ID            string `json:"ID"`
	Content       string `json:"content"`
	CreatedAt     string `json:"created_at"`
	AuthorID      string `json:"author_id"`
	HasAttachment bool   `json:"has_attachment"`
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

// TYPE: BAN
type Ban struct {
	UserID string `json:"user_id"`
	RoomID string `json:"room_id"`
}

// TYPE: DIRECT_MESSAGE
type DirectMessage struct {
	ID            string `json:"ID"`
	Content       string `json:"content"`
	CreatedAt     string `json:"created_at"`
	AuthorID      string `json:"author_id"`
	RecipientID   string `json:"recipient_id"`
	HasAttachment bool   `json:"has_attachment"`
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

// TYPE: BLOCK
type Block struct {
	Blocker string `json:"blocker"`
	Blocked string `json:"blocked"`
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

// TYPE: CALL_USER_RESPONSE
type CallResponse struct {
	Called string `json:"called"`
	Caller string `json:"caller"`
	Accept bool   `json:"accept"`
}

// TYPE: CALL_USER_ACKNOWLEDGE
type CallAcknowledge struct {
	Called string `json:"called"`
	Caller string `json:"caller"`
}

// TYPE: CALL_LEFT
type CallLeft struct{}

// TYPE: CALL_WEBRTC_OFFER_FROM_INITIATOR
type CallWebRTCOfferFromInitiator struct {
	Signal            string `json:"signal"`
	UserMediaStreamID string `json:"um_stream_id"`
	UserMediaVid      bool   `json:"um_vid"`
	DisplayMediaVid   bool   `json:"dm_vid"`
}

// TYPE: CALL_WEBRTC_ANSWER_FROM_RECIPIENT
type CallWebRTCOfferAnswer struct {
	Signal            string `json:"signal"`
	UserMediaStreamID string `json:"um_stream_id"`
	UserMediaVid      bool   `json:"um_vid"`
	DisplayMediaVid   bool   `json:"dm_vid"`
}

// TYPE: CALL_WEBRTC_REQUESTED_REINITIALIZATION
type CallWebRTCRequestedReInitialization struct{}

// TYPE: UPDATE_MEDIA_OPTIONS_OUT
type UpdateMediaOptions struct {
	Uid               string `json:"uid"`
	UserMediaStreamID string `json:"um_stream_id"`
	UserMediaVid      bool   `json:"um_vid"`
	DisplayMediaVid   bool   `json:"dm_vid"`
}

// TYPE: CHANNEL_WEBRTC_JOINED
type ChannelWebRTCUserJoined struct {
	Signal            string `json:"signal"`
	UserMediaStreamID string `json:"um_stream_id"`
	UserMediaVid      bool   `json:"um_vid"`
	DisplayMediaVid   bool   `json:"dm_vid"`
	CallerID          string `json:"caller_id"`
}

// TYPE: CHANNEL_WEBRTC_LEFT
type ChannelWebRTCUserLeft struct {
	Uid string `json:"uid"`
}

// TYPE: CHANNEL_WEBRTC_RETURN_SIGNAL_OUT
type ChannelWebRTCReturnSignal struct {
	Uid               string `json:"uid"`
	Signal            string `json:"signal"`
	UserMediaStreamID string `json:"um_stream_id"`
	UserMediaVid      bool   `json:"um_vid"`
	DisplayMediaVid   bool   `json:"dm_vid"`
}

// TYPE: CHANNEL_WEBRTC_ALL_USERS
type ChannelWebRTCAllUsers struct {
	Users []ChannelWebRTCOutUser `json:"users"`
}
type ChannelWebRTCOutUser struct {
	Uid               string `json:"uid"`
	UserMediaStreamID string `json:"um_stream_id"`
	UserMediaVid      bool   `json:"um_vid"`
	DisplayMediaVid   bool   `json:"dm_vid"`
}

// TYPE: ROOM_CHANNEL_WEBRTC_USER_JOINED/ROOM_CHANNEL_WEBRTC_USER_LEFT
type RoomChannelWebRTCUserJoinedLeft struct {
	ChannelID string `json:"channel_id"`
	Uid       string `json:"uid"`
}

// TYPE: REQUEST_ATTACHMENT
type RequestAttachment struct {
	ID string `json:"ID"`
}

// TYPE: ATTACHMENT_PROGRESS
type AttachmentProgress struct {
	Ratio  float32 `json:"ratio"`
	Failed bool    `json:"failed"`
	MsgID  string  `json:"ID"`
}

// TYPE: ATTACHMENT_METADATA_CREATED
type AttachmentMetadataCreated struct {
	Mime string `json:"mime"`
	Size int    `json:"size"`
	Name string `json:"name"`
	ID   string `json:"ID"`
}
