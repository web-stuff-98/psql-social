import { onBeforeUnmount, onMounted, Ref, ref } from "vue";
import { IResMsg } from "../interfaces/GeneralInterfaces";
import { makeRequest } from "../services/makeRequest";
import { StopWatching } from "../socketHandling/OutEvents";

import useAuthStore from "../store/AuthStore";
import useSocketStore from "../store/SocketStore";
import useUserStore from "../store/UserStore";
import useRoomStore from "../store/RoomStore";

/*
  This composable is for intervals that run in the background
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
  });

  onBeforeUnmount(() => {
    clearInterval(refreshTokenInterval.value);
    clearInterval(pingSocketInterval.value);
    clearInterval(clearUserCacheInterval.value);
    clearInterval(clearRoomCacheInterval.value);
  });

  return undefined;
}
