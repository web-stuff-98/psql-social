<script lang="ts" setup>
import { ref } from "vue";
import { Form, Field } from "vee-validate";
import { createRoom, uploadRoomImage } from "../../../../../services/room";
import { IResMsg } from "../../../../../interfaces/GeneralInterfaces";
import { validateRoomName } from "../../../../../validators/validators";
import Modal from "../../../../modal/Modal.vue";
import ModalCloseButton from "../../../../shared/ModalCloseButton.vue";
import ResMsg from "../../../../shared/ResMsg.vue";
import CustomCheckbox from "../../../../shared/CustomCheckbox.vue";
import ErrorMessage from "../../../../shared/ErrorMessage.vue";

defineProps<{ closeClicked: Function }>();

const resMsg = ref<IResMsg>({});

const imgFile = ref<File>();
const imgUrl = ref<string>();
const imgInput = ref<HTMLInputElement>();

async function handleSubmit(values: any) {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    const id = await createRoom(values.name, values.isPrivate);
    if (imgFile.value) uploadRoomImage(id, imgFile.value);
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
}

function selectImage(e: Event) {
  const target = e.target as HTMLInputElement;
  if (!target.files || !target.files[0]) return;
  if (imgUrl.value && imgFile.value) URL.revokeObjectURL(imgUrl.value);
  const file = target.files[0];
  imgFile.value = file;
  imgUrl.value = URL.createObjectURL(file);
}
</script>

<template>
  <Modal>
    <ModalCloseButton @click="closeClicked()" />
    <Form v-if="!resMsg.pen" @submit="handleSubmit">
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
      <button @click="imgInput?.click()" id="select image" type="button">
        Select image
      </button>
      <!-- Hidden file input -->
      <input
        accept=".png,.jpeg.jpg"
        @change="selectImage"
        ref="imgInput"
        type="file"
      />
      <button type="submit">Create room</button>
      <div class="input-label">
        <label for="private">Private</label>
        <CustomCheckbox name="isPrivate" />
        <ErrorMessage name="isPrivate" />
      </div>
      <img v-if="imgUrl" :src="imgUrl" />
    </Form>
    <ResMsg :resMsg="resMsg" />
  </Modal>
</template>

<style lang="scss" scoped>
form {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--gap-md);
  button {
    width: 100%;
  }
  img {
    border-radius: var(--border-radius-md);
    border: 1px solid var(--border-medium);
    box-shadow: 0px 2px 3px rgba(0, 0, 0, 0.1666);
    max-width: 8rem;
  }
}
</style>
