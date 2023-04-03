<script lang="ts" setup>
import { IRoomChannel } from "../../../interfaces/GeneralInterfaces";
import useRoomChannelStore from "../../../store/RoomChannelStore";

const roomChannelStore = useRoomChannelStore();

defineProps<{
  channel: IRoomChannel;
  isAuthor?: boolean;
  joinChannel: (channelId: string) => void;
  editClicked: (channelId: string) => void;
  deleteClicked: (channelId: string) => void;
}>();
</script>

<template>
  <div
    :style="
      roomChannelStore.current === channel.ID ? {} : { filter: 'opacity(0.6)' }
    "
    class="channel"
  >
    <div class="name"># {{ channel.name }}</div>
    <div
      v-if="roomChannelStore.current !== channel.ID || isAuthor"
      class="buttons"
    >
      <button
        v-if="roomChannelStore.current !== channel.ID"
        @click="joinChannel(channel.ID)"
        name="enter room"
        type="button"
      >
        <v-icon name="md-sensordoor-round" />
      </button>
      <button
        v-if="isAuthor"
        @click="editClicked(channel.ID)"
        name="edit channel"
        type="button"
      >
        <v-icon name="md-modeeditoutline" />
      </button>
      <button
        @click="deleteClicked(channel.ID)"
        v-if="isAuthor && !channel.main"
        class="delete-button"
        name="delete channel"
        type="button"
      >
        <v-icon name="md-delete-round" />
      </button>
    </div>
  </div>
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
  border-radius: var(--border-radius-sm);
  .name {
    text-align: left;
    padding: 4px 7px;
  }
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
