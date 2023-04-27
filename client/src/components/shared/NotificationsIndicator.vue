<script lang="ts" setup>
defineProps<{ count: number }>();
</script>

<template>
  <div class="notifications-indicator">{{ count }}</div>
</template>

<style lang="scss" scoped>
.notifications-indicator {
  background: green;
  border-radius: 50%;
  width: 1.666rem;
  height: 1.666rem;
  font-size: var(--md);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  border: 2px solid var(--text-colour);
}
</style>
