<script lang="ts" setup>
import { IRoomMessage } from "../../interfaces/GeneralInterfaces";
import User from "./User.vue";
import useSocketStore from "../../store/SocketStore";
import {
  RoomMessageDelete,
  RoomMessageUpdate,
} from "../../socketHandling/OutEvents";
import { validateMessage } from "../../validators/validators";
import { ref, toRefs } from "vue";
import { Field, Form } from "vee-validate";
import ErrorMessage from "./ErrorMessage.vue";
const props = defineProps<{ msg: IRoomMessage }>();

const socketStore = useSocketStore();

const { msg } = toRefs(props);
const isEditing = ref(false);
const inputRef = ref<HTMLInputElement>();

function editClicked() {
  isEditing.value = true;
  //@ts-ignore
  inputRef.value.value = msg.value.content;
}

function deleteClicked() {
  socketStore.send({
    event_type: "ROOM_MESSAGE_DELETE",
    data: {
      msg_id: msg.value.ID,
    },
  } as RoomMessageDelete);
}

function handleSubmitEdit(values: any) {
  socketStore.send({
    event_type: "ROOM_MESSAGE_UPDATE",
    data: {
      msg_id: msg.value.ID,
      content: values.content as string,
    },
  } as RoomMessageUpdate);
  isEditing.value = false;
}
</script>

<template>
  <div class="msg-container">
    <div class="user">
      <User :uid="msg.author_id" />
    </div>
    <div v-show="!isEditing" class="content">
      {{ msg.content }}
    </div>
    <Form @submit="handleSubmitEdit" v-show="isEditing">
      <div class="field-error">
        <Field
          ref="inputRef"
          as="textarea"
          type="textarea"
          name="content"
          :rules="validateMessage as any"
        />
        <ErrorMessage name="content" />
      </div>
      <div class="buttons">
        <button type="submit" name="submit edit">
          <v-icon name="io-send" />
        </button>
        <button @click="isEditing = false" type="button" name="submit edit">
          <v-icon name="io-close" />
        </button>
      </div>
    </Form>
    <div v-show="!isEditing" class="buttons">
      <button @click="editClicked()" name="edit message" type="button">
        <v-icon name="md-modeeditoutline" />
      </button>
      <button
        @click="deleteClicked()"
        class="delete-button"
        name="delete message"
        type="button"
      >
        <v-icon name="md-delete-round" />
      </button>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.msg-container {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  gap: var(--gap-md);
  .content {
    font-size: var(--xs);
    flex-grow: 1;
  }
  .buttons {
    display: flex;
    align-items: center;
    flex-direction: column;
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

  form {
    display: flex;
    align-items: center;
    width: 100%;
    gap: var(--gap-sm);
    .field-error {
      textarea {
        width: 100%;
        flex-grow: 1;
      }
      width: 100%;
      display: flex;
      flex-direction: column;
    }
  }
}
</style>