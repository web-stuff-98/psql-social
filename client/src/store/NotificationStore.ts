import { defineStore } from "pinia";
import { getNotifications } from "../services/account";

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
    getUserNotifications(state) {
      return (uid: string) =>
        state.directMessages[uid] ? state.directMessages[uid] : 0;
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
          if (!this.roomMessages[n.room_id]) {
            this.roomMessages[n.room_id] = {};
            this.roomMessages[n.room_id][n.channel_id] = 1;
          } else
            this.roomMessages[n.room_id][n.channel_id] =
              this.roomMessages[n.room_id][n.channel_id] === undefined
                ? this.roomMessages[n.room_id][n.channel_id] + 1
                : 1;
        });
      else this.roomMessages = {};
    },
    clearChannelNotifications(roomId: string, id: string) {
      this.roomMessages[roomId][id] = 0;
    },
    clearUserNotifications(uid: string) {
      this.directMessages[uid] = 0;
    },
  },
});

export default useNotificationStore;
