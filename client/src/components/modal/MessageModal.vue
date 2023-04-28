<script lang="ts" setup>
import ResMsg from "../shared/ResMsg.vue";
import messageModalStore from "../../store/MessageModalStore";
</script>

<template>
  <div v-if="messageModalStore.show" class="modal-backdrop" />
  <div v-if="messageModalStore.show" class="modal-container">
    <div :style="{ paddingTop: 'var(--gap-md)' }" class="modal">
      <ResMsg :resMsg="messageModalStore.msg" />
      <div v-if="!messageModalStore.msg.pen" class="buttons">
        <button
          v-if="messageModalStore.cancellationCallback"
          @click="() => messageModalStore.cancellationCallback!()"
        >
          Cancel
        </button>
        <button @click="() => messageModalStore.confirmationCallback()">
          Confirm
        </button>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.buttons {
  width: 100%;
  gap: var(--gap-md);
  display: flex;
  margin-top: var(--gap-md);
  button {
    flex-grow: 1;
    font-size: var(--sm);
    padding: 3px;
  }
}
* {
  margin: 0;
  text-align: center;
  font-size: var(--md);
}
</style>
