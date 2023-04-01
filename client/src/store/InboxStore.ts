import { defineStore } from "pinia";
import {
  IDirectMessage,
  IFriendRequest,
  IInvitation,
} from "../interfaces/GeneralInterfaces";

type InboxStoreState = {
  convs: Record<
    string,
    Array<IDirectMessage | IInvitation | IFriendRequest>
  >;
};

const useInboxStore = defineStore("inbox", {
  state: () =>
    ({
      convs: {},
    } as InboxStoreState),
});

export default useInboxStore;
