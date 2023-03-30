<script lang="ts" setup>
defineProps<{ on: boolean; toggleFunc: Function; name: string }>();
</script>

<template>
  <div class="toggle-container">
    <label :for="name">{{ name }}</label>
    <button
      type="button"
      @click="() => toggleFunc()"
      :id="name"
      class="slider-container"
    >
      <div class="slider-bar" />
      <div
        :style="on ? { left: 'calc(100% - 12px)' } : { left: '0%' }"
        class="slider-handle"
      />
    </button>
  </div>
</template>

<style lang="scss" scoped>
.toggle-container {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  width: 100%;
  height: 1.5rem;
  gap: var(--gap-md);
  padding: var(--gap-lg) 0;
  margin-bottom: var(--gap-lg);
  .slider-container {
    width: 100%;
    position: relative;
    padding: 0;
    border: none;
    background: none;
    box-shadow: none;
    .slider-bar {
      position: absolute;
      width: 100%;
      top: calc(50% - 2px);
      height: 3px;
      background: var(--base);
      border-radius: 2px;
    }
    .slider-handle {
      position: absolute;
      left: 0;
      height: 7px;
      background: var(--foreground-hover);
      border: 2px solid var(--border-light);
      width: 8px;
      border-radius: 3px;
      top: calc(50% - 6px);
      transition: left 120ms linear;
    }
  }
}
</style>
