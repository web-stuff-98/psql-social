<script lang="ts" setup>
import { onBeforeUnmount, onMounted, ref } from "vue";
import { IResMsg } from "../../../../../interfaces/GeneralInterfaces";
import User from "../../../../shared/User.vue";
import ResMsg from "../../../../shared/ResMsg.vue";
import {
  isBlock,
  isChangeEvent,
} from "../../../../../socketHandling/InterpretEvent";
import useAuthStore from "../../../../../store/AuthStore";
import useSocketStore from "../../../../../store/SocketStore";
import useUserStore from "../../../../../store/UserStore";
import { UnBlock } from "../../../../../socketHandling/OutEvents";
import { getBlockedUids } from "../../../../../services/account";

const authStore = useAuthStore();
const socketStore = useSocketStore();
const userStore = useUserStore();

const blocked = ref<string[]>([]);
const resMsg = ref<IResMsg>({});

function watchForBlocksAndDeletes(e: MessageEvent) {
  const msg = JSON.parse(e.data);
  if (!msg) return;
  if (isBlock(msg))
    if (msg.data.blocker === authStore.uid)
      blocked.value.push(msg.data.blocked);
  if (isChangeEvent(msg)) {
    if (msg.data.entity === "USER") {
      const i = blocked.value.findIndex((b) => b === msg.data.data.ID);
      if (i !== -1) blocked.value.splice(i, 1);
    }
  }
}

function unblock(uid: string) {
  socketStore.send({
    event_type: "UNBLOCK",
    data: { uid },
  } as UnBlock);
  const i = blocked.value.findIndex((b) => b === uid);
  if (i !== -1) blocked.value.splice(i, 1);
}

onMounted(async () => {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    const ids: string[] | null = await getBlockedUids();
    // remove duplicates that somehow magically end up in the array
    blocked.value =
      ids?.filter((item, index) => ids.indexOf(item) === index) || [];
    if (ids) ids.forEach((id) => userStore.cacheUser(id));
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }

  socketStore.socket?.addEventListener("message", watchForBlocksAndDeletes);
});

onBeforeUnmount(() => {
  socketStore.socket?.removeEventListener("message", watchForBlocksAndDeletes);
});
</script>

<template>
  <div class="blocked-section">
    <ResMsg :resMsg="resMsg" />
    <div class="list">
      <div v-for="uid in blocked" class="item">
        <User :noPfp="true" :uid="uid" />
        <button @click="unblock(uid)" type="button">Unblock</button>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.blocked-section {
  border: 2px solid var(--border-light);
  border-radius: var(--border-radius-sm);
  height: 100%;
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  .list {
    position: absolute;
    width: 100%;
    height: 100%;
    left: 0;
    top: 0;
    overflow-y: auto;
    padding: var(--gap-md);
    gap: var(--gap-md);
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    justify-content: flex-start;
    .item {
      display: flex;
      justify-content: space-between;
      width: 100%;
      gap: 4px;
      align-items: center;
      button {
        padding: 3px;
        margin: 0;
        font-size: var(--xs);
      }
    }
  }
}
</style>
