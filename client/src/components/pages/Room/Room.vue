<script lang="ts" setup>
import { IResMsg, IRoomChannel } from "../../../interfaces/GeneralInterfaces";
import { onBeforeUnmount, onMounted, toRef, ref, watch, computed } from "vue";
import { useRoute } from "vue-router";
import {
  JoinRoom,
  LeaveRoom,
  RoomMessage as RoomMessageEvent,
  JoinChannel,
  LeaveChannel,
} from "../../../socketHandling/OutEvents";
import { deleteRoomChannel, getRoomChannel } from "../../../services/room";
import {
  isBan,
  isChangeEvent,
  isRoomMsg,
  isRoomMsgDelete,
  isRoomMsgUpdate,
  isRequestAttachment,
  isRoomChannelWebRTCUserJoined,
  isRoomChannelWebRTCUserLeft,
} from "../../../socketHandling/InterpretEvent";
import { IRoomMessage } from "../../../interfaces/GeneralInterfaces";
import MessageForm from "../../../components/shared/MessageForm.vue";
import useSocketStore from "../../../store/SocketStore";
import useRoomStore from "../../../store/RoomStore";
import useUserStore from "../../../store/UserStore";
import useRoomChannelStore from "../../../store/RoomChannelStore";
import useAttachmentStore from "../../../store/AttachmentStore";
import useNotificationStore from "../../../store/NotificationStore";
import ResMsg from "../../../components/shared/ResMsg.vue";
import RoomMessage from "../../../components/shared/Message.vue";
import Channel from "./Channel.vue";
import useAuthStore from "../../../store/AuthStore";
import router from "../../../router";
import EditRoomChannel from "./EditRoomChannel.vue";
import CreateRoomChannel from "./CreateRoomChannel.vue";
import RoomVidChat from "./RoomVidChat.vue";
import User from "../../shared/User.vue";

const roomChannelStore = useRoomChannelStore();
const roomStore = useRoomStore();
const socketStore = useSocketStore();
const userStore = useUserStore();
const authStore = useAuthStore();
const attachmentStore = useAttachmentStore();
const notificationStore = useNotificationStore();

const route = useRoute();
const roomId = toRef(route.params, "id");
const pendingAttachmentFile = ref<File>();

const isEditingChannel = ref("");
const isCreatingChannel = ref(false);
function editChannelClicked(channelId: string) {
  isEditingChannel.value = channelId;
}

const vidChatOpen = ref(false);

async function deleteChannelClicked(channelId: string) {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    await deleteRoomChannel(channelId);
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
}

const room = computed(() => roomStore.getRoom(roomId.value as string));

const resMsg = ref<IResMsg>({});

const messages = ref<IRoomMessage[]>([]);
const messagesBottomRef = ref<HTMLElement>();

onMounted(async () => {
  socketStore.send({
    event_type: "JOIN_ROOM",
    data: { room_id: roomId.value },
  } as JoinRoom);

  try {
    resMsg.value = { msg: "", err: false, pen: true };
    const main = await roomChannelStore.getRoomChannels(roomId.value as string);
    const { messages: msgs, users_in_webrtc } = await getRoomChannel(main);
    notificationStore.clearChannelNotifications(roomId.value as string, main);
    if (msgs) msgs.forEach((m) => userStore.cacheUser(m.author_id));
    messages.value = msgs || [];
    if (users_in_webrtc)
      users_in_webrtc.forEach((uid) => userStore.cacheUser(uid));
    roomChannelStore.uidsInCurrentWebRTCChat = users_in_webrtc || [];
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }

  roomStore.roomEnteredView(roomId.value as string);

  socketStore.socket?.addEventListener("message", handleMessages);
});

onBeforeUnmount(() => {
  socketStore.send({
    event_type: "LEAVE_ROOM",
    data: { room_id: roomId.value },
  } as LeaveRoom);

  roomStore.roomLeftView(roomId.value as string);

  socketStore.socket?.removeEventListener("message", handleMessages);
});

