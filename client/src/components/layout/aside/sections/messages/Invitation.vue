<script lang="ts" setup>
import { onBeforeUnmount, onMounted, toRefs, computed } from "vue";
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

const inviter = userStore.getUser(inv.value.inviter)?.username;
const invited = userStore.getUser(inv.value.invited)?.username;
const room = computed(() => roomStore.getRoom(inv.value.room_id));

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
  roomStore.cacheRoom(inv.value.room_id, true);
});

onBeforeUnmount(() => {
  userStore.userLeftView(inv.value.invited);
  userStore.userLeftView(inv.value.inviter);
  roomStore.roomLeftView(inv.value.room_id);
});

const uppercaseFirstLetter = (str: string) =>
  str.charAt(0).toUpperCase() + str.slice(1);
</script>

<template>
  <div class="invitation-friend-request">
    <span v-if="inv.inviter !== authStore.uid">
      {{
        inv.accepted
          ? `You accepted ${inviter}'s invitation to ${room?.name}`
          : `${uppercaseFirstLetter(inviter || "")} sent you an invitation to ${
              room?.name
            }`
      }}
    </span>
    <span v-else>
      {{
        inv.accepted
          ? `${uppercaseFirstLetter(
              invited || ""
            )} accepted your invitation to ${room?.name}`
          : `You sent an invitation to ${invited} to join ${room?.name}`
      }}
    </span>
    <div v-if="inv.invited === authStore.uid && !inv.accepted" class="buttons">
      <button @click="respond(true)" type="button">Accept</button>
      <button @click="respond(false)" type="button">Decline</button>
    </div>
  </div>
</template>
