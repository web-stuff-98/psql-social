<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { IResMsg, IRoom } from "../../../../interfaces/GeneralInterfaces";
import { getRooms } from "../../../../services/room";
import CreateRoom from "./CreateRoom.vue";

const showCreate = ref(false);
const rooms = ref<IRoom[]>([]);
const resMsg = ref<IResMsg>({});

onMounted(async () => {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    const result = await getRooms();
    rooms.value = result || [];
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
});
</script>

<template>
  <div class="rooms">
    <div class="results">
        {{ rooms }}
    </div>
    <button
      @click="showCreate = true"
      type="button"
      name="create room"
      class="create-button"
    >
      <v-icon name="io-add-circle-sharp" />
      Create
    </button>
  </div>
  <CreateRoom :closeClicked="() => (showCreate = false)" v-if="showCreate" />
</template>

<style lang="scss" scoped>
.rooms {
  border: 2px solid var(--border-pale);
  width: 100%;
  height: 100%;
  position: relative;
  border-radius: var(--border-radius-sm);
  display: flex;
  flex-direction: column;
  .results {
    flex-grow: 1;
    overflow-y: auto;
    border-bottom: 1px solid var(--border-pale);
  }
  .create-button {
    padding: var(--gap-sm);
    background: none;
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-colour);
    text-shadow: none;
    font-weight: 600;
    gap: 3px;
    border-radius: 0;
    svg {
      width: 1.333rem;
      height: 1.333rem;
      fill: var(--text-colour);
    }
  }
  .create-button:hover {
    background: var(--border-pale);
  }
}
</style>
