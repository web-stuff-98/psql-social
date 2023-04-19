<script lang="ts" setup>
import { onBeforeUnmount, onMounted, ref, toRefs, computed } from "vue";
import { deleteRoom } from "../../../../../services/room";
import useRoomStore from "../../../../../store/RoomStore";
import useAuthStore from "../../../../../store/AuthStore";
import EditRoom from "./EditRoom.vue";
import messageModalStore from "../../../../../store/MessageModalStore";

const props = defineProps<{ rid: string }>();

const { rid } = toRefs(props);

const authStore = useAuthStore();
const roomStore = useRoomStore();

const container = ref<HTMLElement>();
const room = computed(() => roomStore.getRoom(rid.value));

const isEditing = ref(false);

const observer = new IntersectionObserver(([entry]) => {
  if (entry.isIntersecting) {
    roomStore.roomEnteredView(rid.value);
  } else roomStore.roomLeftView(rid.value);
});

onMounted(() => {
  observer.observe(container.value!);
  roomStore.cacheRoomImage(rid.value);
});

onBeforeUnmount(() => {
  observer.disconnect();
});

function deleteRoomClicked() {
  messageModalStore.show = true;
  messageModalStore.msg = {
    msg: "Are you sure you want to delete this room?",
  };
  messageModalStore.cancellationCallback = () =>
    (messageModalStore.show = false);
  messageModalStore.confirmationCallback = () => {
    messageModalStore.msg = { msg: "Deleting...", err: false, pen: true };
    messageModalStore.cancellationCallback = undefined;
    messageModalStore.confirmationCallback = () =>
      (messageModalStore.show = false);
    deleteRoom(rid.value)
      .then(() => (messageModalStore.show = false))
      .catch((e) => {
        messageModalStore.msg = {
          msg: `${e}`,
          err: true,
          pen: false,
        };
        messageModalStore.cancellationCallback = undefined;
        messageModalStore.confirmationCallback = () =>
          (messageModalStore.show = false);
      });
  };
}
</script>

<template>
  <div
    :style="
      room?.img
        ? {
            color: 'white',
            textShadow: '0px 0px 3px black, 0px 2px 6px black',
            fontWeight: 600,
            backgroundImage: `url(${room.img})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
          }
        : {}
    "
    v-if="room"
    ref="container"
    class="room"
  >
    {{ room?.name }}
    <div class="buttons">
      <button
        v-if="authStore.uid === room?.author_id"
        @click="isEditing = true"
        name="edit room"
        type="button"
      >
        <v-icon name="md-modeeditoutline" />
      </button>
      <router-link :to="`/room/${rid}`">
        <button name="enter room" type="button">
          <v-icon name="md-sensordoor-round" />
        </button>
      </router-link>
      <button
        @click="deleteRoomClicked"
        v-if="authStore.uid === room?.author_id"
        class="delete-button"
        name="delete room"
        type="button"
      >
        <v-icon name="md-delete-round" />
      </button>
    </div>
  </div>
  <EditRoom
    v-if="isEditing"
    :closeClicked="() => (isEditing = false)"
    :roomId="room?.ID!"
  />
</template>

<style lang="scss" scoped>
.room {
  border: 1px solid var(--border-heavy);
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
