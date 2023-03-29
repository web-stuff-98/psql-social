/* Intervals that run in the background (refresh token, ping socket, et cet) */
import { onBeforeUnmount, onMounted, Ref, ref } from "vue";
import { IResMsg } from "../interfaces/GeneralInterfaces";
import { makeRequest } from "../services/makeRequest";
import useAuthStore from "../store/AuthStore";

export default function useIntervals({
  resMsg,
}: {
  resMsg: Ref<IResMsg | undefined>;
}) {
  const refreshTokenInterval = ref<NodeJS.Timer>();
  const authStore = useAuthStore();

  onMounted(() => {
    refreshTokenInterval.value = setInterval(async () => {
      try {
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
  });
  onBeforeUnmount(() => {
    clearInterval(refreshTokenInterval.value);
  });

  return undefined;
}
