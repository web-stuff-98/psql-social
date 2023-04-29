<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { IResMsg, IRoom } from "../../../../../interfaces/GeneralInterfaces";
import {
  getRooms,
  getRoomsPage,
  searchRooms,
} from "../../../../../services/room";
import useRoomStore from "../../../../../store/RoomStore";
import ResMsg from "../../../../shared/ResMsg.vue";
import CreateRoom from "./CreateRoom.vue";
import Room from "./Room.vue";
import { isChangeEvent } from "../../../../../socketHandling/InterpretEvent";
import useSocketStore from "../../../../../store/SocketStore";
import useNotificationStore from "../../../../../store/NotificationStore";

// OWN_AND_MEMBERS mode is rooms the user is a member of and rooms the user owns (no pagination) (using getRooms())
// ALL mode is rooms that are public, rooms the user is a member of, the users own rooms (using pagination) (using getRoomsPage())
// SEARCH mode is the same as explore mode but includes search by name (using pagination) (using searchRooms())
enum EMode {
  "OWN_AND_MEMBERS" = "Joined or owned",
  "ALL" = "Explore all",
  "SEARCH" = "Search all",
}

const roomStore = useRoomStore();
const socketStore = useSocketStore();
const notificationStore = useNotificationStore();

const showCreate = ref(false);
const rooms = ref<string[]>([]);
const resMsg = ref<IResMsg>({});

// for pagination and only used for ALL and SEARCH modes
const currentPage = ref(1);
// for pagination and only used for ALL and SEARCH modes
const count = ref(0);
// only used for SEARCH mode
const searchTerm = ref("");
// only used for SEARCH mode
const searchTimeout = ref<NodeJS.Timeout>();
const mode = ref<EMode>(EMode.OWN_AND_MEMBERS);

function watchForNewRooms(e: MessageEvent) {
  const msg = JSON.parse(e.data);
  if (!msg) return;
  if (isChangeEvent(msg)) {
    if (msg.data.entity == "ROOM") {
      if (msg.data.change_type === "INSERT") {
        if (
          mode.value === EMode.OWN_AND_MEMBERS ||
          ((mode.value === EMode.ALL ||
            (mode.value === EMode.SEARCH &&
              searchTerm.value.includes(
                (msg.data.data as Partial<IRoom>).name?.toLocaleLowerCase()!
              ))) &&
            count.value < 30)
        )
          rooms.value.push(msg.data.data.ID);
      }
    }
  }
}

async function retrieveResult() {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    rooms.value = [];
    if (mode.value === EMode.OWN_AND_MEMBERS) {
      const result = await getRooms();
      if (result) roomStore.addRoomsData(result);
      rooms.value = result?.map((r) => r.ID) || [];
      count.value = rooms.value.length;
      currentPage.value = 1;
    }
    if (mode.value === EMode.ALL || mode.value === EMode.SEARCH) {
      const result = await (mode.value === EMode.ALL
        ? getRoomsPage(currentPage.value)
        : searchRooms(searchTerm.value, currentPage.value));
      if (result.rooms) roomStore.addRoomsData(result.rooms);
      rooms.value = result.rooms?.map((r) => r.ID) || [];
      count.value = result.count;
    }
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
}

function handleSearchInput(e: Event) {
  const target = e.target as HTMLInputElement;
  if (!target) return;
  if (searchTerm.value.length <= 16) {
    searchTerm.value = target.value;
    if (searchTimeout.value) clearTimeout(searchTimeout.value);
    searchTimeout.value = setTimeout(retrieveResult, 300);
  }
}

function handleModeChange(e: Event) {
  const target = e.target as HTMLSelectElement;
  changeMode(target.value as EMode);
}

function changeMode(newMode: EMode) {
  mode.value = newMode;
  currentPage.value = 1;
  count.value = 0;
  searchTerm.value = "";
  retrieveResult();
}

