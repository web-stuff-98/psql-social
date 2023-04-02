<script lang="ts" setup>
import { IResMsg, IRoomChannel } from "../../interfaces/GeneralInterfaces";
import { onBeforeUnmount, onMounted, toRef, ref, computed } from "vue";
import { useRoute } from "vue-router";
import {
  JoinRoom,
  LeaveRoom,
  RoomMessage as RoomMessageEvent,
  JoinChannel,
  LeaveChannel,
} from "../../socketHandling/OutEvents";
import { getRoomChannel } from "../../services/room";
import {
  isBan,
  isChangeEvent,
  isRoomMsg,
  isRoomMsgDelete,
  isRoomMsgUpdate,
} from "../../socketHandling/InterpretEvent";
import { IRoomMessage } from "../../interfaces/GeneralInterfaces";
import MessageForm from "../../components/shared/MessageForm.vue";
import useSocketStore from "../../store/SocketStore";
import useRoomStore from "../../store/RoomStore";
import useUserStore from "../../store/UserStore";
import useRoomChannelStore from "../../store/RoomChannelStore";
import ResMsg from "../../components/shared/ResMsg.vue";
import RoomMessage from "../../components/shared/Message.vue";
import Channel from "./Channel.vue";
import useAuthStore from "../../store/AuthStore";
import router from "../../router";
import EditRoomChannel from "./EditRoomChannel.vue";

const roomChannelStore = useRoomChannelStore();
const roomStore = useRoomStore();
const socketStore = useSocketStore();
const userStore = useUserStore();
const authStore = useAuthStore();

const route = useRoute();
const roomId = toRef(route.params, "id");

const room = computed(() => roomStore.getRoom(roomId.value as string));

const resMsg = ref<IResMsg>({});

const messages = ref<IRoomMessage[]>([]);

onMounted(async () => {
  socketStore.send({
    event_type: "JOIN_ROOM",
    data: { room_id: roomId.value },
  } as JoinRoom);

  try {
    resMsg.value = { msg: "", err: false, pen: true };
    const main = await roomChannelStore.getRoomChannels(roomId.value as string);
    const msgs = await getRoomChannel(main);
    if (msgs) msgs.forEach((m) => userStore.cacheUser(m.author_id));
    messages.value = msgs || [];
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

function joinChannel(channelId: string) {
  socketStore.send({
    event_type: "LEAVE_CHANNEL",
    data: { channel_id: roomChannelStore.current },
  } as LeaveChannel);
  roomChannelStore.current = channelId;
  socketStore.send({
    event_type: "JOIN_CHANNEL",
    data: { channel_id: channelId },
  } as JoinChannel);
}

function handleMessages(e: MessageEvent) {
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
  }
}

function handleSubmit(values: any) {
  if (!roomChannelStore.current) return;
  const content: string = values.message;
  socketStore.send({
    event_type: "ROOM_MESSAGE",
    data: { content, channel_id: roomChannelStore.current },
  } as RoomMessageEvent);
}

const isEditingChannel = ref("");
function editChannelClicked(channelId: string) {
  isEditingChannel.value = channelId;
}
</script>

<template>
  <div class="room">
    <div class="channels-messages">
      <div class="channels">
        <div class="channels-list">
          <div class="list">
            <!-- Main channel -->
            <Channel
              :editClicked="editChannelClicked"
              :isAuthor="authStore.uid === room?.author_id"
              v-if="roomChannelStore.channels.find((c) => c.main) as IRoomChannel"
              :channel="roomChannelStore.channels.find((c) => c.main) as IRoomChannel"
            />
            <!-- Secondary channels -->
            <Channel
              :editClicked="editChannelClicked"
              :isAuthor="authStore.uid === room?.author_id"
              :channel="channel"
              v-for="channel in roomChannelStore.channels.filter(
                (c) => !c.main
              )"
            />
          </div>
        </div>
        <button type="button" name="create room" class="create-button">
          <v-icon name="io-add-circle-sharp" />
          Create
        </button>
      </div>
      <div v-if="!resMsg.pen && !resMsg.err" class="messages">
        <div class="list">
          <RoomMessage
            :isAuthor="authStore.uid === msg.author_id"
            :roomMsg="true"
            :msg="msg"
            v-for="msg in messages"
          />
        </div>
      </div>
      <div v-else class="res-msg-container">
        <ResMsg :resMsg="resMsg" />
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
    }
    .channels {
      width: 14rem;
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
    .messages {
      border: 2px solid var(--border-light);
      border-radius: var(--border-radius-md);
      position: relative;
      overflow: hidden;
      height: 100%;
    }
    .messages {
      width: 100%;
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
