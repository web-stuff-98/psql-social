import { defineStore } from "pinia";

type NotificationStoreState = {
  directMessages: Record<string, number>;
  roomMessages: Record<string, number>;
};

const useNotificationStore = defineStore("notifications", {
  state: () =>
    ({
      directMessages: {},
      roomMessages: {},
    } as NotificationStoreState),
});

export default useNotificationStore;