const maxPage = computed(() => Math.max(1, Math.ceil(count.value / 30)));
const prevPage = () => {
  currentPage.value = Math.max(1, currentPage.value - 1);
  retrieveResult();
};
const nextPage = () => {
  currentPage.value = Math.min(currentPage.value + 1, maxPage.value);
  retrieveResult();
};

onMounted(async () => {
  retrieveResult();
  socketStore.socket?.addEventListener("message", watchForNewRooms);
});

onBeforeUnmount(() => {
  socketStore.socket?.removeEventListener("message", watchForNewRooms);
});
</script>

<template>
  <div class="rooms">
    <div class="mode-selection-search-container">
      <select @change="handleModeChange" v-model="mode">
        <option :value="mode" v-for="mode in EMode">
          {{ mode }}
        </option>
      </select>
      <div v-if="mode === EMode.SEARCH" class="search-container">
        <input
          name="username"
          id="username"
          @change="handleSearchInput"
          type="text"
        />
        <v-icon
          :class="resMsg.pen ? 'spin' : ''"
          :name="resMsg.pen ? 'pr-spinner' : 'io-search'"
        />
      </div>
    </div>
    <div class="results-container">
      <div class="results">
        <Room
          :notificationCount="notificationStore.getRoomNotifications(rid)"
          :rid="rid"
          v-for="rid in rooms"
        />
      </div>
      <ResMsg :style="{ minWidth: '100%' }" :resMsg="resMsg" />
    </div>
    <div class="pagination-controls-create-button">
      <button
        @click="showCreate = true"
        type="button"
        name="create room"
        class="create-button"
      >
        <v-icon name="io-add-circle-sharp" />
        Create
      </button>
      <div
        v-if="mode === EMode.SEARCH || mode === EMode.ALL"
        class="pagination-controls"
      >
        <button @click="prevPage()" type="button">
          <v-icon name="bi-caret-left-fill" />
        </button>
        {{ currentPage }}/{{ maxPage }}
        <button @click="nextPage()" type="button">
          <v-icon name="bi-caret-right-fill" />
        </button>
      </div>
    </div>
  </div>
  <CreateRoom :closeClicked="() => (showCreate = false)" v-if="showCreate" />
</template>

<style lang="scss" scoped>
.rooms {
  border: 2px solid var(--border-light);
  width: 100%;
  height: 100%;
  position: relative;
  border-radius: var(--border-radius-sm);
  display: flex;
  flex-direction: column;
  .mode-selection-search-container {
    width: 100%;
    border-bottom: 1px solid var(--border-light);
    padding: 2px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    select,
    option {
      width: 100%;
      padding: 1px;
      border-radius: var(--border-radius-sm);
    }
  }
  .results-container {
    flex-grow: 1;
    border-bottom: 1px solid var(--border-pale);
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
    .results {
      overflow-y: auto;
      width: 100%;
      height: 100%;
      padding: var(--gap-sm);
      gap: var(--gap-sm);
      display: flex;
      flex-direction: column;
      position: absolute;
      left: 0;
      top: 0;
    }
  }
  .pagination-controls-create-button {
    padding: var(--gap-sm);
    width: 100%;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 2px;
    .create-button {
      padding: 0;
      background: none;
      color: var(--text-colour);
      text-shadow: none;
      font-weight: 600;
      gap: 1.5px;
      padding-right: 2px;
      font-size: var(--md);
      border: none;
      svg {
        width: 1.25rem;
        height: 1.25rem;
        fill: var(--text-colour);
      }
    }
    .pagination-controls,
    .create-button {
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .pagination-controls {
      height: 100%;
      border-radius: var(--border-radius-sm);
      font-size: var(--sm);
      button {
        padding: 0;
        margin: 0;
        background: none;
        border: none;
        svg {
          width: 1rem;
        }
      }
    }
  }
  .create-button:hover {
    background: var(--border-pale);
  }
}
</style>
