<script lang="ts" setup>
import { IResMsg, IRoomChannel } from "../../interfaces/GeneralInterfaces";
import { onBeforeUnmount, onMounted, toRef, ref } from "vue";
import { useRoute } from "vue-router";
import {
  JoinRoom,
  LeaveRoom,
  RoomMessage as RoomMessageEvent,
} from "../../socketHandling/OutEvents";
import { getRoomChannel } from "../../services/room";
import {
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

const roomChannelStore = useRoomChannelStore();
const roomStore = useRoomStore();
const socketStore = useSocketStore();
const userStore = useUserStore();

const route = useRoute();
const roomId = toRef(route.params, "id");

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

  roomStore.roomEnteredView(roomId.value as string);

  socketStore.socket?.removeEventListener("message", handleMessages);
});

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
}

function handleSubmit(values: any) {
  if (!roomChannelStore.current) return;
  const content: string = values.message;
  socketStore.send({
    event_type: "ROOM_MESSAGE",
    data: { content, channel_id: roomChannelStore.current },
  } as RoomMessageEvent);
}
</script>

<template>
  <div class="room">
    <div class="channels-messages">
      <div class="channels">
        <!-- Main channel -->
        <Channel
          v-if="roomChannelStore.channels.find((c) => c.main) as IRoomChannel"
          :channel="roomChannelStore.channels.find((c) => c.main) as IRoomChannel"
        />
        <!-- Secondary channels -->
        <Channel
          :channel="channel"
          v-for="channel in roomChannelStore.channels.filter((c) => !c.main)"
        />
      </div>
      <div v-if="!resMsg.pen && !resMsg.err" class="messages">
        <div class="list">
          <RoomMessage :msg="msg" v-for="msg in messages" />
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
      width: 10rem;
      height: 100%;
      padding: var(--gap-sm);
      button:first-of-type {
        margin-bottom: var(--gap-md);
        font-weight: 600;
      }
    }
    .channels,
    .messages {
      border: 2px solid var(--border-pale);
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
    border: 2px solid var(--border-pale);
    border-radius: var(--border-radius-md);
  }
}
</style>
