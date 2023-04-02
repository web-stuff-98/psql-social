<script lang="ts" setup>
import { showFullscreenVid } from "../../../store/FullscreenVidStore";
</script>

<template>
  <div v-show="showFullscreenVid" class="modal-backdrop" />
  <div
    @click="showFullscreenVid = ''"
    v-show="showFullscreenVid"
    class="modal-container"
  >
    <div id="fullscreen-vid"></div>
  </div>
</template>
