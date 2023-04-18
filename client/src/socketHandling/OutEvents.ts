/**
 * Types for socket messages that go out
 */

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

/* Direct messages / friend requests / invitations / call events */
export type DirectMessage = {
  event_type: "DIRECT_MESSAGE";
  data: {
    content: string;
    uid: string;
    has_attachment?: boolean;
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
  data: { uid: string; room_id: string };
};
export type InvitationResponse = {
  event_type: "INVITATION_RESPONSE";
  data: { accepted: boolean; room_id: string; inviter: string };
};
export type Block = {
  event_type: "BLOCK";
  data: { uid: string };
};
export type UnBlock = {
  event_type: "UNBLOCK";
  data: { uid: string };
};
export type CallUser = {
  event_type: "CALL_USER";
  data: {
    uid: string;
  };
};
export type CallResponse = {
  event_type: "CALL_USER_RESPONSE";
  data: {
    caller: string;
    called: string;
    accept: boolean;
  };
};

/* Room events */
export type RoomMessage = {
  event_type: "ROOM_MESSAGE";
  data: {
    content: string;
    channel_id: string;
    has_attachment?: boolean;
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
export type JoinChannel = {
  event_type: "JOIN_CHANNEL";
  data: { channel_id: string };
};
export type LeaveChannel = {
  event_type: "LEAVE_CHANNEL";
  data: { channel_id: string };
};
export type Ban = {
  event_type: "BAN";
  data: { uid: string; room_id: string };
};
export type UnBan = {
  event_type: "UNBAN";
  data: { uid: string; room_id: string };
};

/* Call & WebRTC events */
export type CallWebRTCRequestReInitialization = {
  event_type: "CALL_WEBRTC_RECIPIENT_REQUEST_REINITIALIZATION";
};
export type CallWebRTCOffer = {
  event_type: "CALL_WEBRTC_OFFER";
  data: {
    signal: string;
    um_stream_id: string;
    um_vid: boolean;
    dm_vid: boolean;
  };
};
export type CallWebRTCAnswer = {
  event_type: "CALL_WEBRTC_ANSWER";
  data: {
    signal: string;
    um_stream_id: string;
    um_vid: boolean;
    dm_vid: boolean;
  };
};
export type CallLeave = {
  event_type: string;
};
export type CallWebRTCUpdateMediaOptions = {
  event_type: "CALL_UPDATE_MEDIA_OPTIONS";
  data: {
    um_vid: boolean;
    dm_vid: boolean;
    um_stream_id: string;
  };
};
export type ChannelWebRTCUpdateMediaOptions = {
  event_type: "CHANNEL_WEBRTC_UPDATE_MEDIA_OPTIONS";
  data: {
    um_vid: boolean;
    dm_vid: boolean;
    um_stream_id: string;
    channel_id: string;
  };
};
export type ChannelWebRTCLeave = {
  event_type: "CHANNEL_WEBRTC_LEAVE";
  data: { channel_id: string };
};
export type ChannelWebRTCSendingSignal = {
  event_type: "CHANNEL_WEBRTC_SENDING_SIGNAL";
  data: {
    signal: string;
    to_uid: string;
    um_stream_id: string;
    um_vid: boolean;
    dm_vid: boolean;
  };
};
export type ChannelWebRTCReturningSignal = {
  event_type: "CHANNEL_WEBRTC_RETURNING_SIGNAL";
  data: {
    signal: string;
    caller_id: string;
    um_stream_id: string;
    um_vid: boolean;
    dm_vid: boolean;
  };
};
export type ChannelWebRTCJoin = {
  event_type: "CHANNEL_WEBRTC_JOIN";
  data: {
    channel_id: string;
    um_stream_id: string;
    um_vid: boolean;
    dm_vid: boolean;
  };
};
