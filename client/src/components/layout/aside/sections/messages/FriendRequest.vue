<script lang="ts" setup>
import { onBeforeUnmount, onMounted, toRefs } from "vue";
import { IFriendRequest } from "../../../../../interfaces/GeneralInterfaces";
import { FriendRequestResponse } from "../../../../../socketHandling/OutEvents";
import useAuthStore from "../../../../../store/AuthStore";
import useUserStore from "../../../../../store/UserStore";
import useSocketStore from "../../../../../store/SocketStore";
const props = defineProps<{ frq: IFriendRequest }>();

const { frq } = toRefs(props);

const userStore = useUserStore();
const authStore = useAuthStore();
const socketStore = useSocketStore();

const friender = userStore.getUser(frq.value.friender)?.username;
const friended = userStore.getUser(frq.value.friended)?.username;

function respond(accepted: boolean) {
  socketStore.send({
    event_type: "FRIEND_REQUEST_RESPONSE",
    data: {
      accepted,
      friender: frq.value.friender,
    },
  } as FriendRequestResponse);
}

onMounted(() => {
  userStore.userEnteredView(frq.value.friended);
  userStore.userEnteredView(frq.value.friender);
});

onBeforeUnmount(() => {
  userStore.userLeftView(frq.value.friended);
  userStore.userLeftView(frq.value.friender);
});

const uppercaseFirstLetter = (str: string) =>
  str.charAt(0).toUpperCase() + str.slice(1);
</script>

<template>
  <div class="invitation-friend-request">
    <span v-if="frq.friender !== authStore.uid">
      {{
        frq.accepted
          ? `You accepted ${friender}'s friend request`
          : `${uppercaseFirstLetter(friender || "")} sent you a friend request`
      }}
    </span>
    <span v-else>
      {{
        frq.accepted
          ? `${uppercaseFirstLetter(
              friended || ""
            )} accepted your friend request`
          : `You sent a friend request to ${friended}`
      }}
    </span>
    <div v-if="frq.friended === authStore.uid && !frq.accepted" class="buttons">
      <button @click="respond(true)" type="button">Accept</button>
      <button @click="respond(false)" type="button">Decline</button>
    </div>
  </div>
</template>
