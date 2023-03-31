<script lang="ts" setup>
import { onBeforeUnmount, onMounted, toRef } from "vue";
import { useRoute } from "vue-router";
import { JoinRoom, LeaveRoom } from "../socketHandling/OutEvents";
import useSocketStore from "../store/SocketStore";

const socketStore = useSocketStore();

const route = useRoute();

const roomId = toRef(route.params, "id");

onMounted(() => {
  socketStore.send({
    event_type: "JOIN_ROOM",
    data: { room_id: roomId.value },
  } as JoinRoom);
});

onBeforeUnmount(() => {
  socketStore.send({
    event_type: "LEAVE_ROOM",
    data: { room_id: roomId.value },
  } as LeaveRoom);
});
</script>

<template>
  <div class="room">{{ roomId }}</div>
</template>

<style lang="scss" scoped>
.room {
  width: 100%;
  height: 100%;
}
</style>
