<script lang="ts" setup>
import { toRefs, ref } from "vue";
import ModalCloseButton from "../../components/shared/ModalCloseButton.vue";
import Modal from "../../components/modal/Modal.vue";
import { validateChannelName } from "../../validators/validators";
import ErrorMessage from "../../components/shared/ErrorMessage.vue";
import { Field, Form } from "vee-validate";
import CustomCheckbox from "../../components/shared/CustomCheckbox.vue";
import { IResMsg } from "../../interfaces/GeneralInterfaces";
import { createRoomChannel } from "../../services/room";

const props = defineProps<{ closeClicked: Function; roomId: string }>();
const { roomId } = toRefs(props);

const resMsg = ref<IResMsg>({});

async function handleSubmit(values: any) {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    await createRoomChannel({
      name: values.name,
      main: values.main,
      roomId: roomId.value,
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
    <div class="edit-channel-container">
      <Form @submit="handleSubmit">
        <div class="input-label">
          <label for="main">Set to new main channel</label>
          <CustomCheckbox id="main" name="main" />
          <ErrorMessage name="main" />
        </div>
        <div class="input-label">
          <label for="name">Channel name</label>
          <Field
            type="text"
            name="name"
            id="name"
            :rules="validateChannelName as any"
          />
          <ErrorMessage name="name" />
        </div>
        <button type="submit">Create channel</button>
      </Form>
    </div>
  </Modal>
</template>

<style lang="scss" scoped>
.edit-channel-container {
  form {
    gap: var(--gap-md);
    display: flex;
    flex-direction: column;
    button {
      width: 100%;
    }
  }
}
</style>
