/* Used to get types for data sent through the socket by the server */

type RoomMessageData = {
  data: {
    ID: string;
    content: string;
    created_at: string;
    author_id: string;
  };
};

export function isRoomMsg(object: any): object is RoomMessageData {
  return object.event_type === "ROOM_MESSAGE";
}
