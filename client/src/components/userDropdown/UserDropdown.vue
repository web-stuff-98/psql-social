<script lang="ts" setup>
import { onMounted, watch, ref, onBeforeUnmount, nextTick } from "vue";
import { userdropdownStore } from "../../store/UserDropdownStore";
import { IResMsg } from "../../interfaces/GeneralInterfaces";
import { getRooms } from "../../services/room";
import ResMsg from "../shared/ResMsg.vue";
import InviteToRoomCard from "./InviteToRoomCard.vue";
import useSocketStore from "../../store/SocketStore";
import useAuthStore from "../../store/AuthStore";
import { Field, Form } from "vee-validate";
import { validateMessage } from "../../validators/validators";
import ErrorMessage from "../shared/ErrorMessage.vue";
import {
  Block,
  DirectMessage,
  FriendRequest,
  Invitation,
  CallUser,
} from "../../socketHandling/OutEvents";
import { Ban } from "../../socketHandling/OutEvents";
import useRoomStore from "../../store/RoomStore";
import { bioUid } from "../../store/ViewBioStore";

enum EUserdropdownMenuSection {
  "MENU" = "Menu",
  "INVITE_TO_ROOM" = "Invite to room",
  "DIRECT_MESSAGE" = "Direct message",
}

const mousePos = ref<{ left: number; top: number }>({ left: 0, top: 0 });
const menuPos = ref<{ left: number; top: number }>({ left: 0, top: 0 });
const containerRef = ref<HTMLElement>();
const mouseInside = ref(false);
const handleMouseMove = (e: MouseEvent) =>
  (mousePos.value = { left: e.clientX, top: e.clientY });
const section = ref<EUserdropdownMenuSection>(EUserdropdownMenuSection.MENU);
const handleMouseEnter = () => (mouseInside.value = true);
const handleMouseLeave = () => (mouseInside.value = false);
const getOwnRoomIDsResMsg = ref<IResMsg>({});

const socketStore = useSocketStore();
const authStore = useAuthStore();
const roomStore = useRoomStore();

// this could be nicer but i cant be asked
function adjust() {
  if (
    containerRef.value?.clientWidth! + mousePos.value.left >
    window.innerWidth
  ) {
    menuPos.value.left = window.innerWidth - containerRef.value?.clientWidth!;
  }
  if (
    containerRef.value?.clientHeight! + mousePos.value.top >
    window.innerHeight
  ) {
    menuPos.value.top = window.innerHeight - containerRef.value?.clientHeight!;
  }
}

watch(userdropdownStore, async () => {
  menuPos.value = mousePos.value;
  section.value = EUserdropdownMenuSection.MENU;
  getOwnRoomIDsResMsg.value = { msg: "", err: false, pen: false };
  await nextTick(() => adjust());
});

watch(section, async () => {
  await nextTick(() => adjust());
});

onMounted(() => {
  window.addEventListener("mousemove", handleMouseMove);
  window.addEventListener("resize", adjust);
});

onBeforeUnmount(() => {
  window.removeEventListener("mousemove", handleMouseMove);
  window.removeEventListener("resize", adjust);
});

const directMessageClicked = () =>
  (section.value = EUserdropdownMenuSection.DIRECT_MESSAGE);

const ownRoomIDs = ref<string[]>([]);
async function inviteToRoomClicked() {
  section.value = EUserdropdownMenuSection.INVITE_TO_ROOM;
  try {
    ownRoomIDs.value = [];
    getOwnRoomIDsResMsg.value = { msg: "", err: false, pen: true };
    let rooms = await getRooms();
    if (!rooms) rooms = [];
    rooms = rooms.filter((r) => r.author_id === authStore.uid);
    ownRoomIDs.value = rooms.map((r) => r.ID);
    getOwnRoomIDsResMsg.value = {
      msg: rooms.length > 0 ? "" : "You have no rooms",
      err: false,
      pen: false,
    };
  } catch (e) {
    getOwnRoomIDsResMsg.value = { msg: `${e}`, err: true, pen: false };
  }
}

function inviteToRoom(roomId: string, uid: string) {
  socketStore.send({
    event_type: "INVITATION",
    data: { uid, room_id: roomId },
  } as Invitation);
  userdropdownStore.open = false;
}

function friendRequestClicked() {
  socketStore.send({
    event_type: "FRIEND_REQUEST",
    data: { uid: userdropdownStore.subject },
  } as FriendRequest);
  userdropdownStore.open = false;
}

