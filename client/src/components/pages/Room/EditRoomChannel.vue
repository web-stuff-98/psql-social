<script lang="ts" setup>
import { toRefs, ref } from "vue";
import ModalCloseButton from "../../../components/shared/ModalCloseButton.vue";
import Modal from "../../../components/modal/Modal.vue";
import useRoomChannelStore from "../../../store/RoomChannelStore";
import { validateChannelName } from "../../../validators/validators";
import ErrorMessage from "../../../components/shared/ErrorMessage.vue";
import { Field, Form } from "vee-validate";
import CustomCheckbox from "../../../components/shared/CustomCheckbox.vue";
import { IResMsg } from "../../../interfaces/GeneralInterfaces";
import { updateRoomChannel } from "../../../services/room";

const roomChannelStore = useRoomChannelStore();

const props = defineProps<{ closeClicked: Function; channelId: string }>();
const { channelId } = toRefs(props);

// used for initial values
const ch = roomChannelStore.channels.find((c) => c.ID === channelId.value);

const resMsg = ref<IResMsg>({});

async function handleSubmit(values: any) {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    await updateRoomChannel({
      name: values.name,
      main: values.main,
      id: channelId.value,
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
      @submit="handleSubmit"
      :initial-values="{name:ch!.name, main:ch!.main}"
    >
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
      <button type="submit">Submit update</button>
      <div class="input-label">
        <label for="main">Set main (unchecked ignored)</label>
        <CustomCheckbox id="main" name="main" />
        <ErrorMessage name="main" />
      </div>
    </Form>
  </Modal>
</template>

<style lang="scss" scoped>
form {
  gap: var(--gap-md);
  display: flex;
  flex-direction: column;
  button {
    width: 100%;
  }
}
</style>
