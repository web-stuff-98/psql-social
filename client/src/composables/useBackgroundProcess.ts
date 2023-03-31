import { onBeforeUnmount, onMounted, Ref, ref, watchEffect } from "vue";
import { IResMsg, IRoom, IUser } from "../interfaces/GeneralInterfaces";
import { makeRequest } from "../services/makeRequest";
import { StartWatching, StopWatching } from "../socketHandling/OutEvents";

import useAuthStore from "../store/AuthStore";
import useSocketStore from "../store/SocketStore";
import useUserStore from "../store/UserStore";
import useRoomStore from "../store/RoomStore";
import { isChangeEvent } from "../socketHandling/InterpretEvent";

/*
  This composable is for intervals that run in the background,
  and socket event listeners that need to run constantly
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

  // Watch for change events on rooms and users (bio watch is done from component)
  function watchForChangeEvents(e: MessageEvent) {
    const msg = JSON.parse(e.data);
    if (!msg) return;
    if (isChangeEvent(msg)) {
      // Watch for room update events
      if (msg.data.entity === "ROOM") {
        if (msg.data.change_type === "UPDATE") {
          const i = roomStore.rooms.findIndex((r) => r.ID === msg.data.data.ID);
          if (i !== -1)
            roomStore.rooms[i] = {
              ...roomStore.rooms[i],
              ...(msg.data.data as Partial<IRoom>),
            };
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
        if (msg.data.change_type === "UPDATE") {
          const i = userStore.users.findIndex((u) => u.ID === msg.data.data.ID);
          if (i !== -1)
            userStore.users[i] = {
              ...userStore.users[i],
              ...(msg.data.data as Partial<IUser>),
            };
        }
        if (msg.data.change_type === "DELETE") {
          const i = userStore.users.findIndex((u) => u.ID === msg.data.data.ID);
          if (i !== -1) userStore.users.splice(i, 1);
        }
      }
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
      const disappared = userStore.disappearedUsers.map((du) =>
        Date.now() - du.disappearedAt > 30000 && du.id !== authStore.uid
          ? du.id
          : ""
      );
      userStore.users = [
        ...userStore.users.filter((u) => !disappared.includes(u.ID)),
      ];
      userStore.disappearedUsers = [
        ...userStore.disappearedUsers.filter(
          (du) => !disappared.includes(du.id)
        ),
      ];
      disappared.forEach((id) =>
        socketStore.send({
          event_type: "STOP_WATCHING",
          data: { entity: "USER", id },
        } as StopWatching)
      );
    }, 30000);

    /* Remove data for rooms that haven't been seen in 30 seconds - except for the current users data, also stop watching users depending on visiblity */
    clearRoomCacheInterval.value = setInterval(() => {
      const disappared = roomStore.disappearedRooms.map((dr) =>
        Date.now() - dr.disappearedAt > 30000 ? dr.id : ""
      );
      roomStore.rooms = [
        ...roomStore.rooms.filter((r) => !disappared.includes(r.ID)),
      ];
      roomStore.disappearedRooms = [
        ...roomStore.disappearedRooms.filter(
          (dr) => !disappared.includes(dr.id)
        ),
      ];
      disappared.forEach((id) =>
        socketStore.send({
          event_type: "STOP_WATCHING",
          data: { entity: "ROOM", id },
        } as StopWatching)
      );
    }, 30000);

    /* Automatically start watching users */
    watchEffect(() => {
      userStore.visibleUsers.forEach((id) =>
        socketStore.send({
          event_type: "START_WATCHING",
          data: {
            entity: "USER",
            id,
          },
        } as StartWatching)
      );
    });

    /* Automatically start watching rooms */
    watchEffect(() => {
      roomStore.visibleRooms.forEach((id) =>
        socketStore.send({
          event_type: "START_WATCHING",
          data: {
            entity: "ROOM",
            id,
          },
        } as StartWatching)
      );
    });

    socketStore.socket?.addEventListener("message", watchForChangeEvents);
  });

  onBeforeUnmount(() => {
    clearInterval(refreshTokenInterval.value);
    clearInterval(pingSocketInterval.value);
    clearInterval(clearUserCacheInterval.value);
    clearInterval(clearRoomCacheInterval.value);

    socketStore.socket?.removeEventListener("message", watchForChangeEvents);
  });

  return undefined;
}
