<script lang="ts" setup>
defineProps<{ ratio: number }>();
</script>

<template>
  <div class="bar-container">
    <div class="progress-bar">
      <span class="progress" :style="{ width: `${ratio * 100}` }" />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.bar-container {
  width: 100%;
  height: 0.5rem;
  position: relative;
  .progress-bar {
    position: absolute;
    top: calc(50% - 4px);
    height: 3px;
    width: 100%;
    left: 0;
    border-radius: 2px;
    background: var(--border-pale);
    border: 1px solid var(--border-medium);
    display: flex;
    .progress {
      height: 100%;
      background: lime;
      transition: width 100ms linear;
      box-shadow: 0px 0px 6px lime;
    }
  }
}
</style>
