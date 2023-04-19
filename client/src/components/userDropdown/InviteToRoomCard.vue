<script lang="ts" setup>
import { toRefs, onMounted, onBeforeUnmount, computed } from "vue";
import useRoomStore from "../../store/RoomStore";
const props = defineProps<{ id: string }>();

const { id } = toRefs(props);

const roomStore = useRoomStore();

const room = computed(() => roomStore.getRoom(id.value));

onMounted(() => {
  roomStore.cacheRoom(id.value);
  roomStore.roomEnteredView(id.value);
});

onBeforeUnmount(() => {
  roomStore.roomLeftView(id.value);
});
</script>

<template>
  <div ref="containerRef" class="card">
    {{ room?.name }}
  </div>
</template>

<style lang="scss" scoped>
.card {
  display: flex;
  padding: var(--gap-md);
  box-sizing: border-box;
  background-size: cover;
  background-position: center;
  text-align: left;
  text-shadow: 0px 2px 3px black, 0px 1px 7px black;
  font-weight: 600;
  color: white;
  border-radius: var(--border-radius-md);
  border: 1px solid var(--border-light);
  cursor: pointer;
}
</style>
