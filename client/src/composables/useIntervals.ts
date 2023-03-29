/* Intervals that run in the background (refresh token, ping socket, et cet) */
import { onBeforeUnmount, onMounted, Ref, ref } from "vue";
import { IResMsg } from "../interfaces/GeneralInterfaces";
import { makeRequest } from "../services/makeRequest";
import useAuthStore from "../store/AuthStore";
import useSocketStore from "../store/SocketStore";

export default function useIntervals({
  resMsg,
}: {
  resMsg: Ref<IResMsg | undefined>;
}) {
  const refreshTokenInterval = ref<NodeJS.Timer>();
  const pingSocketInterval = ref<NodeJS.Timer>();

  const authStore = useAuthStore();
  const socketStore = useSocketStore();

  onMounted(() => {
    refreshTokenInterval.value = setInterval(async () => {
      try {
        if (authStore.user)
          await makeRequest("/api/acc/refresh", {
            method: "POST",
          });
      } catch (e) {
        resMsg.value = {
          msg: `${e}`,
          err: true,
          pen: false,
        };
        authStore.user = undefined;
      }
    }, 90000);

    pingSocketInterval.value = setInterval(() => {
      socketStore.send("PING");
    }, 20000);
  });

  onBeforeUnmount(() => {
    clearInterval(refreshTokenInterval.value);
    clearInterval(pingSocketInterval.value);
  });

  return undefined;
}
