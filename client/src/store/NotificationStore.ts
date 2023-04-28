import { defineStore } from "pinia";
import { getNotifications } from "../services/account";
import {
  isDirectMessageNotifyData,
  isDirectMessageNotifyDeleteData,
  isRoomMsgNotify,
  isRoomMsgNotifyDelete,
} from "../socketHandling/InterpretEvent";
import notify from "../assets/notify.wav";

type NotificationStoreState = {
  // sender id -> num notifications
  directMessages: Record<string, number>;
  // room id -> channel id -> num notifications
  roomMessages: Record<string, Record<string, number>>;
};

const useNotificationStore = defineStore("notifications", {
  state: () =>
    ({
      directMessages: {},
      roomMessages: {},
    } as NotificationStoreState),

  getters: {
    getChannelNotifications(state) {
      return (roomId: string, id: string) =>
        state.roomMessages[roomId]
          ? state.roomMessages[roomId][id]
            ? state.roomMessages[roomId][id]
            : 0
          : 0;
    },
    getRoomNotifications(state) {
      return (id: string) =>
        state.roomMessages[id]
          ? Object.values(state.roomMessages[id]).reduce(
              (acc, val) => acc + val,
              0
            )
          : 0;
    },
    getAllRoomNotifications(state) {
      return () =>
        Object.values(state.roomMessages).reduce(
          (acc, val) =>
            acc + Object.values(val).reduce((acc, val) => acc + val, 0),
          0
        );
    },
    getUserNotifications(state) {
      return (uid: string) =>
        state.directMessages[uid] ? state.directMessages[uid] : 0;
    },
    getAllUserNotifications(state) {
      return () =>
        Object.values(state.directMessages).reduce((acc, val) => acc + val, 0);
    },
  },

  actions: {
    async retrieveNotifications() {
      const ns = await getNotifications();
      if (ns.dm_ns !== undefined)
        ns.dm_ns.forEach(
          (n) =>
            (this.directMessages[n.sender_id] =
              this.directMessages[n.sender_id] === undefined
                ? 1
                : this.directMessages[n.sender_id] + 1)
        );
      else this.directMessages = {};
      if (ns.rm_ns)
        ns.rm_ns.forEach((n) => {
          if (!this.roomMessages[n.room_id])
            this.roomMessages[n.room_id][n.channel_id] = 1;
          else
            this.roomMessages[n.room_id][n.channel_id] =
              this.roomMessages[n.room_id][n.channel_id] === undefined
                ? this.roomMessages[n.room_id][n.channel_id] + 1
                : 1;
        });
      else this.roomMessages = {};
    },
    clearChannelNotifications(roomId: string, id: string) {
      if (this.roomMessages[roomId])
        if (this.roomMessages[roomId][id]) this.roomMessages[roomId][id] = 0;
    },
    clearUserNotifications(uid: string) {
      this.directMessages[uid] = 0;
    },
    clearAllNotifications() {
      this.directMessages = {};
      this.roomMessages = {};
    },
    directMessageNotify(uid: string) {
      const audio = new Audio(notify);
      audio.play();
      this.directMessages[uid] = this.directMessages[uid]
        ? this.directMessages[uid] + 1
        : 1;
    },
    directMessageNotifyDelete(uid: string) {
      this.directMessages[uid] = this.directMessages[uid]
        ? Math.max(0, this.directMessages[uid] - 1)
        : 0;
    },
    roomMessageNotify(roomId: string, channelId: string) {
      const audio = new Audio(notify);
      audio.play();
      if (this.roomMessages[roomId]) {
        this.roomMessages[roomId][channelId] = this.roomMessages[roomId][
          channelId
        ]
          ? this.roomMessages[roomId][channelId] + 1
          : 1;
      } else {
        this.roomMessages[roomId] = {};
        this.roomMessages[roomId][channelId] = 1;
      }
    },
    roomMessageNotifyDelete(roomId: string, channelId: string) {
      if (this.roomMessages[roomId]) {
        this.roomMessages[roomId][channelId] = this.roomMessages[roomId][
          channelId
        ]
          ? this.roomMessages[roomId][channelId] - 1
          : 0;
      }
    },

    watchNotifications(e: MessageEvent) {
      const msg = JSON.parse(e.data);
      if (!msg) return;
      if (isDirectMessageNotifyData(msg))
        this.directMessageNotify(msg.data.uid);
      if (isDirectMessageNotifyDeleteData(msg))
        this.directMessageNotifyDelete(msg.data.uid);
      if (isRoomMsgNotify(msg))
        this.roomMessageNotify(msg.data.room_id, msg.data.channel_id);
      if (isRoomMsgNotifyDelete(msg))
        this.roomMessageNotifyDelete(msg.data.room_id, msg.data.channel_id);
    },
  },
});

export default useNotificationStore;
