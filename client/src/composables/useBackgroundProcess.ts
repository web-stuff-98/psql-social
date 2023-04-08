import { onBeforeUnmount, onMounted, Ref, ref, watchEffect } from "vue";
import {
  IDirectMessage,
  IFriendRequest,
  IInvitation,
  IResMsg,
  IRoom,
  IUser,
} from "../interfaces/GeneralInterfaces";
import { makeRequest } from "../services/makeRequest";
import { StartWatching, StopWatching } from "../socketHandling/OutEvents";
import useAuthStore from "../store/AuthStore";
import useSocketStore from "../store/SocketStore";
import useUserStore from "../store/UserStore";
import useRoomStore from "../store/RoomStore";
import useInboxStore from "../store/InboxStore";
import useAttachmentStore from "../store/AttachmentStore";
import {
  isAttachmentMetadataCreated,
  isAttachmentProgress,
  isBlock,
  isCallAcknowledge,
  isCallResponse,
  isChangeEvent,
  isDirectMsg,
  isDirectMsgDelete,
  isDirectMsgUpdate,
  isFriendRequest,
  isFriendRequestResponse,
  isInvitation,
  isInvitationResponse,
} from "../socketHandling/InterpretEvent";
import { getUserPfp } from "../services/user";
import { pendingCallsStore } from "../store/CallsStore";
import { useRouter } from "vue-router";
import { getRoomImage } from "../services/room";

/**
 * This composable is for intervals that run in the background
 * and socket event listeners that aren't tied to components,
 * et cet
 */

