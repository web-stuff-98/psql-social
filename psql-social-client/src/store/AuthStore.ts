import { defineStore } from "pinia";
import { IUser } from "../interfaces/GeneralInterfaces";
import { makeRequest } from "../services/makeRequest";

type AuthStoreState = {
  user?: IUser;
  token: string;
};

const useAuthStore = defineStore("auth", {
  state: () =>
    ({
      user: undefined,
      token: "",
    } as AuthStoreState),
  getters: {
    getCurrentUser(state) {
      return state.user;
    },
    getToken(state) {
      return state.token;
    },
  },
  actions: {
    async login(username: string, password: string) {
      const user: {
        ID: string;
        username: string;
        token: string;
        role: "ADMIN" | "USER";
      } = await makeRequest("/api/acc/login", {
        method: "POST",
        data: { username, password },
      });
      this.$state.token = user.token;
      this.$state.user = {
        ID: user.ID,
        username: user.username,
        role: user.role,
      };
    },
    async register(username: string, password: string) {
      const user: {
        ID: string;
        username: string;
        token: string;
        role: "ADMIN" | "USER";
      } = await makeRequest("/api/acc/register", {
        method: "POST",
        data: { username, password },
      });
      this.$state.token = user.token;
      this.$state.user = {
        ID: user.ID,
        username: user.username,
        role: user.role,
      };
    },
    async logout() {
      await makeRequest("/api/acc/logout", {
        method: "POST",
      });
    },
  },
});

export default useAuthStore;
