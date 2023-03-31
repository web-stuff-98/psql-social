import { defineStore } from "pinia";
import { IDirectMessage } from "../interfaces/GeneralInterfaces";

type InboxStoreState = {
  messages: IDirectMessage[];
};

const useInboxStore = defineStore("inbox", {
  state: () =>
    ({
      messages: [],
    } as InboxStoreState),
});

export default useInboxStore