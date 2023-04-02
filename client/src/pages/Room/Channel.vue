<script lang="ts" setup>
import useAuthStore from "../../store/AuthStore";
import { IRoomChannel } from "../../interfaces/GeneralInterfaces";

const authStore = useAuthStore();

defineProps<{
  channel: IRoomChannel;
  isAuthor?: boolean;
  editClicked: (channelId: string) => void;
}>();
</script>

<template>
  <button class="channel">
    # {{ channel.name }}
    <div v-if="isAuthor" class="buttons">
      <button @click="editClicked(channel.ID)" name="edit room" type="button">
        <v-icon name="md-modeeditoutline" />
      </button>
    </div>
  </button>
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
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-left:7px;
  .buttons {
    display: flex;
    align-items: center;
    gap: 1px;
    padding: 1px;
    border: 1px solid var(--border-medium);
    background: rgba(0, 0, 0, 0.25);
    border-radius: var(--border-radius-sm);
    margin-left: 2px;
    button {
      padding: 1px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: var(--base-colour);
      svg {
        width: var(--sm);
        height: var(--sm);
      }
    }
    .delete-button {
      background: red;
    }
  }
}
</style>
