<script lang="ts" setup>
import { onBeforeUnmount, onMounted, ref } from "vue";
import { makeRequest } from "../../../../services/makeRequest";
import {
  IDirectMessage,
  IFriendRequest,
  IInvitation,
  IResMsg,
} from "../../../../interfaces/GeneralInterfaces";
import useInboxStore from "../../../../store/InboxStore";
import User from "../../../shared/User.vue";

const inboxStore = useInboxStore();

const section = ref<"USERS" | "MESSAGES">("USERS");
const resMsg = ref<IResMsg>({});
const currentUid = ref("");

onMounted(async () => {
  try {
    resMsg.value = { msg: "", pen: true, err: false };
    const uids: string[] | null = await makeRequest("/api/acc/uids");
    uids?.forEach((uid) => (inboxStore.convs[uid] = []));
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
</script>

<template>
  <div class="messaging-section">
    <div v-if="section === 'MESSAGES'" class="messages-section">
      <div class="messages-section-messages-container">
        <div class="messages-section-messages">
          {{ inboxStore.convs[currentUid] }}
        </div>
      </div>
      <div class="messages-section-back-button-container">
        <button @click="section = 'USERS'" type="button">Back</button>
      </div>
    </div>
    <div v-if="section === 'USERS'" class="users">
      <User
        @click="
          {
            currentUid = uid;
            section = 'MESSAGES';
            getConversation(uid);
          }
        "
        :uid="uid"
        v-for="uid in Object.keys(inboxStore.convs)"
      />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.messaging-section {
  border: 2px solid var(--border-pale);
  border-radius: var(--border-radius-sm);
  height: 100%;
  position: relative;

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

  .messages-section {
    display: flex;
    flex-direction: column;
    .messages-section-messages-container {
      position: relative;
      flex-grow: 1;
      width: 100%;
    }
    .messages-section-back-button-container {
      padding: var(--gap-sm);
      button {
        padding: 2px;
        font-size: var(--xs);
      }
    }
  }
}
</style>
