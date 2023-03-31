<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref, toRefs } from "vue";
import useRoomStore from "../../../../../store/RoomStore";

const props = defineProps<{ rid: string }>();

const { rid } = toRefs(props);

const roomStore = useRoomStore();

const container = ref<HTMLElement>();
const room = computed(() => roomStore.getRoom(rid.value));

const observer = new IntersectionObserver(([entry]) => {
  if (entry.isIntersecting) roomStore.roomEnteredView(rid.value);
  else roomStore.roomLeftView(rid.value);
});

onMounted(() => {
  observer.observe(container.value!);
});

onBeforeUnmount(() => {
  observer.disconnect();
});
</script>

<template>
  <div ref="container" class="room">
    {{ room?.name }}
    <div class="buttons">
      <button name="edit room" type="button">
        <v-icon name="md-modeeditoutline" />
      </button>
      <router-link :to="`/room/${rid}`">
        <button name="enter room" type="button">
          <v-icon name="md-sensordoor-round" />
        </button>
      </router-link>
      <button class="delete-button" name="delete room" type="button">
        <v-icon name="md-delete-round" />
      </button>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.room {
  border: 1px solid var(--border-pale);
  border-radius: var(--border-radius-sm);
  padding: 1px;
  padding-left: 6px;
  font-size: var(--xs);
  font-weight: 600;
  box-shadow: 0px 2px 2px rgba(0, 0, 0, 0.12);
  display: flex;
  align-items: center;
  justify-content: space-between;
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
