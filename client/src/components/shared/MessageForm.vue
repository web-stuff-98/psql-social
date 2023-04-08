<script lang="ts" setup>
import { ref, toRefs } from "vue";
import { Field, Form } from "vee-validate";
import messageModalStore from "../../store/MessageModalStore";

const props = defineProps<{
  handleSubmit: (values: any, file?: File) => void;
}>();
const { handleSubmit } = toRefs(props);

const attachmentFile = ref<File>();
const inputRef = ref<HTMLElement>();
const attachmentInputRef = ref<HTMLElement>();

const submit = (values: any) => {
  handleSubmit.value(values, attachmentFile.value);
  attachmentFile.value = undefined;
  //@ts-ignore
  inputRef.value.reset();
};

function selectAttachment(e: Event) {
  const target = e.target as HTMLInputElement;
  if (!target.files || !target.files[0]) {
    attachmentFile.value = undefined;
    return;
  }
  if (target.files[0].size > 30 * 1024 * 1024) {
    messageModalStore.msg = {
      msg: "File too large. Max 30mb.",
      err: true,
      pen: false,
    };
    messageModalStore.show = true;
    messageModalStore.cancellationCallback = undefined;
    messageModalStore.confirmationCallback = () =>
      (messageModalStore.show = false);
    return;
  }
  attachmentFile.value = target.files[0];
}
</script>

<template>
  <Form @submit="submit">
    <Field ref="inputRef" name="message" />
    <button type="submit">
      <v-icon name="io-send" />
    </button>
    <button @click="attachmentInputRef?.click()" type="button">
      <v-icon
        :style="attachmentFile ? { fill: 'green', color: 'green' } : {}"
        name="bi-paperclip"
      />
    </button>
    <input @change="selectAttachment" ref="attachmentInputRef" type="file" />
  </Form>
</template>

<style lang="scss" scoped>
form {
  width: 100%;
  gap: var(--gap-sm);
  margin: 0;
  display: flex;
  align-items: center;
  input {
    width: 100%;
    height: 100%;
    padding: 3px;
  }
  button {
    background: none;
    border: none;
    padding: 0;
  }
  button:hover {
    background: none;
  }
  button:first-of-type {
    padding-right: 0;
  }
  button:last-of-type {
    padding: 0;
    svg {
      width: 1.5rem;
      height: 1.5rem;
    }
  }
}
</style>
