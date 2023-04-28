<script lang="ts" setup>
import { onBeforeUnmount, onMounted, ref, toRefs, computed } from "vue";
import { IMediaOptions } from "../../../interfaces/GeneralInterfaces";
import { showFullscreenVid } from "../../../store/FullscreenVidStore";
import useUserStore from "../../../store/UserStore";

const userStore = useUserStore();

const props = defineProps<{
  userMedia: MediaStream | undefined;
  displayMedia: MediaStream | undefined;
  isOwner: boolean;
  uid?: string;
  smaller?: boolean;
  userMediaStreamID: string;
  // This should only be used for the current users container, not other peers.
  // Used to enable/disable microphone and camera access
  mediaOptions?: IMediaOptions;
  hasDisplayMediaVideo: boolean;
  hasUserMediaVideo: boolean;
}>();
const { userMedia, displayMedia, userMediaStreamID, uid, isOwner } =
  toRefs(props);

const user = computed(() => userStore.getUser(uid?.value!));
const forceMute = ref(false);
const hideSmallWindow = ref(false);

onMounted(() => userStore.userEnteredView(uid?.value as string));
onBeforeUnmount(() => userStore.userLeftView(uid?.value as string));
</script>

<template>
  <Teleport to="#fullscreen-vid" :disabled="showFullscreenVid !== uid">
    <div v-show="userMediaStreamID" class="container">
      <!-- Pfp container - For when there are no video streams present, or when the stream failed -->
      <div
        :style="{
          ...(user?.pfp ? { backgroundImage: `url(${user?.pfp})` } : {}),
        }"
        class="pfp"
        v-if="!hasDisplayMediaVideo && !hasUserMediaVideo"
      >
        <v-icon v-if="!user?.pfp" name="fa-user-alt" />
        <v-icon
          v-if="userMediaStreamID === 'FAILED'"
          name="md-error-round"
          class="error-icon"
        />
      </div>
      <!-- Video container - For when there is either or both video streams present -->
      <div
        v-show="hasDisplayMediaVideo || hasUserMediaVideo"
        class="vid-container"
      >
        <div class="name">
          {{ user?.username }}
        </div>
        <video
          v-show="hasDisplayMediaVideo || hasUserMediaVideo"
          :srcObject="hasDisplayMediaVideo ? displayMedia : userMedia"
          :muted="isOwner || forceMute"
          :class="
            showFullscreenVid === uid
              ? 'main-video-expanded'
              : smaller
              ? 'main-video-smaller'
              : 'main-video'
          "
          autoplay
        />
        <div
          v-if="!isOwner"
          :style="{
            width: 'fit-content',
            height: 'fit-content',
            position: 'absolute',
            bottom: '0.5rem',
            left: '0.25rem',
            padding: '0',
          }"
          class="buttons"
        >
          <!-- Fullscreen button -->
          <button
            @click="
              {
                showFullscreenVid = uid as string;
              }
            "
            class="fullscreen-button"
          >
            <v-icon
              :style="{
                filter: 'drop-shadow(0px, 2px, 2px black)',
              }"
              name="gi-expand"
            />
          </button>
        </div>
        <!-- Smaller video, for when display media is present -->
        <div
          :style="
            hideSmallWindow
              ? {
                  filter: 'opacity(0.666)',
                }
              : {}
          "
          class="small-video-container"
        >
          <video
            v-show="
              hasDisplayMediaVideo && hasUserMediaVideo && !hideSmallWindow
            "
            :srcObject="userMedia"
            :muted="isOwner || forceMute"
            autoplay
          />
          <div class="buttons">
            <!-- Mute/unmute button -->
            <button @click="forceMute = !forceMute" class="mute-button">
              <v-icon
                :style="{
                  filter: 'drop-shadow(0px, 2px, 2px black)',
                }"
                :name="forceMute ? 'bi-mic-mute-fill' : 'bi-mic-fill'"
              />
            </button>
            <!-- Hide/unhide button -->
            <button
              v-if="hasDisplayMediaVideo"
              @click="
                {
                  hideSmallWindow = !hideSmallWindow;
                }
              "
            >
              <v-icon
                :style="{
                  filter: 'drop-shadow(0px, 2px, 2px black)',
                }"
                :name="hideSmallWindow ? 'gi-expand' : 'io-close'"
              />
            </button>
          </div>
        </div>
      </div>
    </div>
    <!-- Loading spinner -->
    <v-icon v-show="!userMediaStreamID" class="spinner" name="pr-spinner" />
  </Teleport>
</template>

<style lang="scss" scoped>
.container {
  position: relative;
  width: fit-content;
  height: fit-content;

  .pfp {
    box-shadow: 0px 2px 3px rgba(0, 0, 0, 0.166);
    width: 9vh;
    height: 9vh;
    border: 3px solid var(--border-heavy);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: var(--shadow);
    gap: var(--gap-sm);
    background-size: cover;
    background-position: center;
    position: relative;
    svg {
      width: 60%;
      height: 60%;
    }
    .error-icon {
      position: absolute;
      top: -15%;
      right: -15%;
      width: 45%;
      height: 45%;
      fill: red;
      color: red;
    }
  }
  .vid-container {
    position: relative;
    .buttons {
      display: flex;
      justify-content: flex-end;
      height: 1.5rem;
      button svg {
        width: 100%;
        height: 100%;
      }
      .mute-button {
        svg {
          width: 70%;
          height: 70%;
        }
      }
      button {
        height: 1.5rem;
        width: 1.5rem;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 0;
        margin: 0;
        border: none;
        box-shadow: none;
        background: none;
      }
      .fullscreen-button {
        svg {
          fill: white;
          width: 2rem;
          height: 2rem;
        }
      }
      video {
        width: 100%;
      }
    }
    .name {
      padding: var(--gap-md);
      position: absolute;
      top: var(--gap-sm);
      left: var(--gap-sm);
      padding: var(--gap-md);
      font-weight: 600;
      text-shadow: 0px 2px 2px black;
      color: white;
    }
    .main-video,
    .main-video-smaller,
    .main-video-expanded,
    .small-video-container {
      border: 1px solid var(--border-light);
      height: auto;
      box-shadow: var(--shadow);
      border-radius: var(--border-radius-md);
    }
    .main-video {
      width: 30vw;
      max-width: min(9rem, 33vh);
    }
    .main-video-smaller {
      width: 7rem;
    }
    .main-video-expanded {
      width: 100%;
    }
    .small-video-container {
      position: absolute;
      bottom: 0;
      right: 0;
      width: 30%;
      height: auto;
      background: var(--base-colour);
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }
  }
}
.spinner {
  width: 7vh;
  height: 7vh;
  animation: spin 500ms linear infinite;
}
</style>
