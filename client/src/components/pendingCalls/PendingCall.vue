<script lang="ts" setup>
import { onMounted, toRefs, ref, onBeforeUnmount, computed } from "vue";
import useUserStore from "../../store/UserStore";
import ring from "../../assets/ring.wav";

const userStore = useUserStore();

const props = defineProps<{
  uid: string;
  index: number;
  acceptClicked: Function;
  cancelHangupClicked: Function;
  showAcceptDevice: boolean;
}>();
const { uid } = toRefs(props);
const user = computed(() => userStore.getUser(uid.value));

const sound = ref();

onMounted(() => {
  const audio = new Audio(ring);
  sound.value = audio;
  audio.loop = true;
  audio.play();
  userStore.cacheUser(uid.value);
  userStore.userEnteredView(uid.value);
});

onBeforeUnmount(() => {
  sound.value.pause();
  sound.value = undefined;
  userStore.userLeftView(uid.value);
});
</script>

<template>
  <div
    :style="{
      backgroundImage: `url(${user?.pfp})`,
    }"
    class="pending-call"
  >
    {{ !user?.pfp ? user?.username : "" }}
    <!-- Accept call button -->
    <button
      v-if="showAcceptDevice"
      @click="() => acceptClicked(index)"
      class="accept-button"
    >
      <v-icon name="hi-phone-incoming" />
    </button>
    <!-- Cancel/Hangup call button -->
    <button
      @click="() => cancelHangupClicked(index)"
      class="cancel-hangup-button"
    >
      <v-icon name="hi-phone-missed-call" />
    </button>
  </div>
</template>

<style lang="scss" scoped>
.pending-call {
  min-width: 4rem;
  min-height: 4rem;
  border-radius: 50%;
  border: 2px solid var(--border-medium);
  background-size: cover;
  background-position: center;
  font-weight: 600;
  font-size: var(--xs);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  background: var(--base-colour);
  filter: drop-shadow(0px, 2px, 2px rgba(0, 0, 0.5));
  button {
    border: none;
    padding: 0;
    margin: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: none;
    position: absolute;
    width: 1.5rem;
    height: 1.5rem;
    svg {
      width: 100%;
      height: 100%;
      fill: none;
    }
  }
  .cancel-hangup-button {
    filter: drop-shadow(0px 2px 3px black);
    bottom: -8px;
    right: -8px;
    background: red;
    border-radius: 50%;
    padding: 3px;
    border: 2px solid var(--text-colour);
    transform: scaleX(-1);
  }
  .accept-button {
    filter: drop-shadow(0px 2px 3px black);
    bottom: -8px;
    left: -8px;
    background: lime;
    border-radius: 50%;
    padding: 3px;
    border: 2px solid var(--text-colour);
  }
}
</style>
