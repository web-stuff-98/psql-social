import { defineStore } from "pinia";
import {
  IDirectMessage,
  IFriendRequest,
  IInvitation,
} from "../interfaces/GeneralInterfaces";
import {
  isBlock,
  isDirectMsg,
  isDirectMsgDelete,
  isDirectMsgUpdate,
  isFriendRequest,
  isFriendRequestResponse,
  isInvitation,
  isInvitationResponse,
} from "../socketHandling/InterpretEvent";
import useAuthStore from "./AuthStore";
import useUserStore from "./UserStore";

type InboxStoreState = {
  convs: Record<string, Array<IDirectMessage | IInvitation | IFriendRequest>>;
};

const useInboxStore = defineStore("inbox", {
  state: () =>
    ({
      convs: {},
    } as InboxStoreState),

  actions: {
    watchInbox(e: MessageEvent) {
      const authStore = useAuthStore();
      const userStore = useUserStore();

      const msg = JSON.parse(e.data);
      if (!msg) return;
      if (isDirectMsg(msg)) {
        const otherUser =
          msg.data.author_id === authStore.uid
            ? msg.data.recipient_id
            : msg.data.author_id;
        this.convs[otherUser] = [
          ...(this.convs[otherUser] || []),
          msg.data as IDirectMessage,
        ];
        userStore.cacheUser(msg.data.author_id);
      }
      if (isDirectMsgUpdate(msg)) {
        const otherUser =
          msg.data.author_id === authStore.uid
            ? msg.data.recipient_id
            : msg.data.author_id;
        let newConv = this.convs[otherUser] || [];
        const i = newConv.findIndex((item) => {
          // if it has an ID then its a direct message, not an invite or friend request
          if ((item as any)["ID"] !== undefined)
            return (item as any)["ID"] === msg.data.ID;
        });
        //@ts-ignore
        newConv[i]["content"] = msg.data.content;
        this.convs[otherUser] = [...newConv];
      }
      if (isDirectMsgDelete(msg)) {
        const otherUser =
          msg.data.author_id === authStore.uid
            ? msg.data.recipient_id
            : msg.data.author_id;
        if (this.convs[otherUser]) {
          const i = this.convs[otherUser].findIndex((item) => {
            // if it has an ID then its a direct message, not an invite or friend request
            if ((item as any)["ID"] !== undefined)
              return (item as any)["ID"] === msg.data.ID;
          });
          this.convs[otherUser].splice(i, 1);
        }
      }

      if (isFriendRequest(msg)) {
        const otherUser =
          msg.data.friended === authStore.uid
            ? msg.data.friender
            : msg.data.friended;
        this.convs[otherUser] = [
          ...(this.convs[otherUser] || []),
          msg.data as IFriendRequest,
        ];
        userStore.cacheUser(otherUser);
      }
      if (isFriendRequestResponse(msg)) {
        const otherUser =
          msg.data.friended === authStore.uid
            ? msg.data.friender
            : msg.data.friended;
        let newConv = this.convs[otherUser] || [];
        const i = newConv.findIndex((item) => {
          // if it has a "friender" then its a friend request
          if ((item as any)["friender"] !== undefined)
            return (
              (item as any)["friender"] === msg.data.friender &&
              (item as any)["friended"] === msg.data.friended
            );
        });
        //@ts-ignore
        newConv[i]["accepted"] = msg.data.accepted;
        this.convs[otherUser] = [...newConv];
        userStore.cacheUser(otherUser);
      }

      if (isInvitation(msg)) {
        const otherUser =
          msg.data.inviter === authStore.uid
            ? msg.data.invited
            : msg.data.inviter;
        this.convs[otherUser] = [
          ...(this.convs[otherUser] || []),
          msg.data as IInvitation,
        ];
        userStore.cacheUser(otherUser);
      }
      if (isInvitationResponse(msg)) {
        const otherUser =
          msg.data.inviter === authStore.uid
            ? msg.data.invited
            : msg.data.inviter;
        let newConv = this.convs[otherUser] || [];
        const i = newConv.findIndex((item) => {
          // if it has an "inviter" then its an invitation
          if ((item as any)["inviter"] !== undefined)
            return (
              (item as any)["inviter"] === msg.data.inviter &&
              (item as any)["invited"] === msg.data.invited &&
              (item as any)["room_id"] === msg.data.room_id
            );
        });
        //@ts-ignore
        newConv[i]["accepted"] = msg.data.accepted;
        this.convs[otherUser] = [...newConv];
        userStore.cacheUser(otherUser);
      }

      if (isBlock(msg)) {
        const otherUser =
          msg.data.blocker === authStore.uid
            ? msg.data.blocked
            : msg.data.blocker;
        delete this.convs[otherUser];
      }
    },
  },
});

export default useInboxStore;
