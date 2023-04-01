<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { IResMsg } from "../../../../../interfaces/GeneralInterfaces";
import { makeRequest } from "../../../../../services/makeRequest";
import User from "../../../../shared/User.vue";
import ResMsg from "../../../../shared/ResMsg.vue";

const friends = ref<string[]>([]);
const resMsg = ref<IResMsg>({});

onMounted(async () => {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    const ids: string[] | null = await makeRequest("/api/acc/friends");
    friends.value = ids || [];
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
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