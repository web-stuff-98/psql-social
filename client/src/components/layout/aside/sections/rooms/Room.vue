<script lang="ts" setup>
import { Field, Form } from "vee-validate";
import { computed, onBeforeUnmount, onMounted, ref, toRefs } from "vue";
import { IResMsg } from "../../../../../interfaces/GeneralInterfaces";
import { validateRoomName } from "../../../../../validators/validators";
import { updateRoom } from "../../../../../services/room";
import useRoomStore from "../../../../../store/RoomStore";
import Modal from "../../../../modal/Modal.vue";
import CustomCheckbox from "../../../../shared/CustomCheckbox.vue";
import ErrorMessage from "../../../../shared/ErrorMessage.vue";
import ModalCloseButton from "../../../../shared/ModalCloseButton.vue";
import ResMsg from "../../../../shared/ResMsg.vue";

const props = defineProps<{ rid: string }>();

const { rid } = toRefs(props);

const roomStore = useRoomStore();

const container = ref<HTMLElement>();
const room = computed(() => roomStore.getRoom(rid.value));

const isEditing = ref(false);
const editResMsg = ref<IResMsg>({});

async function handleSubmitEdit(values: any) {
  try {
    editResMsg.value = { msg: "", err: false, pen: true };
    await updateRoom({
      name: values.name,
      isPrivate: values.isPrivate,
      id: rid.value,
    });
    editResMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    editResMsg.value = { msg: `${e}`, err: true, pen: false };
  }
}

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
      <button @click="isEditing = true" name="edit room" type="button">
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
  <Modal v-if="isEditing">
    <ModalCloseButton @click="isEditing = false" />
    <Form @submit="handleSubmitEdit">
      <div class="input-label">
        <label for="name">Name</label>
        <Field
          :rules="validateRoomName as any"
          type="text"
          name="name"
          id="name"
        />
        <ErrorMessage name="name" />
      </div>
      <div class="input-label">
        <label for="private">Private</label>
        <CustomCheckbox name="isPrivate" />
        <ErrorMessage name="isPrivate" />
      </div>
      <button type="submit">Update room</button>
      <ResMsg :resMsg="editResMsg" />
    </Form>
  </Modal>
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
form {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--gap-md);
  button[type="submit"] {
    width: 100%;
  }
}
</style>
