import { defineStore } from "pinia";
import { IUser } from "../interfaces/GeneralInterfaces";
import { getUser, getUserPfp } from "../services/user";

type DisappearedUser = {
  id: string;
  disappearedAt: number;
};

type UserStoreState = {
  users: IUser[];
  visibleUsers: string[];
  disappearedUsers: DisappearedUser[];
};

const useUserStore = defineStore("users", {
  state: () =>
    ({
      users: [],
      disappearedUsers: [],
      visibleUsers: [],
    } as UserStoreState),
  getters: {
    getUser(state) {
      return (id: string) => state.users.find((u) => u.ID === id);
    },
  },
  actions: {
    async cacheUser(id: string, force?: boolean) {
      if (this.$state.users.findIndex((u) => u.ID === id) !== -1 && !force)
        return;
      try {
        const u = await getUser(id);
        const pfp: BlobPart | undefined = await new Promise((resolve) =>
          getUserPfp(id)
            .catch(() => resolve(undefined))
            .then((pfp) => resolve(pfp))
        );
        if (pfp)
          u.pfp = URL.createObjectURL(new Blob([pfp], { type: "image/jpeg" }));
        // spread operator to make sure DOM updates, not sure if necessary
        this.$state.users = [
          ...this.$state.users.filter((u) => u.ID !== id),
          u,
        ];
      } catch (e) {
        console.warn("Failed to cache user data for", id);
      }
    },
    userEnteredView(id: string) {
      this.$state.visibleUsers = [...this.$state.visibleUsers, id];
      const i = this.$state.disappearedUsers.findIndex((u) => u.id === id);
      if (i !== -1) this.$state.disappearedUsers.splice(i, 1);
    },
    userLeftView(id: string) {
      const i = this.$state.visibleUsers.findIndex((u) => u === id);
      if (i !== -1) this.$state.visibleUsers.splice(i, 1);
      if (this.$state.disappearedUsers.findIndex((u) => u.id === id) === -1)
        this.$state.disappearedUsers = [
          ...this.$state.disappearedUsers,
          {
            id,
            disappearedAt: Date.now(),
          },
        ];
    },
  },
});

export default useUserStore;
