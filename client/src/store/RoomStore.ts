import { defineStore } from "pinia";
import { getRoom } from "../services/room";
import { IRoom } from "../interfaces/GeneralInterfaces";

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
      this.$state.rooms = [
        ...this.$state.rooms.filter((r) => !rooms.find((or) => or.ID === r.ID)),
        ...rooms,
      ];
    },
    async cacheRoom(id: string, force?: boolean) {
      if (this.$state.rooms.findIndex((r) => r.ID === id) !== -1 && !force)
        return;
      try {
        const r = await getRoom(id);
        // spread operator to make sure DOM updates, not sure if necessary
        this.$state.rooms = [
          ...this.$state.rooms.filter((r) => r.ID !== id),
          r,
        ];
      } catch (e) {
        console.warn("Failed to cache room data for", id);
      }
    },
    roomEnteredView(id: string) {
      this.$state.visibleRooms = [...this.$state.visibleRooms, id];
      const i = this.$state.disappearedRooms.findIndex((r) => r.id === id);
      if (i !== -1) this.$state.disappearedRooms.splice(i, 1);
    },
    roomLeftView(id: string) {
      const i = this.$state.visibleRooms.findIndex((r) => r === id);
      if (i !== -1) this.$state.visibleRooms.splice(i, 1);
      if (this.$state.disappearedRooms.findIndex((r) => r.id === id) === -1)
        this.$state.disappearedRooms = [
          ...this.$state.disappearedRooms,
          {
            id,
            disappearedAt: Date.now(),
          },
        ];
    },
  },
});

export default useRoomStore;
