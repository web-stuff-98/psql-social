/* All types for socket messages that go out */

/* Watch messages (subscribe to updates for entity) */
type Watchable = "USER" | "ROOM" | "BIO";
export type StartWatching = {
  event_type: "START_WATCHING";
  data: { entity: Watchable; id: string };
};
export type StopWatching = {
  event_type: "STOP_WATCHING";
  data: { entity: Watchable; id: string };
};

/* Direct messages / friend requests / invitations */
export type DirectMessage = {
  event_type: "DIRECT_MESSAGE";
  data: {
    content: string;
    uid: string;
  };
};
export type DirectMessageUpdate = {
  event_type: "DIRECT_MESSAGE_UPDATE";
  data: {
    msg_id: string;
    content: string;
  };
};
export type DirectMessageDelete = {
  event_type: "DIRECT_MESSAGE_DELETE";
  data: { msg_id: string };
};
export type FriendRequest = {
  event_type: "FRIEND_REQUEST";
  data: { uid: string };
};
export type FriendRequestResponse = {
  event_type: "FRIEND_REQUEST_RESPONSE";
  data: { accepted: boolean; friender: string };
};
export type Invitation = {
  event_type: "INVITATION";
  data: { inviter: string; invited: string; room_id: string };
};
export type InvitationResponse = {
  event_type: "INVITATION_RESPONSE";
  data: { accepted: boolean; room_id: string; inviter: string };
};

/* Room events */
export type RoomMessage = {
  event_type: "ROOM_MESSAGE";
  data: {
    content: string;
    channel_id: string;
  };
};
export type RoomMessageUpdate = {
  event_type: "ROOM_MESSAGE_UPDATE";
  data: {
    msg_id: string;
    content: string;
  };
};
export type RoomMessageDelete = {
  event_type: "ROOM_MESSAGE_DELETE";
  data: {
    msg_id: string;
  };
};
export type JoinRoom = {
  event_type: "JOIN_ROOM";
  data: { room_id: string };
};
export type LeaveRoom = {
  event_type: "LEAVE_ROOM";
  data: { room_id: string };
};

/* WebRTC events */
export type ChannelWebRTCUpdateMediaOptions = {
  event_type: "CHANNEL_WEBRTC_UPDATE_MEDIA_OPTIONS";
  data: {
    um_vid: boolean;
    dm_vid: boolean;
    um_stream_id: string;
    channel_id: string;
  };
};
export type CallWebRTCUpdateMediaOptions = {
  event_type: "CALL_UPDATE_MEDIA_OPTIONS";
  data: {
    um_vid: boolean;
    dm_vid: boolean;
    um_stream_id: string;
  };
};
