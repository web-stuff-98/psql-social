<script lang="ts" setup>
import { IResMsg } from "../interfaces/GeneralInterfaces";
import { onBeforeUnmount, onMounted, toRef, ref } from "vue";
import { useRoute } from "vue-router";
import { JoinRoom, LeaveRoom, RoomMessage } from "../socketHandling/OutEvents";
import { isRoomMsg } from "../socketHandling/InterpretEvent";
import { IRoomMessage } from "../interfaces/GeneralInterfaces";
import MessageForm from "../components/shared/MessageForm.vue";
import useSocketStore from "../store/SocketStore";
import useRoomChannelStore from "../store/RoomChannelStore";
import ResMsg from "../components/shared/ResMsg.vue";

const roomChannelStore = useRoomChannelStore();
const socketStore = useSocketStore();

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
    await roomChannelStore.getRoomChannels(roomId.value as string);
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }

  socketStore.socket?.addEventListener("message", handleMessages);
});

onBeforeUnmount(() => {
  socketStore.send({
    event_type: "LEAVE_ROOM",
    data: { room_id: roomId.value },
  } as LeaveRoom);

  socketStore.socket?.removeEventListener("message", handleMessages);
});

function handleMessages(e: MessageEvent) {
  const msg = JSON.parse(e.data);
  if (!msg) return;

  if (isRoomMsg(msg)) {
    messages.value = [...messages.value, msg];
  }
}

function handleSubmit(values: any) {
  if (!roomChannelStore.current) return;
  const content: string = values.message;
  socketStore.send({
    event_type: "ROOM_MESSAGE",
    data: { content, channel_id: roomChannelStore.current },
  } as RoomMessage);
}
</script>

<template>
  <div class="room">
    <div v-if="!resMsg.pen && !resMsg.err" class="messages">
      {{ messages }}
    </div>
    <div v-else class="res-msg-container">
      <ResMsg :resMsg="resMsg" />
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
  .messages {
    width: 100%;
    height: 100%;
    border: 2px solid var(--border-pale);
    border-radius: var(--border-radius-md);
  }
  .form-container {
    width: 100%;
    padding: var(--gap-sm);
    border: 2px solid var(--border-pale);
    border-radius: var(--border-radius-md);
  }
}
</style>
