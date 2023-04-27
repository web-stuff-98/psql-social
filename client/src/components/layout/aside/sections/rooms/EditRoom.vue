<script lang="ts" setup>
import { onMounted, ref, toRefs, computed } from "vue";
import { IResMsg } from "../../../../../interfaces/GeneralInterfaces";
import { updateRoom } from "../../../../../services/room";
import { validateRoomName } from "../../../../../validators/validators";
import { Field, Form } from "vee-validate";
import { uploadRoomImage } from "../../../../../services/room";
import useRoomStore from "../../../../../store/RoomStore";
import ModalCloseButton from "../../../../shared/ModalCloseButton.vue";
import Modal from "../../../../modal/Modal.vue";
import ErrorMessage from "../../../../shared/ErrorMessage.vue";
import CustomCheckbox from "../../../../shared/CustomCheckbox.vue";
import ResMsg from "../../../../shared/ResMsg.vue";

const props = defineProps<{ closeClicked: Function; roomId: string }>();

const { roomId } = toRefs(props);

const roomStore = useRoomStore();

const resMsg = ref<IResMsg>({});

const imgFile = ref<File>();
const imgUrl = ref<string>();
const imgInput = ref<HTMLInputElement>();

const r = computed(() => roomStore.getRoom(roomId.value));

onMounted(() => {
  if (r.value?.img) imgUrl.value = r.value.img;
});

async function handleSubmitEdit(values: any) {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    await updateRoom(roomId.value, values.name, values.isPrivate);
    if (imgFile.value) await uploadRoomImage(r.value?.ID!, imgFile.value!);
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
    <Form
      v-if="!resMsg.pen"
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
      <button type="submit">Update room</button>
      <div class="input-label">
        <label for="private">Private </label>
        <CustomCheckbox id="private" name="isPrivate" />
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
  margin-top: var(--gap-lg);
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
