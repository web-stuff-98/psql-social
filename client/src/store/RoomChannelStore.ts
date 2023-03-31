import { defineStore } from "pinia";
import { IRoomChannel } from "../interfaces/GeneralInterfaces";
import { getRoomChannels } from "../services/room";

type RoomChannelStore = {
  channels: IRoomChannel[];
  current: string;
};

const useRoomChannelStore = defineStore("channels", {
  state: () =>
    ({
      channels: [],
      current: "",
      main: "",
    } as RoomChannelStore),
  actions: {
    async getRoomChannels(id: string) {
      const channels = await getRoomChannels(id);
      this.$state.channels = channels || [];
      if (channels) this.$state.current = channels.find((c) => c.main)?.ID!;
    },
  },
});

export default useRoomChannelStore;