async function joinChannel(channelId: string) {
  // got to leave old channel first, if null channel server handle it
  socketStore.send({
    event_type: "LEAVE_CHANNEL",
    data: { channel_id: roomChannelStore.current },
  } as LeaveChannel);
  roomChannelStore.current = channelId;

  messages.value = [];

  socketStore.send({
    event_type: "JOIN_CHANNEL",
    data: { channel_id: channelId },
  } as JoinChannel);

  notificationStore.clearChannelNotifications(
    roomId.value as string,
    channelId
  );

  try {
    resMsg.value = { msg: "", err: false, pen: true };
    const { messages: msgs, users_in_webrtc } = await getRoomChannel(channelId);
    if (msgs) {
      for await (const msg of msgs) {
        await userStore.cacheUser(msg.author_id);
      }
    }
    messages.value = msgs || [];
    if (users_in_webrtc) {
      for await (const uid of users_in_webrtc) {
        await userStore.cacheUser(uid);
      }
    }
    roomChannelStore.uidsInCurrentWebRTCChat = users_in_webrtc || [];
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
}

async function handleMessages(e: MessageEvent) {
  const msg = JSON.parse(e.data);
  if (!msg) return;

  if (isRoomMsg(msg)) {
    userStore.cacheUser(msg.data.ID);
    messages.value = [...messages.value, msg.data];
  }

  if (isRoomMsgDelete(msg)) {
    const i = messages.value.findIndex((m) => m.ID === msg.data.ID);
    if (i !== -1) messages.value.splice(i, 1);
  }

  if (isRoomMsgUpdate(msg)) {
    const i = messages.value.findIndex((m) => m.ID === msg.data.ID);
    if (i !== -1) messages.value[i].content = msg.data.content;
  }

  if (isBan(msg)) {
    if (msg.data.room_id !== roomId.value) return;
    if (msg.data.user_id === authStore.uid) {
      router.push("/");
      return;
    }
    messages.value = messages.value.filter(
      (m) => m.author_id !== msg.data.user_id
    );
  }

  if (isChangeEvent(msg)) {
    if (msg.data.entity === "CHANNEL") {
      if (msg.data.change_type === "UPDATE") {
        const i = roomChannelStore.channels.findIndex(
          (c) => c.ID === msg.data.data.ID
        );
        if (i !== -1) {
          const newChannel = {
            ...roomChannelStore.channels[i],
            ...msg.data.data,
          };
          roomChannelStore.channels = [
            ...roomChannelStore.channels
              .filter((c) => c.ID !== msg.data.data.ID)
              .map((c) => ({
                ...c,
                main: (msg.data.data as any)["main"] ? false : c.main,
              })),
            newChannel,
          ];
        }
      }
      if (msg.data.change_type === "INSERT") {
        roomChannelStore.channels = [
          ...roomChannelStore.channels.map((c) => ({
            ...c,
            main: (msg.data.data as any)["main"] ? false : c.main,
          })),
          msg.data.data as IRoomChannel,
        ];
      }
      if (msg.data.change_type === "DELETE") {
        const i = roomChannelStore.channels.findIndex(
          (c) => c.ID === msg.data.data.ID
        );
        if (i !== -1) {
          if (roomChannelStore.channels[i].ID === roomChannelStore.current) {
            joinChannel(
              roomChannelStore.channels.find((c) => c.ID === msg.data.data.ID)
                ?.ID!
            );
          }
          roomChannelStore.channels.splice(i, 1);
        }
      }
    }
    if (msg.data.entity === "USER") {
      if (msg.data.change_type === "DELETE") {
        messages.value = [
          ...messages.value.filter((m) => m.author_id === msg.data.data.ID),
        ];
      }
    }
    if (msg.data.entity === "ROOM")
      if (msg.data.change_type === "DELETE")
        if (msg.data.data.ID === roomId.value) router.push("/");
  }

  if (isRoomChannelWebRTCUserJoined(msg)) {
    roomChannelStore.uidsInCurrentWebRTCChat = [
      ...roomChannelStore.uidsInCurrentWebRTCChat,
      msg.data.uid,
    ];
  }

  if (isRoomChannelWebRTCUserLeft(msg)) {
    const i = roomChannelStore.uidsInCurrentWebRTCChat.findIndex(
      (uid) => uid === msg.data.uid
    );
    if (i !== -1) roomChannelStore.uidsInCurrentWebRTCChat.splice(i, 1);
  }

  if (isRequestAttachment(msg)) {
    if (pendingAttachmentFile.value)
      attachmentStore.uploadAttachment(
        pendingAttachmentFile.value,
        msg.data.ID
      );
    else
      console.warn(
        "Server requested attachment file, but attachment file is undefined"
      );
  }
}

function handleSubmit(values: any, file?: File) {
  if (!roomChannelStore.current) return;
  const content: string = values.message;
  socketStore.send({
    event_type: "ROOM_MESSAGE",
    data: {
      content,
      channel_id: roomChannelStore.current,
      has_attachment: Boolean(file),
    },
  } as RoomMessageEvent);
  pendingAttachmentFile.value = file;
}

watch(messages, (oldVal, newVal) => {
  if (newVal && oldVal)
    if (newVal.length > oldVal.length)
      messagesBottomRef.value?.scrollIntoView({ behavior: "auto" });
});
</script>

<template>
  <div class="room">
    <div class="channels-messages">
      <div class="channels">
        <div class="channels-list">
          <div class="list">
            <!-- Main channel -->
            <Channel
              :notificationCount="notificationStore.getChannelNotifications(roomId as string,roomChannelStore.channels.find((c) => c.main)?.ID as string)"
              :joinChannel="joinChannel"
              :deleteClicked="deleteChannelClicked"
              :editClicked="editChannelClicked"
              :isAuthor="authStore.uid === room?.author_id"
              v-if="roomChannelStore.channels.find((c) => c.main) as IRoomChannel"
              :channel="roomChannelStore.channels.find((c) => c.main) as IRoomChannel"
            />
            <!-- Secondary channels -->
            <Channel
              :notificationCount="notificationStore.getChannelNotifications(roomId as string, channel.ID)"
              :joinChannel="joinChannel"
              :deleteClicked="deleteChannelClicked"
              :editClicked="editChannelClicked"
              :isAuthor="authStore.uid === room?.author_id"
              :channel="channel"
              v-for="channel in roomChannelStore.channels.filter(
                (c) => !c.main
              )"
            />
          </div>
        </div>
        <button
          v-if="room?.author_id === authStore.uid"
          @click="isCreatingChannel = true"
          type="button"
          name="create channel"
          class="create-button"
        >
          <v-icon name="io-add-circle-sharp" />
          Create
        </button>
      </div>
      <div class="messages-vid-chat">
        <div class="vid-chat">
          <button
            class="join-button"
            @click="vidChatOpen = true"
            v-if="!vidChatOpen"
            type="button"
          >
            Enter channel voip/video chat
          </button>
          <User
            :noPfp="true"
            :uid="uid"
            v-for="uid in roomChannelStore.uidsInCurrentWebRTCChat"
            v-if="!vidChatOpen"
          />
          <RoomVidChat
            :exitButtonClicked="() => (vidChatOpen = false)"
            v-if="vidChatOpen"
          />
        </div>
        <div v-if="!resMsg.pen && !resMsg.err" class="messages">
          <div class="list">
            <RoomMessage
              :isAuthor="authStore.uid === msg.author_id"
              :roomId="roomId as string"
              :msg="msg"
              v-for="msg in messages"
            />
            <div class="bottom" ref="messagesBottomRef" />
          </div>
        </div>
        <div v-else class="res-msg-container">
          <ResMsg :resMsg="resMsg" />
        </div>
      </div>
    </div>
    <div class="form-container">
      <MessageForm :handleSubmit="handleSubmit" />
    </div>
  </div>
  <EditRoomChannel
    v-if="isEditingChannel"
    :channelId="isEditingChannel"
    :closeClicked="() => (isEditingChannel = '')"
  />
  <CreateRoomChannel
    v-if="isCreatingChannel"
    :closeClicked="() => (isCreatingChannel = false)"
    :roomId="roomId as string"
  />
