import { defineStore } from "pinia";
import { IUser } from "../interfaces/GeneralInterfaces";
import { getUser, getUserPfp } from "../services/user";
import { ChangeEventData } from "../socketHandling/InterpretEvent";
import { StopWatching } from "../socketHandling/OutEvents";
import useSocketStore from "./SocketStore";
import useAuthStore from "./AuthStore";

type DisappearedUser = {
  id: string;
  disappearedAt: number;
};

type thisState = {
  users: IUser[];
  visibleUsers: string[];
  disappearedUsers: DisappearedUser[];
};

const usethis = defineStore("users", {
  state: () =>
    ({
      users: [],
      disappearedUsers: [],
      visibleUsers: [],
    } as thisState),
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
        this.$state.users = [
          ...this.$state.users.filter((u) => u.ID !== id),
          u,
        ];
      } catch (e) {
        console.warn("Failed to cache user data for", id);
      }
    },

    cleanupInterval() {
      const socketStore = useSocketStore();
      const authStore = useAuthStore();

      const disappeared = this.disappearedUsers.map((du) =>
        Date.now() - du.disappearedAt > 30000 && du.id !== authStore.uid
          ? du.id
          : ""
      );
      this.users = [...this.users.filter((u) => !disappeared.includes(u.ID))];
      this.disappearedUsers = [
        ...this.disappearedUsers.filter((du) => !disappeared.includes(du.id)),
      ];
      disappeared.forEach((id) => {
        if (id)
          socketStore.send({
            event_type: "STOP_WATCHING",
            data: { entity: "USER", id },
          } as StopWatching);
      });
      socketStore.currentlyWatching = socketStore.currentlyWatching.filter(
        (id) => !disappeared.includes(id)
      );
    },

    async changeEvent({ data: { data, change_type } }: ChangeEventData) {
      if (change_type === "UPDATE_IMAGE") {
        const i = this.users.findIndex((u) => u.ID === data.ID);
        if (i !== -1) {
          URL.revokeObjectURL(this.users[i].pfp!);
          // wait a bit to make sure the new image is retrieved
          await new Promise<void>((r) => setTimeout(r, 80));
          try {
            const pfp: BlobPart | undefined = await new Promise((resolve) =>
              getUserPfp(data.ID)
                .catch(() => resolve(undefined))
                .then((pfp) => resolve(pfp))
            );
            if (pfp) {
              const newUser = {
                ...this.users[i],
                pfp: URL.createObjectURL(
                  new Blob([pfp], { type: "image/jpeg" })
                ),
              };
              this.users = [
                ...this.users.filter((u) => u.ID !== data.ID),
                newUser,
              ];
            }
          } catch (e) {
            console.warn("Error retrieving image for user:", data.ID);
          }
        } else {
          console.log("Update failed - user not found");
        }
      }
      if (change_type === "DELETE") {
        const i = this.users.findIndex((u) => u.ID === data.ID);
        if (i !== -1) this.users.splice(i, 1);
      }
      if (change_type === "UPDATE") {
        const i = this.users.findIndex((u) => u.ID === data.ID);
        if (i !== -1) {
          const newUser = {
            ...this.users[i],
            ...(data as Partial<IUser>),
          };
          this.users = [...this.users.filter((u) => u.ID !== data.ID), newUser];
        } else {
          console.log("Update failed - user not found");
        }
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

export default usethis;
