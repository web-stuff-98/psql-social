<script lang="ts" setup>
import {
  IDirectMessage,
  IFriendRequest,
  IInvitation,
} from "../../../../../interfaces/GeneralInterfaces";
import Message from "../../../../shared/Message.vue";
import FriendRequest from "./FriendRequest.vue";
import Invitation from "./Invitation.vue";
import useAuthStore from "../../../../../store/AuthStore";

/**
 * Determines if the conversation item is a message, invitation or friend request,
 * and renders the correct component
 */

defineProps<{
  item: IDirectMessage | IInvitation | IFriendRequest;
}>();

const authStore = useAuthStore();
</script>

<template>
  <Invitation v-if="(item as any)['inviter']" :inv="item as IInvitation" />
  <FriendRequest
    v-if="(item as any)['friender']"
    :frq="item as IFriendRequest"
  />
  <!-- Only messages have ids -->
  <Message
    :isAuthor="authStore.uid === (item as any)['author_id']"
    :msg="item as IDirectMessage"
    v-if="(item as any)['ID']"
  />
</template>