function blockClicked() {
  socketStore.send({
    event_type: "BLOCK",
    data: { uid: userdropdownStore.subject },
  } as Block);
  userdropdownStore.open = false;
}

function banClicked() {
  socketStore.send({
    event_type: "BAN",
    data: { uid: userdropdownStore.subject, room_id: userdropdownStore.roomId },
  } as Ban);
  userdropdownStore.open = false;
}

function callClicked() {
  socketStore.send({
    event_type: "CALL_USER",
    data: { uid: userdropdownStore.subject },
  } as CallUser);
  userdropdownStore.open = false;
}

function bioClicked() {
  bioUid.value = userdropdownStore.subject;
  userdropdownStore.open = false;
}

const msgInputRef = ref<HTMLElement | null>();
function submitDirectMessage(values: any) {
  //@ts-ignore
  msgInputRef.value = "";
  socketStore.send({
    event_type: "DIRECT_MESSAGE",
    data: {
      content: values.content,
      uid: userdropdownStore.subject,
    },
  } as DirectMessage);
  userdropdownStore.open = false;
}
</script>

<template>
  <div
    @mouseenter="handleMouseEnter"
    @mouseleave="handleMouseLeave"
    v-if="userdropdownStore.open"
    class="user-dropdown"
    ref="containerRef"
    :style="{ left: `${menuPos.left}px`, top: `${menuPos.top}px` }"
  >
    <!-- Menu section -->
    <div v-if="section === EUserdropdownMenuSection.MENU" class="menu">
      <button @click="inviteToRoomClicked">Invite to room</button>
      <button @click="directMessageClicked">Direct message</button>
      <button @click="friendRequestClicked">Friend request</button>
      <button @click="blockClicked">Block</button>
      <button @click="bioClicked">Bio</button>
      <button
        v-if="userdropdownStore.roomId && roomStore.getRoom(userdropdownStore.roomId)?.author_id! === authStore.uid"
        @click="banClicked"
      >
        Ban
      </button>
      <button @click="callClicked">Call user</button>
    </div>
    <!-- Direct message section -->
    <Form
      @submit="submitDirectMessage"
      v-if="section === EUserdropdownMenuSection.DIRECT_MESSAGE"
      class="direct-message"
    >
      <div class="hor">
        <Field
          name="content"
          @input="validateMessage as any"
          ref="msgInputRef"
        />
        <button type="submit">
          <v-icon name="io-send" />
        </button>
      </div>
      <ErrorMessage name="content" />
    </Form>
    <!-- Invite to room section -->
    <div
      v-if="section === EUserdropdownMenuSection.INVITE_TO_ROOM"
      class="invite-to-room"
    >
      <ResMsg :resMsg="getOwnRoomIDsResMsg" />
      <div
        @click="() => inviteToRoom(id, userdropdownStore.subject)"
        class="room-container"
        :key="id"
        v-for="id in ownRoomIDs"
      >
        <InviteToRoomCard :id="id" />
      </div>
    </div>
    <!-- Close button -->
    <button @click="userdropdownStore.open = false" class="close-button">
      <v-icon name="io-close" />
    </button>
  </div>
</template>

<style lang="scss" scoped>
.user-dropdown {
  position: fixed;
  padding: 2px;
  gap: 2px;
  background: var(--base-colour);
  border: 1px solid var(--border-medium);
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  border-radius: var(--border-radius-md);
  border-top-left-radius: var(--border-radius-sm);
  z-index: 100;
  box-shadow: 0px 2px 4px rgba(0, 0, 0, 0.333);
  .menu {
    padding: 0;
    width: fit-content;
    display: flex;
    gap: 2px;
    flex-direction: column;
    align-items: center;
    button {
      display: flex;
    }
  }
  .direct-message {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    .hor {
      display: flex;
      gap: 2px;
      align-items: center;
      width: 100%;
      input {
        min-width: calc(100% - 1.5rem);
      }
      button {
        padding: 0;
        display: flex;
        border: none;
        box-shadow: none;
        background: none;
        width: 2rem;
      }
    }
  }
  .invite-to-room {
    display: flex;
    flex-direction: column;
    gap: 2px;
    .room-container {
      padding: 0;
    }
  }
  button {
    padding: 4px;
    text-align: left;
    box-shadow: none;
    flex-grow: 1;
    width: 100%;
    font-size: var(--sm);
  }
  .close-button {
    width: fit-content;
    height: fit-content;
    flex-grow: 0;
    padding: 0;
  }
}
</style>