export default function useBackgroundProcess({
  resMsg,
}: {
  resMsg: Ref<IResMsg | undefined>;
}) {
  const refreshTokenInterval = ref<NodeJS.Timer>();
  const pingSocketInterval = ref<NodeJS.Timer>();
  const clearUserCacheInterval = ref<NodeJS.Timer>();
  const clearRoomCacheInterval = ref<NodeJS.Timer>();
  const clearAttachmentCacheInterval = ref<NodeJS.Timer>();

  const authStore = useAuthStore();
  const socketStore = useSocketStore();
  const userStore = useUserStore();
  const roomStore = useRoomStore();
  const inboxStore = useInboxStore();
  const attachmentStore = useAttachmentStore();

  const router = useRouter();

  const currentlyWatching = ref<string[]>([]);

  async function watchForChangeEvents(e: MessageEvent) {
    const msg = JSON.parse(e.data);
    if (!msg) return;
    if (isChangeEvent(msg)) {
      // Watch for room update events
      if (msg.data.entity === "ROOM") {
        if (msg.data.change_type === "UPDATE") {
          const i = roomStore.rooms.findIndex((r) => r.ID === msg.data.data.ID);
          if (i !== -1) {
            const newRoom = {
              ...roomStore.rooms[i],
              ...(msg.data.data as Partial<IRoom>),
            };
            roomStore.rooms = [
              ...roomStore.rooms.filter((r) => r.ID !== msg.data.data.ID),
              newRoom,
            ];
          } else {
            console.log("Update failed - room not found");
          }
        }
        if (msg.data.change_type === "INSERT") {
          roomStore.addRoomsData([msg.data.data as IRoom]);
        }
        if (msg.data.change_type === "DELETE") {
          const i = roomStore.rooms.findIndex((r) => r.ID === msg.data.data.ID);
          if (i !== -1) roomStore.rooms.splice(i, 1);
        }
        if (msg.data.change_type === "UPDATE_IMAGE") {
          const i = roomStore.rooms.findIndex((r) => r.ID === msg.data.data.ID);
          if (i !== -1) {
            URL.revokeObjectURL(roomStore.rooms[i].img!);
            // wait a bit to make sure the new image is retrieved
            await new Promise<void>((r) => setTimeout(r, 80));
            try {
              const img: BlobPart | undefined = await new Promise((resolve) =>
                getRoomImage(msg.data.data.ID)
                  .catch(() => resolve(undefined))
                  .then((pfp) => resolve(pfp))
              );
              if (img) {
                const newRoom = {
                  ...roomStore.rooms[i],
                  img: URL.createObjectURL(
                    new Blob([img], { type: "image/jpeg" })
                  ),
                };
                roomStore.rooms = [
                  ...roomStore.rooms.filter((r) => r.ID !== msg.data.data.ID),
                  newRoom,
                ];
              }
            } catch (e) {
              console.warn(
                "Error retrieving image for room:",
                msg.data.data.ID
              );
            }
          } else {
            console.log("Update failed - room not found");
          }
        }
      }
      // Watch for user update events
      if (msg.data.entity === "USER") {
        if (msg.data.change_type === "UPDATE_IMAGE") {
          const i = userStore.users.findIndex((u) => u.ID === msg.data.data.ID);
          if (i !== -1) {
            URL.revokeObjectURL(userStore.users[i].pfp!);
            // wait a bit to make sure the new image is retrieved
            await new Promise<void>((r) => setTimeout(r, 80));
            try {
              const pfp: BlobPart | undefined = await new Promise((resolve) =>
                getUserPfp(msg.data.data.ID)
                  .catch(() => resolve(undefined))
                  .then((pfp) => resolve(pfp))
              );
              if (pfp) {
                const newUser = {
                  ...userStore.users[i],
                  pfp: URL.createObjectURL(
                    new Blob([pfp], { type: "image/jpeg" })
                  ),
                };
                userStore.users = [
                  ...userStore.users.filter((u) => u.ID !== msg.data.data.ID),
                  newUser,
                ];
              }
            } catch (e) {
              console.warn(
                "Error retrieving image for user:",
                msg.data.data.ID
              );
            }
          } else {
            console.log("Update failed - user not found");
          }
        }
        if (msg.data.change_type === "DELETE") {
          const i = userStore.users.findIndex((u) => u.ID === msg.data.data.ID);
          if (i !== -1) userStore.users.splice(i, 1);
        }
        if (msg.data.change_type === "UPDATE") {
          const i = userStore.users.findIndex((u) => u.ID === msg.data.data.ID);
          if (i !== -1) {
            const newUser = {
              ...userStore.users[i],
              ...(msg.data.data as Partial<IUser>),
            };
            userStore.users = [
              ...userStore.users.filter((u) => u.ID !== msg.data.data.ID),
              newUser,
            ];
          } else {
            console.log("Update failed - user not found");
          }
        }
      }
    }
  }

  function watchInbox(e: MessageEvent) {
    const msg = JSON.parse(e.data);
    if (!msg) return;
    if (isDirectMsg(msg)) {
      const otherUser =
        msg.data.author_id === authStore.uid
          ? msg.data.recipient_id
          : msg.data.author_id;
      inboxStore.convs[otherUser] = [
        ...(inboxStore.convs[otherUser] || []),
        msg.data as IDirectMessage,
      ];
      userStore.cacheUser(msg.data.author_id);
    }
    if (isDirectMsgUpdate(msg)) {
      const otherUser =
        msg.data.author_id === authStore.uid
          ? msg.data.recipient_id
          : msg.data.author_id;
      let newConv = inboxStore.convs[otherUser] || [];
      const i = newConv.findIndex((item) => {
        // if it has an ID then its a direct message, not an invite or friend request
        if ((item as any)["ID"] !== undefined)
          return (item as any)["ID"] === msg.data.ID;
      });
      //@ts-ignore
      newConv[i]["content"] = msg.data.content;
      inboxStore.convs[otherUser] = [...newConv];
    }
    if (isDirectMsgDelete(msg)) {
      const otherUser =
        msg.data.author_id === authStore.uid
          ? msg.data.recipient_id
          : msg.data.author_id;
      if (inboxStore.convs[otherUser]) {
        const i = inboxStore.convs[otherUser].findIndex((item) => {
          // if it has an ID then its a direct message, not an invite or friend request
          if ((item as any)["ID"] !== undefined)
            return (item as any)["ID"] === msg.data.ID;
        });
        inboxStore.convs[otherUser].splice(i, 1);
      }
    }

    if (isFriendRequest(msg)) {
      const otherUser =
        msg.data.friended === authStore.uid
          ? msg.data.friender
          : msg.data.friended;
      inboxStore.convs[otherUser] = [
        ...(inboxStore.convs[otherUser] || []),
        msg.data as IFriendRequest,
      ];
      userStore.cacheUser(otherUser);
    }
    if (isFriendRequestResponse(msg)) {
      const otherUser =
        msg.data.friended === authStore.uid
          ? msg.data.friender
          : msg.data.friended;
      let newConv = inboxStore.convs[otherUser] || [];
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
      inboxStore.convs[otherUser] = [...newConv];
      userStore.cacheUser(otherUser);
    }

    if (isInvitation(msg)) {
      const otherUser =
        msg.data.inviter === authStore.uid
          ? msg.data.invited
          : msg.data.inviter;
      inboxStore.convs[otherUser] = [
        ...(inboxStore.convs[otherUser] || []),
        msg.data as IInvitation,
      ];
      userStore.cacheUser(otherUser);
    }
    if (isInvitationResponse(msg)) {
      const otherUser =
        msg.data.inviter === authStore.uid
          ? msg.data.invited
          : msg.data.inviter;
      let newConv = inboxStore.convs[otherUser] || [];
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
      inboxStore.convs[otherUser] = [...newConv];
      userStore.cacheUser(otherUser);
    }

    if (isBlock(msg)) {
      const otherUser =
        msg.data.blocker === authStore.uid
          ? msg.data.blocked
          : msg.data.blocker;
      delete inboxStore.convs[otherUser];
    }
  }

  const watchForCalls = (e: MessageEvent) => {
    const msg = JSON.parse(e.data);
    if (!msg) return;
    if (isCallAcknowledge(msg)) {
      pendingCallsStore.push(msg.data);
    }
    if (isCallResponse(msg)) {
      const i = pendingCallsStore.findIndex(
        (c) => c.called === msg.data.called && c.caller === msg.data.caller
      );
      if (i !== -1) pendingCallsStore.splice(i, 1);
      if (msg.data.accept)
        router.push(
          `/call/${
            msg.data.called === authStore.uid
              ? msg.data.caller
              : msg.data.called
          }${msg.data.caller === authStore.uid ? "?initiator" : ""}`
        );
    }
  };

  const watchAttachments = (e: MessageEvent) => {
    const msg = JSON.parse(e.data);
    if (!msg) return;
    if (isAttachmentProgress(msg)) {
      const i = attachmentStore.attachments.findIndex(
        (a) => a.ID === msg.data.ID
      );
      if (i !== -1) {
        const newMeta = {
          ...attachmentStore.attachments[i],
          ratio: msg.data.ratio,
          failed: msg.data.failed,
        };
        attachmentStore.attachments = [
          ...attachmentStore.attachments.filter((a) => a.ID !== msg.data.ID),
          newMeta,
        ];
      }
    }
    if (isAttachmentMetadataCreated(msg)) {
      attachmentStore.attachments = [
        ...attachmentStore.attachments.filter((a) => a.ID !== msg.data.ID),
        {
          ...msg.data,
          failed: false,
          ratio: 0,
        },
      ];
    }
  };

  onMounted(() => {
    /* Refresh the token */
    refreshTokenInterval.value = setInterval(async () => {
      try {
        if (authStore.uid)
          await makeRequest("/api/acc/refresh", {
            method: "POST",
          });
      } catch (e) {
        resMsg.value = {
          msg: `${e}`,
          err: true,
          pen: false,
        };
        authStore.uid = undefined;
      }
    }, 90000);

    /* Ping the server websocket connection to keep connection alive */
    pingSocketInterval.value = setInterval(
      () => socketStore.send("PING"),
      20000
    );

    /* Remove data for users that haven't been seen in 30 seconds - except for the current users data, also stop watching users depending on visiblity */
    clearUserCacheInterval.value = setInterval(() => {
      const disappeared = userStore.disappearedUsers.map((du) =>
        Date.now() - du.disappearedAt > 30000 && du.id !== authStore.uid
          ? du.id
          : ""
      );
      userStore.users = [
        ...userStore.users.filter((u) => !disappeared.includes(u.ID)),
      ];
      userStore.disappearedUsers = [
        ...userStore.disappearedUsers.filter(
          (du) => !disappeared.includes(du.id)
        ),
      ];
      disappeared.forEach((id) => {
        if (id)
          socketStore.send({
            event_type: "STOP_WATCHING",
            data: { entity: "USER", id },
          } as StopWatching);
      });
      currentlyWatching.value = currentlyWatching.value.filter(
        (id) => !disappeared.includes(id)
      );
    }, 30000);

    /* Remove data for rooms that haven't been seen in 30 seconds, also stop watching rooms depending on visiblity */
    clearRoomCacheInterval.value = setInterval(() => {
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
      currentlyWatching.value = currentlyWatching.value.filter(
        (id) => !disappeared.includes(id)
      );
    }, 30000);

    /* Remove data for attachments that haven't been seen in 30 seconds */
    clearAttachmentCacheInterval.value = setInterval(() => {
      const disappeared = attachmentStore.disappearedAttachments.map((da) =>
        Date.now() - da.disappearedAt > 30000 ? da.id : ""
      );
      attachmentStore.attachments = [
        ...attachmentStore.attachments.filter(
          (a) => !disappeared.includes(a.ID)
        ),
      ];
      attachmentStore.disappearedAttachments = [
        ...attachmentStore.disappearedAttachments.filter(
          (da) => !disappeared.includes(da.id)
        ),
      ];
    }, 30000);
  });

  /* Automatically start watching users */
  watchEffect(() => {
    userStore.visibleUsers.forEach((id) => {
      if (currentlyWatching.value.includes(id)) return;
      socketStore.send({
        event_type: "START_WATCHING",
        data: {
          entity: "USER",
          id,
        },
      } as StartWatching);
      currentlyWatching.value.push(id);
    });
  });

  /* Automatically start watching rooms */
  watchEffect(() => {
    roomStore.visibleRooms.forEach((id) => {
      if (currentlyWatching.value.includes(id)) return;
      socketStore.send({
        event_type: "START_WATCHING",
        data: {
          entity: "ROOM",
          id,
        },
      } as StartWatching);
      currentlyWatching.value.push(id);
    });
  });

  watchEffect(() => {
    socketStore.socket?.addEventListener("message", watchForChangeEvents);
    socketStore.socket?.addEventListener("message", watchInbox);
    socketStore.socket?.addEventListener("message", watchForCalls);
    socketStore.socket?.addEventListener("message", watchAttachments);
  });

  onBeforeUnmount(() => {
    clearInterval(refreshTokenInterval.value);
    clearInterval(pingSocketInterval.value);
    clearInterval(clearUserCacheInterval.value);
    clearInterval(clearRoomCacheInterval.value);
    clearInterval(clearAttachmentCacheInterval.value);

    socketStore.socket?.removeEventListener("message", watchForChangeEvents);
    socketStore.socket?.removeEventListener("message", watchInbox);
    socketStore.socket?.removeEventListener("message", watchForCalls);
    socketStore.socket?.removeEventListener("message", watchAttachments);
  });

  return undefined;
}
