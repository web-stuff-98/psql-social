<script lang="ts" setup>
import { nextTick, ref, toRefs } from "vue";
import { Field, Form } from "vee-validate";
import messageModalStore from "../../store/MessageModalStore";

const props = defineProps<{
  handleSubmit: (values: any, file?: File) => void;
}>();
const { handleSubmit } = toRefs(props);

const attachmentFile = ref<File>();
const inputRef = ref<HTMLElement>();
const attachmentInputRef = ref<HTMLElement>();
const emojiMenuOpen = ref(false);
const emojiMenu = ref<HTMLElement>();
const message = ref("");
const emojis = ref(
  `😀 😃 😄 😁 😆 😅 😂 🤣 🥲 🥹 ☺️ 😊 😇 🙂 🙃 😉 😌 😍 🥰 😘 😗 😙 😚 😋 😛 😝 😜 🤪 🤨 🧐 🤓 😎 🥸 🤩 🥳 😏 😒 😞 😔 😟 😕 🙁 ☹️ 😣 😖 😫 😩 🥺 😢 😭 😮‍💨 😤 😠 😡 🤬 🤯 😳 🥵 🥶 😱 😨 😰 😥 😓 🫣 🤗 🫡 🤔 🫢 🤭 🤫 🤥 😶 😶‍🌫️ 😐 😑 😬 🫨 🫠 🙄 😯 😦 😧 😮 😲 🥱 😴 🤤 😪 😵 😵‍💫 🫥 🤐 🥴 🤢 🤮 🤧 😷 🤒 🤕 🤑 🤠 😈 👿 👹 👺 🤡 💩 👻 💀 ☠️ 👽 👾 🤖 🎃 😺 😸 😹 😻 😼 😽 🙀 😿 😾`
);

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

const toggleEmojiMenu = async () => {
  emojiMenuOpen.value = !emojiMenuOpen.value;
  await nextTick(() => {
    if (emojiMenu.value)
      emojiMenu.value.style.top = `-${emojiMenu.value?.clientHeight}px`;
  });
}

const addEmoji = (emoji: string) => {
  message.value = `${message.value}${emoji}`;
  emojiMenuOpen.value = false;
}
</script>

<template>
  <Form @submit="submit">
    <Field v-model="message" ref="inputRef" name="message" />
    <button type="submit">
      <v-icon name="io-send" />
    </button>
    <button @click="attachmentInputRef?.click()" type="button">
      <v-icon
        :style="attachmentFile ? { fill: 'green', color: 'green' } : {}"
        name="bi-paperclip"
      />
    </button>
    <button @click="toggleEmojiMenu" class="emoji-button" type="button">
      🙂
      <div ref="emojiMenu" v-if="emojiMenuOpen" class="emoji-menu">
        <button
          @click="() => addEmoji(emoji)"
          type="button"
          v-for="emoji in emojis.split(` `)"
        >
          {{ emoji }}
        </button>
      </div>
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
  position: relative;
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
  .emoji-button {
    .emoji-menu {
      right: 0;
      background: var(--base-colour);
      box-shadow: 0px 2px 3px rgba(0, 0, 0, 0.166);
      border: 1px solid var(--border-medium);
      border-radius: var(--border-radius-md);
      gap: var(--gap-sm);
      padding: var(--gap-sm);
      display: flex;
      justify-content: flex-start;
      align-items: flex-start;
      flex-wrap: wrap;
      position: absolute;
      width: 10rem;
    }
  }
}
</style>
