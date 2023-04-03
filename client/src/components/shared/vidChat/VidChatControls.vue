<script lang="ts" setup>
import { IMediaOptions } from "../../../interfaces/GeneralInterfaces";
import { mediaOptions } from "../../../store/DeviceSettingsStore";
defineProps<{
  exitButtonClicked: Function;
  updateOptions: (opts: Partial<IMediaOptions>) => void;
}>();
</script>

<template>
  <div class="control-buttons">
    <!-- Camera button -->
    <button
      @click="
        updateOptions({
          userMedia: {
            video: !mediaOptions.userMedia.video,
            audio: mediaOptions.userMedia.audio,
          },
        })
      "
      type="button"
    >
      <v-icon
        :name="
          mediaOptions.userMedia.video
            ? 'bi-camera-video-off'
            : 'bi-camera-video'
        "
      />
    </button>
    <!-- Screenshare button -->
    <button
      @click="
        updateOptions({
          displayMedia: {
            video: !mediaOptions.displayMedia.video,
          },
        })
      "
      type="button"
    >
      <v-icon
        :name="
          mediaOptions.displayMedia.video
            ? 'md-stopscreenshare'
            : 'md-screenshare'
        "
      />
    </button>
    <!-- Mute/unmute button -->
    <button
      @click="
        updateOptions({
          userMedia: {
            video: mediaOptions.userMedia.video,
            audio: !mediaOptions.userMedia.audio,
          },
        })
      "
      type="button"
    >
      <v-icon
        :name="
          mediaOptions.userMedia.audio ? 'bi-mic-mute-fill' : 'bi-mic-fill'
        "
      />
    </button>
    <!-- Hangup / close video chat button -->
    <button @click="exitButtonClicked()" class="close-button" type="button">
      <v-icon name="hi-phone-missed-call" />
    </button>
  </div>
</template>

<style lang="scss" scoped>
.control-buttons {
  display: flex;
  gap: 0.25rem;
  padding: 0.25rem;
  border: 3px outset var(--border-light);
  border-radius: 5pc;
  box-shadow: 0px 2px 3px rgba(0, 0, 0, 0.166);
  width: fit-content;
  button {
    border: 2px solid var(--border-heavy);
    border-radius: 50%;
    padding: 0;
    margin: 0;
    width: max(5vh, 2rem);
    height: max(5vh, 2rem);
    display: flex;
    align-items: center;
    justify-content: center;
    svg {
      width: 70%;
      height: 70%;
    }
  }
  .close-button {
    background: red;
    border: 2px solid var(--text-colour);
    svg {
      fill: none;
      margin-right: 0.1rem;
      margin-top: 0.1rem;
    }
  }
}
</style>
