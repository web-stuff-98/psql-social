<script lang="ts" setup>
import { onBeforeUnmount, onMounted, ref } from "vue";
import { IResMsg } from "../../../../../interfaces/GeneralInterfaces";
import { getFriendsUids } from "../../../../../services/account"
import User from "../../../../shared/User.vue";
import ResMsg from "../../../../shared/ResMsg.vue";
import {
  isBlock,
  isChangeEvent,
} from "../../../../../socketHandling/InterpretEvent";
import useAuthStore from "../../../../../store/AuthStore";
import useSocketStore from "../../../../../store/SocketStore";
import useUserStore from "../../../../../store/UserStore";

const authStore = useAuthStore();
const socketStore = useSocketStore();
const userStore = useUserStore();

const friends = ref<string[]>([]);
const resMsg = ref<IResMsg>({});

function watchForBlocksAndDeletes(e: MessageEvent) {
  const msg = JSON.parse(e.data);
  if (!msg) return;
  if (isBlock(msg)) {
    const otherUser =
      msg.data.blocker === authStore.uid ? msg.data.blocked : msg.data.blocker;
    const i = friends.value.findIndex((f) => f === otherUser);
    if (i !== -1) friends.value.splice(i, 1);
  }
  if (isChangeEvent(msg)) {
    if (msg.data.entity === "USER") {
      const i = friends.value.findIndex((f) => f === msg.data.data.ID);
      if (i !== -1) friends.value.splice(i, 1);
    }
  }
}

onMounted(async () => {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    const ids: string[] | null = await getFriendsUids();
    // remove duplicates that somehow magically end up in the array
    friends.value =
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
  <div class="friends-section">
    <ResMsg :resMsg="resMsg" />
    <div class="list">
      <User :uid="uid" v-for="uid in friends" />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.friends-section {
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
  }
}
</style>
