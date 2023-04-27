import { defineStore } from "pinia";
import { getRoom, getRoomImage } from "../services/room";
import { IRoom } from "../interfaces/GeneralInterfaces";
import useUserStore from "./UserStore";
import { ChangeEventData } from "../socketHandling/InterpretEvent";
import useSocketStore from "./SocketStore";
import { StopWatching } from "../socketHandling/OutEvents";

type DisappearedRoom = {
  id: string;
  disappearedAt: number;
};

type RoomStoreState = {
  rooms: IRoom[];
  visibleRooms: string[];
  disappearedRooms: DisappearedRoom[];
};

const useRoomStore = defineStore("rooms", {
  state: () =>
    ({
      rooms: [],
      visibleRooms: [],
      disappearedRooms: [],
    } as RoomStoreState),
  getters: {
    getRoom(state) {
      return (id: string) => state.rooms.find((r) => r.ID === id);
    },
  },
  actions: {
    addRoomsData(rooms: IRoom[]) {
      this.rooms = [
        ...this.rooms.filter((r) => !rooms.find((or) => or.ID === r.ID)),
        ...rooms,
      ];
    },

    cleanupInterval() {
      const roomStore = useRoomStore();
      const socketStore = useSocketStore();

      const disappeared = roomStore.disappearedRooms.map((dr) =>
        Date.now() - dr.disappearedAt > 30000 ? dr.id : ""
      );
      roomStore.rooms = [
        ...roomStore.rooms.filter((r) => !disappeared.includes(r.ID)),
      ];
      roomStore.disappearedRooms = [
        ...roomStore.disappearedRooms.filter(
          (dr) => !disappeared.includes(dr.id)
        ),
      ];
      disappeared.forEach((id) => {
        if (id)
          socketStore.send({
            event_type: "STOP_WATCHING",
            data: { entity: "ROOM", id },
          } as StopWatching);
      });
      socketStore.currentlyWatching = socketStore.currentlyWatching.filter(
        (id) => !disappeared.includes(id)
      );
    },

    async changeEvent({ data: { data, change_type } }: ChangeEventData) {
      if (change_type === "UPDATE") {
        const i = this.rooms.findIndex((r) => r.ID === data.ID);
        if (i !== -1) {
          const newRoom = {
            ...this.rooms[i],
            ...(data as Partial<IRoom>),
          };
          this.rooms = [...this.rooms.filter((r) => r.ID !== data.ID), newRoom];
        } else {
          console.log("Update failed - room not found");
        }
      }
      if (change_type === "INSERT") {
        this.addRoomsData([data as IRoom]);
      }
      if (change_type === "DELETE") {
        const i = this.rooms.findIndex((r) => r.ID === data.ID);
        if (i !== -1) this.rooms.splice(i, 1);
      }
      if (change_type === "UPDATE_IMAGE") {
        const i = this.rooms.findIndex((r) => r.ID === data.ID);
        if (i !== -1) {
          URL.revokeObjectURL(this.rooms[i].img!);
          // wait a bit to make sure the new image is retrieved
          await new Promise<void>((r) => setTimeout(r, 80));
          try {
            const img: BlobPart | undefined = await new Promise((resolve) =>
              getRoomImage(data.ID)
                .catch(() => resolve(undefined))
                .then((pfp) => resolve(pfp))
            );
            if (img) {
              const newRoom = {
                ...this.rooms[i],
                img: URL.createObjectURL(
                  new Blob([img], { type: "image/jpeg" })
                ),
              };
              this.rooms = [
                ...this.rooms.filter((r) => r.ID !== data.ID),
                newRoom,
              ];
            }
          } catch (e) {
            console.warn("Error retrieving image for room:", data.ID);
          }
        } else {
          console.log("Update failed - room not found");
        }
      }
    },

    async cacheRoom(id: string, force?: boolean) {
      if (this.rooms.findIndex((r) => r.ID === id) !== -1 && !force) return;
      try {
        const r = await getRoom(id);
        const img: BlobPart | undefined = await new Promise((resolve) =>
          getRoomImage(id)
            .catch(() => resolve(undefined))
            .then((img) => resolve(img))
        );
        if (img)
          r.img = URL.createObjectURL(new Blob([img], { type: "image/jpeg" }));
        // spread operator to make sure DOM updates, not sure if necessary
        this.rooms = [...this.rooms.filter((r) => r.ID !== id), r];
      } catch (e) {
        console.warn("Failed to cache room data for", id);
      }
    },

    async cacheRoomImage(id: string) {
      try {
        const img: BlobPart | undefined = await new Promise((resolve) =>
          getRoomImage(id)
            .catch(() => resolve(undefined))
            .then((img) => resolve(img))
        );
        if (img) {
          const r = this.rooms.find((r) => r.ID === id);
          if (img && r) {
            r.img = URL.createObjectURL(
              new Blob([img], { type: "image/jpeg" })
            );
            this.rooms = [...this.rooms.filter((r) => r.ID !== id), r];
          }
        }
      } catch (e) {
        console.log(
          "Failed to get image for room",
          id,
          ". Room probably doesn't have an image"
        );
      }
    },

    roomEnteredView(id: string) {
      this.visibleRooms = [...this.visibleRooms, id];
      const i = this.disappearedRooms.findIndex((r) => r.id === id);
      if (i !== -1) this.disappearedRooms.splice(i, 1);
    },
    roomLeftView(id: string) {
      const i = this.visibleRooms.findIndex((r) => r === id);
      if (i !== -1) this.visibleRooms.splice(i, 1);
      if (this.disappearedRooms.findIndex((r) => r.id === id) === -1)
        this.disappearedRooms = [
          ...this.disappearedRooms,
          {
            id,
            disappearedAt: Date.now(),
          },
        ];
    },
  },
});

export default useRoomStore;
