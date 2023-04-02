<script lang="ts" setup>
import { onMounted, toRefs, ref, onBeforeUnmount, computed } from "vue";
import useUserStore from "../../store/UserStore";
import ring from "../../../assets/ring.wav";

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
});

onBeforeUnmount(() => {
  sound.value.pause();
  sound.value = undefined;
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
  width: 4rem;
  height: 4rem;
  border-radius: 50%;
  border: 2px solid var(--base);
  background-size: cover;
  background-position: center;
  font-weight: 600;
  filter: drop-shadow(0px, 2px, 2px rgba(0, 0, 0.5));
  display: flex;
  align-items: center;
  justify-content: center;
  button {
    border: none;
    padding: 0;
    margin: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: none;
    position: absolute;
    width: 2rem;
    height: 2rem;
    svg {
      width: 100%;
      height: 100%;
      fill: none;
    }
  }
  .cancel-hangup-button {
    filter: drop-shadow(var(--shadow-medium));
    bottom: 3px;
    right: 3px;
    background: red;
    border-radius: 50%;
    padding: 3px;
    border: 2px solid var(--text-color);
    transform: scaleX(-1);
  }
  .accept-button {
    filter: drop-shadow(var(--shadow-medium));
    bottom: 3px;
    left: 3px;
    background: lime;
    border-radius: 50%;
    padding: 3px;
    border: 2px solid var(--text-color);
  }
}
</style>
