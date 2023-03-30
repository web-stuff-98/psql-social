/* Intervals that run in the background (refresh token, ping socket, et cet) */
import { onBeforeUnmount, onMounted, Ref, ref } from "vue";
import { IResMsg } from "../interfaces/GeneralInterfaces";
import { makeRequest } from "../services/makeRequest";

import useAuthStore from "../store/AuthStore";
import useSocketStore from "../store/SocketStore";
import useUserStore from "../store/UserStore";

export default function useIntervals({
  resMsg,
}: {
  resMsg: Ref<IResMsg | undefined>;
}) {
  const refreshTokenInterval = ref<NodeJS.Timer>();
  const pingSocketInterval = ref<NodeJS.Timer>();
  const clearUserCacheInterval = ref<NodeJS.Timer>();

  const authStore = useAuthStore();
  const socketStore = useSocketStore();
  const userStore = useUserStore();

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

    /* Remove data for users that haven't been seen in 30 seconds - except for the current users data */
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
    }, 30000);
  });

  onBeforeUnmount(() => {
    clearInterval(refreshTokenInterval.value);
    clearInterval(pingSocketInterval.value);
  });

  return undefined;
}
