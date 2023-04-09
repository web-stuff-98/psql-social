<script lang="ts" setup>
import { onBeforeUnmount, onMounted, ref } from "vue";
import { bioUid } from "../../store/ViewBioStore";
import { getUserBio } from "../../services/user";
import { IResMsg } from "../../interfaces/GeneralInterfaces";
import { StartWatching, StopWatching } from "../../socketHandling/OutEvents";
import useSocketStore from "../../store/SocketStore";
import Modal from "./Modal.vue";
import User from "../shared/User.vue";
import ResMsg from "../shared/ResMsg.vue";
import ModalCloseButton from "../shared/ModalCloseButton.vue";
import { isChangeEvent } from "../../socketHandling/InterpretEvent";

const socketStore = useSocketStore();

const bio = ref("");
const resMsg = ref<IResMsg>({});

function watchBio(e: MessageEvent) {
  const msg = JSON.parse(e.data);
  if (!msg) return;
  if (isChangeEvent(msg)) {
    if (msg.data.entity === "BIO") {
      if (
        msg.data.change_type === "UPDATE" ||
        msg.data.change_type === "INSERT"
      ) {
        bio.value = (msg.data.data as any)["content"];
      }
      if (msg.data.change_type === "DELETE") {
        bio.value = "";
      }
    }
  }
}

onMounted(async () => {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    const data = await getUserBio(bioUid.value);
    bio.value = data;
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }

  socketStore.send({
    event_type: "START_WATCHING",
    data: {
      entity: "BIO",
      id: bioUid.value,
    },
  } as StartWatching);

  socketStore.socket?.addEventListener("message", watchBio);
});

onBeforeUnmount(() => {
  socketStore.send({
    event_type: "STOP_WATCHING",
    data: {
      entity: "BIO",
      id: bioUid.value,
    },
  } as StopWatching);

  socketStore.socket?.removeEventListener("message", watchBio);
});
</script>

<template>
  <Modal v-if="bioUid">
    <ModalCloseButton @click="bioUid = ''" />
    <div class="bio-container">
      <User :uid="bioUid" />
      <p v-if="bio">
        {{ bio }}
      </p>
      <p v-else>This user has no bio</p>
      <ResMsg :resMsg="resMsg" />
    </div>
  </Modal>
</template>

<style lang="scss" scoped>
.bio-container {
  text-align: center;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: var(--gap-lg);
  max-width: 15rem;
  p {
  padding: var(--gap-md);
    margin: 0;
  }
}
</style>
