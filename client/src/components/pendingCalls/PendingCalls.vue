<script lang="ts" setup>
import { CallResponse } from "../../socketHandling/OutEvents";
import useCallStore from "../../store/CallsStore";
import useAuthStore from "../../store/AuthStore";
import useSocketStore from "../../store/SocketStore";
import PendingCall from "./PendingCall.vue";

const authStore = useAuthStore();
const socketStore = useSocketStore();
const callStore = useCallStore();

const cancelHangupClicked = (index: number) =>
  socketStore.send({
    event_type: "CALL_USER_RESPONSE",
    data: {
      caller: callStore.calls[index].caller,
      called: callStore.calls[index].called,
      accept: false,
    },
  } as CallResponse);

const acceptClicked = (index: number) =>
  socketStore.send({
    event_type: "CALL_USER_RESPONSE",
    data: {
      caller: callStore.calls[index].caller,
      called: callStore.calls[index].called,
      accept: true,
    },
  } as CallResponse);
</script>

<template>
  <div class="pending-calls-container">
    <PendingCall
      :key="index"
      :showAcceptDevice="pendingCall.caller !== authStore.uid"
      :cancelHangupClicked="cancelHangupClicked"
      :acceptClicked="acceptClicked"
      :uid="
        pendingCall.caller === authStore.uid
          ? pendingCall.called
          : pendingCall.caller
      "
      :index="index"
      v-for="(pendingCall, index) in callStore.calls"
    />
  </div>
</template>

<style lang="scss" scoped>
.pending-calls-container {
  position: fixed;
  bottom: 0;
  right: 0;
  display: flex;
  gap: 0.5rem;
  padding: 0.5rem;
}
</style>
