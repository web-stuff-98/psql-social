<script lang="ts" setup>
import { IRoomMessage } from "../../interfaces/GeneralInterfaces";
import User from "../shared/User.vue";
defineProps<{ msg: IRoomMessage }>();
</script>

<template>
  <div class="msg-container">
    <div class="user">
      <User :uid="msg.author_id" />
    </div>
    <div class="content">
      {{ msg.content }}
    </div>
  </div>
</template>

<style lang="scss" scoped>
.msg-container {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  gap: var(--gap-md);
  .content {
    font-size: var(--xs);
    flex-grow: 1;
  }
}
</style>
