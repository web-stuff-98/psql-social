<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, toRefs } from "vue";
import { IInvitation } from "../../../../../interfaces/GeneralInterfaces";
import { InvitationResponse } from "../../../../../socketHandling/OutEvents";
import useAuthStore from "../../../../../store/AuthStore";
import useUserStore from "../../../../../store/UserStore";
import useSocketStore from "../../../../../store/SocketStore";
import useRoomStore from "../../../../../store/RoomStore";
const props = defineProps<{ inv: IInvitation }>();

const { inv } = toRefs(props);

const userStore = useUserStore();
const authStore = useAuthStore();
const socketStore = useSocketStore();
const roomStore = useRoomStore();

const inviter = computed(() => userStore.getUser(inv.value.inviter)?.username);
const invited = computed(() => userStore.getUser(inv.value.invited)?.username);
const room = computed(() => roomStore.getRoom(inv.value.room_id)?.name);

function respond(accepted: boolean) {
  socketStore.send({
    event_type: "INVITATION_RESPONSE",
    data: {
      accepted,
      room_id: inv.value.room_id,
      inviter: inv.value.inviter,
    },
  } as InvitationResponse);
}

onMounted(() => {
  userStore.userEnteredView(inv.value.invited);
  userStore.userEnteredView(inv.value.inviter);
  roomStore.roomEnteredView(inv.value.room_id);
  roomStore.cacheRoom(inv.value.room_id);
});

onBeforeUnmount(() => {
  userStore.userLeftView(inv.value.invited);
  userStore.userLeftView(inv.value.inviter);
  roomStore.roomLeftView(inv.value.room_id);
});

function uppercaseFirstLetter(str: string) {
  return str.charAt(0).toUpperCase() + str.slice(1);
}
</script>

<template>
  <div class="invitation-friend-request">
    <span v-if="inv.inviter !== authStore.uid">
      {{ `${uppercaseFirstLetter(inviter || "")} ` }} sent you an invitation to
      {{ ` ${room}` }}
    </span>
    <span v-else>
      You sent an invitation to {{ ` ${invited} ` }} to join {{ ` ${room}` }}
    </span>
    <div v-if="inv.invited === authStore.uid" class="buttons">
      <button @click="respond(true)" type="button">Accept</button>
      <button @click="respond(false)" type="button">Decline</button>
    </div>
  </div>
</template>
