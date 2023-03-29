/* All types for socket messages that go out */

/* Watch messages (subscribe to updates for entity) */
type Watchable = "USER" | "ROOM";
export type StartWatching = {
  event_type: "START_WATCHING";
  data: { entity: Watchable };
};
export type StopWatching = {
  event_type: "STOP_WATCHING";
  data: { entity: Watchable };
};

/* Direct messages */
export type DirectMessage = {
  event_type: "DIRECT_MESSAGE";
  data: {
    content: string;
  };
};
export type DirectMessageUpdate = {
  event_type: "DIRECT_MESSAGE_UPDATE";
  data: {
    content: string;
  };
};
export type DirectMessageDelete = {
  event_type: "DIRECT_MESSAGE_DELETE";
  data: { id: string };
};

/* Room messages */
export type RoomMessage = {
  event_type: "ROOM_MESSAGE";
  data: {
    content: string;
    channel_id: string;
  };
};
export type RoomMessageUpdate = {
  event_type: "ROOM_MESSAGE";
  data: {
    content: string;
    channel_id: string;
  };
};
export type RoomMessageDelete = {
  event_type: "ROOM_MESSAGE_DELETE";
  data: {
    id: string;
    channel_id: string;
  };
};
