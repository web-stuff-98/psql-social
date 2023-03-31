<script lang="ts" setup>
import { IRoomChannel } from "../../interfaces/GeneralInterfaces";
defineProps<{ channel: IRoomChannel }>();
</script>

<template>
  <button class="channel"># {{ channel.name }}</button>
</template>

<style lang="scss" scoped>
.channel {
  border: 1px solid var(--border-medium);
  padding: 3px;
  font-size: var(--xs);
  width: 100%;
  background: var(--base-colour);
  color: black;
  text-shadow: none;
  box-shadow: 0px 2px 3px rgba(0, 0, 0, 0.166);
}
</style>
