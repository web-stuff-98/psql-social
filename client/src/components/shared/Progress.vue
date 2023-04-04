<script lang="ts" setup>
defineProps<{ ratio: number }>();
</script>

<template>
  <div class="bar-container">
    <div class="bar">
      <span :style="{ width: `${ratio * 100}%` }" />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.bar-container {
  width: 6rem;
  height: 0.5rem;
  position: relative;
  .bar {
    position: absolute;
    top: calc(50% - 4px);
    height: 2px;
    width: 100%;
    left: 0;
    border-radius: 2px;
    background: var(--border-pale);
    border: 1px solid var(--border-medium);
    display: flex;
    span {
      height: 100%;
      background: lime;
      transition: width 100ms linear;
      box-shadow: 0px 0px 6px lime;
    }
  }
}
</style>
