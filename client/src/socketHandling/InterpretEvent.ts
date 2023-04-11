/**
 * Used to get types for data sent through the socket by the server.
 * Matches with socketMessages.go
 */

type ChangeEventData = {
  data: {
    change_type: "UPDATE" | "DELETE" | "INSERT" | "UPDATE_IMAGE";
    entity: "ROOM" | "USER" | "BIO" | "CHANNEL";
    data: object & { ID: string };
  };
};

type MessageData = {
  data: {
    ID: string;
    content: string;
    created_at: string;
    author_id: string;
  };
};

type MessageDeleteData = {
  data: {
    ID: string;
  };
};

type MessageUpdateData = {
  data: {
    ID: string;
    content: string;
  };
};

type RoomMessageData = MessageData;
type RoomMessageDeleteData = MessageDeleteData;
type RoomMessageUpdateData = MessageUpdateData;

type RoomMessageNotifyData = {
  data: {
    room_id: string;
    channel_id: string;
  };
};

type BanData = {
  data: {
    user_id: string;
    room_id: string;
  };
};

type DirectMessageData = MessageData & { data: { recipient_id: string } };
type DirectMessageDeleteData = MessageDeleteData & {
  data: {
    author_id: string;
    recipient_id: string;
  };
};
type DirectMessageUpdateData = MessageUpdateData & {
  data: {
    author_id: string;
    recipient_id: string;
  };
};

type BlockData = {
  data: {
    blocker: string;
    blocked: string;
  };
};

type FriendRequest = {
  data: {
    friender: string;
    friended: string;
    created_at: string;
    accepted?: boolean;
  };
};
type FriendRequestResponse = {
  data: {
    friender: string;
    friended: string;
    accepted: boolean;
  };
};

type Invitation = {
  data: {
    inviter: string;
    invited: string;
    room_id: string;
    created_at: string;
    accepted?: boolean;
  };
};
type InvitationResponse = {
  data: {
    inviter: string;
    invited: string;
    room_id: string;
  };
};

type CallLeft = { data: {} };
type CallWebRTCRequestedReInitialization = { data: {} };
type CallUpdateMediaOptions = {
  data: {
    uid: string;
    um_stream_id: string;
    um_vid: boolean;
    dm_vid: boolean;
  };
};
type CallWebRTCOfferFromInitiator = {
  data: {
    signal: string;
    um_stream_id: string;
    um_vid: boolean;
    dm_vid: boolean;
  };
};
type CallWebRTCOfferAnswer = {
  data: {
    signal: string;
    um_stream_id: string;
    um_vid: boolean;
    dm_vid: boolean;
  };
};
type CallAcknowledge = {
  data: {
    caller: string;
    called: string;
  };
};
type CallResponse = {
  data: {
    caller: string;
    called: string;
    accept: boolean;
  };
};

type ChannelWebRTCAllUsers = {
  data: {
    users: {
      uid: string;
      um_stream_id: string;
      um_vid: boolean;
      dm_vid: boolean;
    }[];
  };
};
type ChannelWebRTCReceivingReturnedSignalData = {
  data: {
    uid: string;
    signal: string;
  };
};
type ChannelWebRTCUserJoined = {
  data: {
    signal: string;
    um_stream_id: string;
    um_vid: boolean;
    dm_vid: boolean;
    caller_id: string;
  };
};
type ChannelWebRTCUserLeft = {
  data: {
    uid: string;
  };
};
type RoomChannelWebRTCUserJoined = {
  data: {
    channel_id: string;
    uid: string;
  };
};
type RoomChannelWebRTCUserLeft = {
  data: {
    channel_id: string;
    uid: string;
  };
};

type RequestAttachment = { data: { ID: string } };

type AttachmentProgress = {
  data: {
    ratio: number;
    failed: boolean;
    ID: string;
  };
};

type AttachmentMetadataCreated = {
  data: {
    mime: string;
    size: number;
    name: string;
    ID: string;
  };
};

export function isChangeEvent(object: any): object is ChangeEventData {
  return object.event_type === "CHANGE";
}

export function isRoomMsg(object: any): object is RoomMessageData {
  return object.event_type === "ROOM_MESSAGE";
}
export function isRoomMsgDelete(object: any): object is RoomMessageDeleteData {
  return object.event_type === "ROOM_MESSAGE_DELETE";
}
export function isRoomMsgUpdate(object: any): object is RoomMessageUpdateData {
  return object.event_type === "ROOM_MESSAGE_UPDATE";
}
export function isBan(object: any): object is BanData {
  return object.event_type === "BAN";
}

