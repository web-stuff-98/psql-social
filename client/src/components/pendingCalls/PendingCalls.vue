<script lang="ts" setup>
import { CallResponse } from "../../socketHandling/OutEvents";
import { pendingCallsStore } from "../../store/CallsStore";
import useAuthStore from "../../store/AuthStore";
import useSocketStore from "../../store/SocketStore";
import PendingCall from "./PendingCall.vue";

const authStore = useAuthStore();
const socketStore = useSocketStore();

function cancelHangupClicked(index: number) {
  socketStore.send({
    event_type: "CALL_USER_RESPONSE",
    data: {
      caller: pendingCallsStore[index].caller,
      called: pendingCallsStore[index].called,
      accept: false,
    },
  } as CallResponse);
}

function acceptClicked(index: number) {
  socketStore.send({
    event_type: "CALL_USER_RESPONSE",
    data: {
      caller: pendingCallsStore[index].caller,
      called: pendingCallsStore[index].called,
      accept: true,
    },
  } as CallResponse);
}
</script>

<template>
  <div class="container">
    <PendingCall
      :key="pendingCall.caller"
      v-for="(pendingCall, index) in pendingCallsStore"
      :showAcceptDevice="pendingCall.caller !== authStore.uid"
      :cancelHangupClicked="cancelHangupClicked"
      :acceptClicked="acceptClicked"
      :uid="
        pendingCall.caller === authStore.uid
          ? pendingCall.called
          : pendingCall.caller
      "
      :index="index"
    />
  </div>
</template>

<style lang="scss" scoped>
.container {
  position: fixed;
  bottom: 0;
  right: 0;
  display: flex;
  gap: 0.5rem;
  padding: 0.5rem;
}
</style>
