<script lang="ts" setup>
import { IResMsg } from "../../interfaces/GeneralInterfaces";
defineProps<{ resMsg?: IResMsg }>();
</script>

<template>
  <div v-if="resMsg?.msg || resMsg?.err || resMsg?.pen" class="res-msg">
    <v-icon v-if="resMsg?.pen" class="spinner spin" name="pr-spinner" />
    <v-icon v-if="resMsg?.err" class="error" name="md-error-round" />
    <div
      v-if="resMsg?.msg"
      :style="resMsg?.err ? { color: 'red' } : {}"
      class="msg"
    >
      {{ resMsg?.msg }}
    </div>
  </div>
</template>

<style lang="scss" scoped>
.res-msg {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: var(--gap-md);
  .msg {
    font-weight: 600;
  }
  filter: drop-shadow(0px 3px 3px rgba(0, 0, 0, 0.25));
  svg {
    width: 2.5rem;
    height: 2.5rem;
  }
  .error {
    color: red;
  }
}
</style>
