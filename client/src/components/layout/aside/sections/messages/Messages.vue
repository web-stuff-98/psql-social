<script lang="ts" setup>
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { IResMsg } from "../../../../../interfaces/GeneralInterfaces";
import useInboxStore from "../../../../../store/InboxStore";
import useUserStore from "../../../../../store/UserStore";
import useSocketStore from "../../../../../store/SocketStore";
import User from "../../../../shared/User.vue";
import MessagesItem from "./MessagesItem.vue";
import MessageForm from "../../../../shared/MessageForm.vue";
import NotificationsIndicator from "../../../../shared/NotificationsIndicator.vue";
import {
  DirectMessage,
  ConvOpened,
  ConvClosed,
} from "../../../../../socketHandling/OutEvents";
import {
  isBlock,
  isRequestAttachment,
} from "../../../../../socketHandling/InterpretEvent";
import useAuthStore from "../../../../../store/AuthStore";
import useAttachmentStore from "../../../../../store/AttachmentStore";
import useNotificationStore from "../../../../../store/NotificationStore";
import {
  getConversationUids,
  getConversationContent,
} from "../../../../../services/account";

const inboxStore = useInboxStore();
const userStore = useUserStore();
const socketStore = useSocketStore();
const authStore = useAuthStore();
const attachmentStore = useAttachmentStore();
const notificationStore = useNotificationStore();

const section = ref<"USERS" | "MESSAGES">("USERS");
const resMsg = ref<IResMsg>({});
const currentUid = ref("");
const messagesBottomRef = ref<HTMLElement>();

const pendingAttachmentFile = ref<File>();

function watchForBlocksAndAttachmentRequest(e: MessageEvent) {
  const msg = JSON.parse(e.data);
  if (!msg) return;
  if (isBlock(msg)) {
    const otherUser =
      msg.data.blocker === authStore.uid ? msg.data.blocked : msg.data.blocker;
    if (otherUser === currentUid.value) {
      currentUid.value = "";
      section.value = "USERS";
    }
  }
  if (isRequestAttachment(msg)) {
    if (pendingAttachmentFile.value)
      attachmentStore.uploadAttachment(
        pendingAttachmentFile.value,
        msg.data.ID
      );
    else
      console.warn(
        "Server requested attachment file, but attachment file is undefined"
      );
  }
}

onMounted(async () => {
  try {
    resMsg.value = { msg: "", pen: true, err: false };
    let uids: string[] | null = await getConversationUids();
    uids?.forEach((uid) => {
      inboxStore.convs[uid] = [];
      userStore.cacheUser(uid);
    });
    resMsg.value = { msg: "", pen: false, err: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, pen: false, err: true };
  }
  socketStore.socket?.addEventListener(
    "message",
    watchForBlocksAndAttachmentRequest
  );
});

onBeforeUnmount(() => {
  socketStore.send({
    event_type: "CONV_CLOSED",
    data: {
      uid: currentUid.value,
    },
  } as ConvClosed);
  socketStore.socket?.removeEventListener(
    "message",
    watchForBlocksAndAttachmentRequest
  );
});

async function getConversation(uid: string) {
  try {
    resMsg.value = { msg: "", pen: true, err: false };
    const data = await getConversationContent(uid);
    socketStore.send({
      event_type: "CONV_CLOSED",
      data: {
        uid,
      },
    } as ConvClosed);
    socketStore.send({
      event_type: "CONV_OPENED",
      data: {
        uid,
      },
    } as ConvOpened);
    notificationStore.clearUserNotifications(uid);
    if (data?.friend_requests) {
      for await (const frq of data.friend_requests) {
        await userStore.cacheUser(
          frq.friended === authStore.uid ? frq.friender : frq.friended
        );
      }
    }
    if (data?.invitations) {
      for await (const inv of data.invitations) {
        await userStore.cacheUser(
          inv.invited === authStore.uid ? inv.inviter : inv.invited
        );
      }
    }
    if (data?.direct_messages) {
      for await (const { author_id } of data.direct_messages) {
        await userStore.cacheUser(author_id);
      }
    }
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

function handleSubmit(values: any, file?: File) {
  socketStore.send({
    event_type: "DIRECT_MESSAGE",
    data: {
      content: values.message,
      uid: currentUid.value,
      has_attachment: Boolean(file),
    },
  } as DirectMessage);
  pendingAttachmentFile.value = file;
}

watch(inboxStore.convs, async (oldVal, newVal) => {
  const oldConv = oldVal[currentUid.value];
  const newConv = newVal[currentUid.value];
  if (newConv.length > oldConv.length) {
    await nextTick(() => {
      messagesBottomRef.value?.scrollIntoView({ behavior: "auto" });
    });
  }
});
</script>

<template>
  <div class="messaging-section">
    <div v-if="section === 'MESSAGES'" class="messages-section">
      <div class="messages-section-messages-container">
        <div class="messages-section-messages">
          <MessagesItem
            :item="item"
            v-for="item in inboxStore.convs[currentUid] || []"
          />
          <div ref="messagesBottomRef" class="bottom" />
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
        <NotificationsIndicator
          v-if="notificationStore.getUserNotifications(uid)"
          :count="notificationStore.getUserNotifications(uid)"
        />
      </button>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.messaging-section {
  border: 2px solid var(--border-light);
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
    .bottom {
      padding: 0;
      margin: 0;
      width: 100%;
      height: 0px;
    }
  }

  .users {
    .user-container {
      padding: var(--gap-sm);
      border: 1px solid var(--border-light);
      background: var(--border-pale);
      width: 100%;
      display: flex;
      justify-content: space-between;
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