export function isDirectMsg(object: any): object is DirectMessageData {
  return object.event_type === "DIRECT_MESSAGE";
}
export function isDirectMsgDelete(
  object: any
): object is DirectMessageDeleteData {
  return object.event_type === "DIRECT_MESSAGE_DELETE";
}
export function isDirectMsgUpdate(
  object: any
): object is DirectMessageUpdateData {
  return object.event_type === "DIRECT_MESSAGE_UPDATE";
}
export function isBlock(object: any): object is BlockData {
  return object.event_type === "BLOCK";
}

export function isFriendRequest(object: any): object is FriendRequest {
  return object.event_type === "FRIEND_REQUEST";
}
export function isFriendRequestResponse(
  object: any
): object is FriendRequestResponse {
  return object.event_type === "FRIEND_REQUEST_RESPONSE";
}

export function isInvitation(object: any): object is Invitation {
  return object.event_type === "INVITATION";
}
export function isInvitationResponse(
  object: any
): object is InvitationResponse {
  return object.event_type === "INVITATION_RESPONSE";
}

export function isCallLeft(object: any): object is CallLeft {
  return object.event_type === "CALL_LEFT";
}
export function isCallRequestedReInitialization(
  object: any
): object is CallWebRTCRequestedReInitialization {
  return object.event_type === "CALL_WEBRTC_REQUESTED_REINITIALIZATION";
}
export function isCallOfferFromInitiator(
  object: any
): object is CallWebRTCOfferFromInitiator {
  return object.event_type === "CALL_WEBRTC_OFFER_FROM_INITIATOR";
}
export function isCallAnswerFromRecipient(
  object: any
): object is CallWebRTCOfferAnswer {
  return object.event_type === "CALL_WEBRTC_ANSWER_FROM_RECIPIENT";
}
export function isCallAcknowledge(object: any): object is CallAcknowledge {
  return object.event_type === "CALL_USER_ACKNOWLEDGE";
}
export function isCallResponse(object: any): object is CallResponse {
  return object.event_type === "CALL_USER_RESPONSE";
}

export function isChannelWebRTCAllUsers(
  object: any
): object is ChannelWebRTCAllUsers {
  return object.event_type === "CHANNEL_WEBRTC_ALL_USERS";
}
export function isChannelWebRTCReceivingReturnedSignal(
  object: any
): object is ChannelWebRTCReceivingReturnedSignalData {
  return object.event_type === "CHANNEL_WEBRTC_RETURN_SIGNAL_OUT";
}
export function isChannelWebRTCUserJoined(
  object: any
): object is ChannelWebRTCUserJoined {
  return object.event_type === "CHANNEL_WEBRTC_JOINED";
}
export function isChannelWebRTCUserLeft(
  object: any
): object is ChannelWebRTCUserLeft {
  return object.event_type === "CHANNEL_WEBRTC_LEFT";
}
export function isRoomChannelWebRTCUserJoined(
  object: any
): object is RoomChannelWebRTCUserJoined {
  return object.event_type === "ROOM_CHANNEL_WEBRTC_USER_JOINED";
}
export function isRoomChannelWebRTCUserLeft(
  object: any
): object is RoomChannelWebRTCUserLeft {
  return object.event_type === "ROOM_CHANNEL_WEBRTC_USER_LEFT";
}

export function isUpdateMediaOptions(
  object: any
): object is CallUpdateMediaOptions {
  return object.event_type === "UPDATE_MEDIA_OPTIONS_OUT";
}

export function isRequestAttachment(object: any): object is RequestAttachment {
  return object.event_type === "REQUEST_ATTACHMENT";
}
export function isAttachmentProgress(
  object: any
): object is AttachmentProgress {
  return object.event_type === "ATTACHMENT_PROGRESS";
}
export function isAttachmentMetadataCreated(
  object: any
): object is AttachmentMetadataCreated {
  return object.event_type === "ATTACHMENT_METADATA_CREATED";
}

export function isRoomMsgNotify(object: any): object is RoomMessageNotifyData {
  return object.event_type === "ROOM_MESSAGE_NOTIFY";
}
