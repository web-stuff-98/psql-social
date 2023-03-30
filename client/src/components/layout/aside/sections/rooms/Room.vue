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
}
</style>
