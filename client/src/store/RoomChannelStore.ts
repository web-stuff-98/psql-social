import { defineStore } from "pinia";
import { IRoomChannel } from "../interfaces/GeneralInterfaces";
import { getRoomChannels } from "../services/room";

type RoomChannelStore = {
  channels: IRoomChannel[];
  current: string;
  uidsInCurrentWebRTCChat: string[];
};

const useRoomChannelStore = defineStore("channels", {
  state: () =>
    ({
      channels: [],
      current: "",
      main: "",
      uidsInCurrentWebRTCChat: [],
    } as RoomChannelStore),
  actions: {
    // this is called "get" but doesn't return anything. It modifies the store state. So it's not in getters.
    async getRoomChannels(id: string): Promise<string> {
      const channels = await getRoomChannels(id);
      this.$state.channels = channels || [];
      if (channels) {
        const main = channels.find((c) => c.main)?.ID!;
        this.$state.current = main;
        return main;
      }
      return "";
    },
  },
});

export default useRoomChannelStore;
