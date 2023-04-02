<script lang="ts" setup>
import {
  IRoomMessage,
  IDirectMessage,
} from "../../interfaces/GeneralInterfaces";
import { validateMessage } from "../../validators/validators";
import { ref, toRefs } from "vue";
import { Field, Form } from "vee-validate";
import User from "./User.vue";
import useSocketStore from "../../store/SocketStore";
import {
  DirectMessageDelete,
  DirectMessageUpdate,
  RoomMessageDelete,
  RoomMessageUpdate,
} from "../../socketHandling/OutEvents";
import ErrorMessage from "./ErrorMessage.vue";
const props = defineProps<{
  msg: IRoomMessage | IDirectMessage;
  roomId?: string;
  isAuthor?: boolean;
}>();

const socketStore = useSocketStore();
const { msg, roomId } = toRefs(props);
const isEditing = ref(false);
const inputRef = ref<HTMLInputElement>();

function editClicked() {
  isEditing.value = true;
  //@ts-ignore
  inputRef.value = msg.value.content;
}

function deleteClicked() {
  if (roomId!.value)
    socketStore.send({
      event_type: "ROOM_MESSAGE_DELETE",
      data: {
        msg_id: msg.value.ID,
      },
    } as RoomMessageDelete);
  else
    socketStore.send({
      event_type: "DIRECT_MESSAGE_DELETE",
      data: {
        msg_id: msg.value.ID,
      },
    } as DirectMessageDelete);
}

function handleSubmitEdit(values: any) {
  if (roomId!.value)
    socketStore.send({
      event_type: "ROOM_MESSAGE_UPDATE",
      data: {
        msg_id: msg.value.ID,
        content: values.content as string,
      },
    } as RoomMessageUpdate);
  else
    socketStore.send({
      event_type: "DIRECT_MESSAGE_UPDATE",
      data: {
        msg_id: msg.value.ID,
        content: values.content as string,
      },
    } as DirectMessageUpdate);
  isEditing.value = false;
}
</script>

<template>
  <div
    :style="
      isAuthor ? {} : { flexDirection: 'row-reverse', textAlign: 'right' }
    "
    class="msg-container"
  >
    <div
      :style="{
        ...(isAuthor
          ? {}
          : { flexDirection: 'row-reverse', textAlign: 'right' }),
        ...(roomId
          ? {
              flexDirection: 'column',
              alignItems: isAuthor ? 'flex-start' : 'flex-end',
              gap:'6px'
            }
          : {}),
      }"
      class="user-content"
    >
      <div class="user">
        <User
          :roomId="roomId"
          :date="msg.created_at"
          :reverse="!isAuthor"
          :uid="msg.author_id"
        />
      </div>
      <div v-show="!isEditing" class="content">
        {{ msg.content }}
      </div>
    </div>
    <Form
      :initial-values="{ content: msg.content }"
      @submit="handleSubmitEdit"
      v-show="isEditing"
    >
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
    <div v-show="!isEditing && isAuthor" class="buttons">
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
  .user-content {
    display: flex;
    align-items: center;
    width: 100%;
    .content {
      font-size: var(--xs);
      flex-grow: 1;
    }
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
