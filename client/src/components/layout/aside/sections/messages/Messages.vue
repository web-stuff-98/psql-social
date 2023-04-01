<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { makeRequest } from "../../../../../services/makeRequest";
import {
  IDirectMessage,
  IFriendRequest,
  IInvitation,
  IResMsg,
} from "../../../../../interfaces/GeneralInterfaces";
import useInboxStore from "../../../../../store/InboxStore";
import useUserStore from "../../../../../store/UserStore";
import useSocketStore from "../../../../../store/SocketStore";
import User from "../../../../shared/User.vue";
import MessagesItem from "./MessagesItem.vue";
import MessageForm from "../../../../shared/MessageForm.vue";
import { DirectMessage } from "../../../../../socketHandling/OutEvents";

const inboxStore = useInboxStore();
const userStore = useUserStore();
const socketStore = useSocketStore();

const section = ref<"USERS" | "MESSAGES">("USERS");
const resMsg = ref<IResMsg>({});
const currentUid = ref("");

onMounted(async () => {
  try {
    resMsg.value = { msg: "", pen: true, err: false };
    const uids: string[] | null = await makeRequest("/api/acc/uids");
    uids?.forEach((uid) => {
      inboxStore.convs[uid] = [];
      userStore.cacheUser(uid);
    });
    resMsg.value = { msg: "", pen: false, err: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, pen: false, err: true };
  }
});

async function getConversation(uid: string) {
  try {
    resMsg.value = { msg: "", pen: true, err: false };
    const data: {
      friend_requests: IFriendRequest[] | null;
      invitations: IInvitation[] | null;
      direct_messages: IDirectMessage[] | null;
    } | null = await makeRequest(`/api/acc/conv/${uid}`);
    if (data) {
      inboxStore.convs[uid] = [
        ...(data.friend_requests || []),
        ...(data.invitations || []),
        ...(data.direct_messages || []),
      ].sort((a, b) => {
        const dateA = new Date(a.created_at).getTime();
        const dateB = new Date(b.created_at).getTime();
        return dateA - dateB;
      });
    }
    resMsg.value = { msg: "", pen: false, err: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, pen: false, err: true };
  }
}

function handleSubmit(values: any) {
  socketStore.send({
    event_type: "DIRECT_MESSAGE",
    data: {
      content: values.content,
      uid: currentUid.value,
    },
  } as DirectMessage);
}
</script>

<template>
  <div class="messaging-section">
    <div v-if="section === 'MESSAGES'" class="messages-section">
      <div class="messages-section-messages-container">
        <div class="messages-section-messages">
          <MessagesItem
            :item="item"
            v-for="item in inboxStore.convs[currentUid]"
          />
        </div>
      </div>
      <div class="messages-section-bottom-container">
        <MessageForm :handleSubmit="handleSubmit" />
        <button @click="section = 'USERS'" type="button">Back</button>
      </div>
    </div>
    <div v-if="section === 'USERS'" class="users">
      <button
        v-for="uid in Object.keys(inboxStore.convs)"
        @click="
          {
            currentUid = uid;
            section = 'MESSAGES';
            getConversation(uid);
          }
        "
        class="user-container"
      >
        <User :noClick="true" :uid="uid" />
      </button>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.messaging-section {
  border: 2px solid var(--border-pale);
  border-radius: var(--border-radius-sm);
  height: 100%;
  position: relative;
  display: flex;
  flex-direction: column;

  .users,
  .messages-section-messages {
    position: absolute;
    width: 100%;
    height: 100%;
    left: 0;
    top: 0;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
    gap: var(--gap-md);
    padding: var(--gap-md);
  }

  .users {
    .user-container {
      padding: var(--gap-sm);
      border: 1px solid var(--border-light);
      background: var(--border-pale);
      width: 100%;
      display: flex;
      justify-content: flex-start;
    }
  }

  .messages-section {
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 100%;
    .messages-section-messages-container {
      position: relative;
      flex-grow: 1;
      width: 100%;
    }
    .messages-section-bottom-container {
      display: flex;
      flex-direction: column;
      padding: var(--gap-sm);
      gap: var(--gap-sm);
      button {
        padding: 2px;
        font-size: var(--xs);
        border-radius: var(--border-radius-sm);
        width: 100%;
      }
    }
  }
}
</style>
