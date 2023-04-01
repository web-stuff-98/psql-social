<script lang="ts" setup>
import {
  IDirectMessage,
  IFriendRequest,
  IInvitation,
} from "../../../../../interfaces/GeneralInterfaces";
import Message from "../../../../shared/Message.vue";
import FriendRequest from "./FriendRequest.vue";
import Invitation from "./Invitation.vue";

/**
 * Determines if the conversation item is a message, invitation or friend request,
 * and renders the correct component
 */

defineProps<{
  item: IDirectMessage | IInvitation | IFriendRequest;
}>();
</script>

<template>
  <Invitation v-if="(item as any)['inviter']" :inv="item as IInvitation" />
  <FriendRequest
    v-if="(item as any)['friender']"
    :frq="item as IFriendRequest"
  />
  <!-- Only messages have ids -->
  <Message
    :msg="item as IDirectMessage"
    :roomMsg="false"
    v-if="(item as any)['ID']"
  />
</template>
