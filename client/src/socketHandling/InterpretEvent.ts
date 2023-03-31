/* Used to get types for data sent through the socket by the server */

type RoomMessageData = {
  data: {
    ID: string;
    content: string;
    created_at: string;
    author_id: string;
  };
};

type RoomMessageDeleteData = {
  data: {
    ID: string;
  };
};

type RoomMessageUpdateData = {
  data: {
    ID: string;
    content: string;
  };
};

export function isRoomMsg(object: any): object is RoomMessageData {
  return object.event_type === "ROOM_MESSAGE";
}

export function isRoomMsgDelete(object: any): object is RoomMessageDeleteData {
  return object.event_type === "ROOM_MESSAGE_DELETE";
}

export function isRoomMsgUpdate(object: any): object is RoomMessageUpdateData {
  return object.event_type === "ROOM_MESSAGE_UPDATE";
}
