<script lang="ts" setup>
import ModalCloseButton from "../../../../shared/ModalCloseButton.vue";
import { ref, toRefs } from "vue";
import { IResMsg } from "../../../../../interfaces/GeneralInterfaces";
import { updateRoom } from "../../../../../services/room";
import { validateRoomName } from "../../../../../validators/validators";
import { Field, Form } from "vee-validate";
import useRoomStore from "../../../../../store/RoomStore";
import Modal from "../../../../modal/Modal.vue";
import ErrorMessage from "../../../../shared/ErrorMessage.vue";
import CustomCheckbox from "../../../../shared/CustomCheckbox.vue";
import ResMsg from "../../../../shared/ResMsg.vue";

const props = defineProps<{ closeClicked: Function; roomId: string }>();

const { roomId } = toRefs(props);

const roomStore = useRoomStore();

const resMsg = ref<IResMsg>({});

// used for initial values
const r = roomStore.getRoom(roomId.value);

async function handleSubmitEdit(values: any) {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    await updateRoom({
      name: values.name,
      isPrivate: values.isPrivate,
      id: roomId.value,
    });
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
}
</script>

<template>
  <Modal>
    <ModalCloseButton @click="closeClicked()" />
    <Form
      :initialValues="{ name:r!.name, isPrivate:r!.is_private }"
      @submit="handleSubmitEdit"
    >
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
      <button type="submit">Update room</button>
      <div class="input-label">
        <label for="private">Private </label>
        <CustomCheckbox id="private" name="isPrivate" />
        <ErrorMessage name="isPrivate" />
      </div>
      <ResMsg :resMsg="resMsg" />
    </Form>
  </Modal>
</template>

<style lang="scss" scoped>
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
