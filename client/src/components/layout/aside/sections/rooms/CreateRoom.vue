<script lang="ts" setup>
import { ref } from "vue";
import { Form, Field } from "vee-validate";
import { createRoom } from "../../../../../services/room";
import { IResMsg } from "../../../../../interfaces/GeneralInterfaces";
import { validateRoomName } from "../../../../../validators/validators";
import Modal from "../../../../modal/Modal.vue";
import ModalCloseButton from "../../../../shared/ModalCloseButton.vue";
import ResMsg from "../../../../shared/ResMsg.vue";
import CustomCheckbox from "../../../../shared/CustomCheckbox.vue";
import ErrorMessage from "../../../../shared/ErrorMessage.vue";

defineProps<{ closeClicked: Function }>();

const resMsg = ref<IResMsg>({});

async function handleSubmit(values: any) {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    console.log(values);
    await createRoom({ name: values.name, isPrivate: values.isPrivate });
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
}

/* This is a subsection of rooms */
</script>

<template>
  <Modal>
    <ModalCloseButton @click="closeClicked()" />
    <Form @submit="handleSubmit">
      <div class="input-label">
        <label for="private">Private</label>
        <CustomCheckbox name="isPrivate" />
        <ErrorMessage name="isPrivate" />
      </div>
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
      <button type="submit">Create room</button>
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
