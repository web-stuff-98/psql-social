import { onBeforeUnmount, onMounted, Ref, ref, watch, watchEffect } from "vue";
import { IResMsg } from "../interfaces/GeneralInterfaces";
import { StartWatching } from "../socketHandling/OutEvents";
import { isChangeEvent } from "../socketHandling/InterpretEvent";
import { refreshToken } from "../services/account";
import useAuthStore from "../store/AuthStore";
import useSocketStore from "../store/SocketStore";
import useUserStore from "../store/UserStore";
import useRoomStore from "../store/RoomStore";
import useInboxStore from "../store/InboxStore";
import useAttachmentStore from "../store/AttachmentStore";
import useInterface from "../store/InterfaceStore";
import useCallStore from "../store/CallsStore";
import useNotificationStore from "../store/NotificationStore";

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
  const interfaceStore = useInterface();
  const callStore = useCallStore();
  const notificationsStore = useNotificationStore();

  const currentlyWatching = ref<string[]>([]);

  watch(interfaceStore, (_, newVal) => {
    if (newVal.darkMode) document.body.classList.add("dark-mode");
    else document.body.classList.remove("dark-mode");
  });

  async function watchForChangeEvents(e: MessageEvent) {
    const msg = JSON.parse(e.data);
    if (!msg) return;
    if (isChangeEvent(msg)) {
      if (msg.data.entity === "ROOM") roomStore.changeEvent(msg);
      if (msg.data.entity === "USER") userStore.changeEvent(msg);
    }
  }

  const watchInbox = (e: MessageEvent) => inboxStore.watchInbox(e);
  const watchForCalls = (e: MessageEvent) => callStore.watchCalls(e);
  const watchAttachments = (e: MessageEvent) =>
    attachmentStore.watchAttachments(e);
  const watchDirectMessageNotifications = (e: MessageEvent) =>
    notificationsStore.watchDirectMessageNotifications(e);

  onMounted(() => {
    /* Refresh the token */
    refreshTokenInterval.value = setInterval(async () => {
      try {
        if (authStore.uid) await refreshToken();
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
    clearUserCacheInterval.value = setInterval(
      userStore.cleanupInterval,
      30000
    );
    clearRoomCacheInterval.value = setInterval(
      roomStore.cleanupInterval,
      30000
    );
    clearAttachmentCacheInterval.value = setInterval(
      attachmentStore.cleanupInterval,
      30000
    );
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
    socketStore.socket?.addEventListener(
      "message",
      watchDirectMessageNotifications
    );
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
    socketStore.socket?.removeEventListener(
      "message",
      watchDirectMessageNotifications
    );
  });

  return undefined;
}
