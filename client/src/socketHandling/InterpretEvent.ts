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

export function isUpdateMediaOptions(
  object: any
): object is CallUpdateMediaOptions {
  return object.event_type === "UPDATE_MEDIA_OPTIONS_OUT";
}
