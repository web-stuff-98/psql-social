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
  </div>
</template>

<style lang="scss" scoped>
.room {
  border: 1px solid var(--border-pale);
  border-radius: var(--border-radius-sm);
  padding: 3px;
  font-size: var(--xs);
  font-weight: 600;
  box-shadow: 0px 2px 2px rgba(0, 0, 0, 0.12);
}
</style>
