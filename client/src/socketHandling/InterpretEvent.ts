/**
 * Used to get types for data sent through the socket by the server.
 * Matches with socketMessages.go
 */

type ChangeEventData = {
  data: {
    change_type: "UPDATE" | "DELETE" | "INSERT" | "UPDATE_IMAGE";
    entity: "ROOM" | "USER" | "BIO";
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
  user_id: string;
  room_id: string;
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
