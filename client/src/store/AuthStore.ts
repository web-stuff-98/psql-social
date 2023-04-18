import { defineStore } from "pinia";
import { makeRequest } from "../services/makeRequest";

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
      const uid: string = await makeRequest("/api/acc/login", {
        method: "POST",
        data: { username, password },
        responseType: "text",
      });
      this.$state.uid = uid;
    },
    async register(username: string, password: string, policy: boolean) {
      const uid: string = await makeRequest("/api/acc/register", {
        method: "POST",
        data: { username, password, policy },
        responseType: "text",
      });
      this.$state.uid = uid;
    },
    async logout() {
      await makeRequest("/api/acc/logout", {
        method: "POST",
      });
      this.$state.uid = undefined;
    },
  },
});

export default useAuthStore;
