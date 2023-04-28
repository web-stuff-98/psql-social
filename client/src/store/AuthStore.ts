import { defineStore } from "pinia";
import { makeRequest } from "../services/makeRequest";
import useNotificationStore from "./NotificationStore";

type AuthStoreState = {
  uid?: string;
};

const useAuthStore = defineStore("auth", {
  state: () =>
    ({
      uid: undefined,
    } as AuthStoreState),
  actions: {
    async login(username: string, password: string) {
      const notificationsStore = useNotificationStore();

      const uid: string = await makeRequest("/api/acc/login", {
        method: "POST",
        data: { username, password },
        responseType: "text",
      });
      this.uid = uid;

      await notificationsStore.retrieveNotifications();
    },
    async register(username: string, password: string, policy: boolean) {
      const notificationsStore = useNotificationStore();

      const uid: string = await makeRequest("/api/acc/register", {
        method: "POST",
        data: { username, password, policy },
        responseType: "text",
      });
      this.uid = uid;

      notificationsStore.clearAllNotifications();
    },
    async logout() {
      const notificationsStore = useNotificationStore();

      await makeRequest("/api/acc/logout", {
        method: "POST",
      });
      this.uid = undefined;

      notificationsStore.clearAllNotifications();
    },
  },
});

export default useAuthStore;
