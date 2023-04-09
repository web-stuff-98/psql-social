<script lang="ts" setup>
defineProps<{ noExtraTopPadding?: boolean }>();
</script>

<template>
  <Teleport to="body">
    <div class="modal-backdrop" />
    <div class="modal-container">
      <div
        :style="noExtraTopPadding ? { paddingTop: 'var(--gap-md)' } : {}"
        class="modal"
      >
        <slot> </slot>
      </div>
    </div>
  </Teleport>
</template>