</template>

<style lang="scss" scoped>
.room {
  width: 100%;
  height: 100%;
  padding: var(--gap-sm);
  gap: calc(var(--gap-md) - 1px);
  padding-bottom: calc(var(--gap-sm) + 1px);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  .res-msg-container {
    flex-grow: 1;
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .channels-messages {
    height: 100%;
    width: 100%;
    display: flex;
    gap: calc(var(--gap-md) - 1px);
    .list {
      display: flex;
      flex-direction: column;
      padding: var(--gap-md);
      gap: var(--gap-md);
      width: 100%;
      height: 100%;
      position: absolute;
      left: 0;
      top: 0;
      overflow-y: auto;
      .bottom {
        width: 100%;
        height: 0px;
        padding: 0;
        margin: 0;
      }
    }
    .channels {
      width: fit-content;
      min-width: 10rem;
      height: 100%;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: flex-end;
      .channels-list {
        display: flex;
        width: 100%;
        flex-grow: 1;
        position: relative;
        button:first-of-type {
          margin-bottom: var(--gap-md);
          font-weight: 600;
        }
      }
      .create-button {
        padding: var(--gap-sm);
        background: none;
        width: 100%;
        border: none;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--text-colour);
        text-shadow: none;
        font-weight: 600;
        gap: 3px;
        border-radius: 0;
        border-top: 1px solid var(--border-pale);
        svg {
          width: 1.333rem;
          height: 1.333rem;
          fill: var(--text-colour);
        }
      }
      .create-button:hover {
        background: var(--border-pale);
      }
    }
    .channels,
    .messages,
    .vid-chat {
      border: 2px solid var(--border-light);
      border-radius: var(--border-radius-md);
      position: relative;
      overflow: hidden;
      height: 100%;
    }
    .messages {
      width: 100%;
    }
    .messages-vid-chat {
      display: flex;
      flex-direction: column;
      width: 100%;
      gap: var(--gap-md);
      .vid-chat {
        width: 100%;
        height: fit-content;
        padding: var(--gap-sm);
        display: flex;
        align-items: center;
        justify-content: flex-start;
        flex-wrap: wrap;
        gap: 3px;
        button {
          font-size: var(--xs);
          padding: 3px 6px;
        }
      }
    }
  }
  .form-container {
    width: 100%;
    padding: var(--gap-sm);
    border: 2px solid var(--border-light);
    border-radius: var(--border-radius-md);
  }
}
</style>
