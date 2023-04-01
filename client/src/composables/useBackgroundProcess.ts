import { onBeforeUnmount, onMounted, Ref, ref, watchEffect } from "vue";
import {
  IDirectMessage,
  IFriendRequest,
  IInvitation,
  IResMsg,
  IRoom,
} from "../interfaces/GeneralInterfaces";
import { makeRequest } from "../services/makeRequest";
import { StartWatching, StopWatching } from "../socketHandling/OutEvents";

import useAuthStore from "../store/AuthStore";
import useSocketStore from "../store/SocketStore";
import useUserStore from "../store/UserStore";
import useRoomStore from "../store/RoomStore";
import {
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
import useInboxStore from "../store/InboxStore";

/**
 * This composable is for intervals that run in the background
 * and socket event listeners that aren't tied to components
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

  const authStore = useAuthStore();
  const socketStore = useSocketStore();
  const userStore = useUserStore();
  const roomStore = useRoomStore();
  const inboxStore = useInboxStore();

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
    }
    if (isInvitationResponse(msg)) {
      const otherUser =
        msg.data.inviter === authStore.uid
          ? msg.data.invited
          : msg.data.inviter;
      let newConv = inboxStore.convs[otherUser] || [];
      const i = newConv.findIndex((item) => {
        // if it has an "inviter" then its an invitate
        if ((item as any)["inviter"] !== undefined)
          return (
            (item as any)["inviter"] === msg.data.inviter &&
            (item as any)["invited"] === msg.data.invited
          );
      });
      //@ts-ignore
      newConv[i]["accepted"] = msg.data.accepted;
      inboxStore.convs[otherUser] = [...newConv];
    }
  }

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
      disappeared.forEach((id) =>
        socketStore.send({
          event_type: "STOP_WATCHING",
          data: { entity: "USER", id },
        } as StopWatching)
      );
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
      disappeared.forEach((id) =>
        socketStore.send({
          event_type: "STOP_WATCHING",
          data: { entity: "ROOM", id },
        } as StopWatching)
      );
      currentlyWatching.value = currentlyWatching.value.filter(
        (id) => !disappeared.includes(id)
      );
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
  });

  onBeforeUnmount(() => {
    clearInterval(refreshTokenInterval.value);
    clearInterval(pingSocketInterval.value);
    clearInterval(clearUserCacheInterval.value);
    clearInterval(clearRoomCacheInterval.value);

    socketStore.socket?.removeEventListener("message", watchForChangeEvents);
    socketStore.socket?.removeEventListener("message", watchInbox);
  });

  return undefined;
}
