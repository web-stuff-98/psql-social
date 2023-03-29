<script lang="ts" setup>
import { ErrorMessage } from "vee-validate";
defineProps<{ name: string }>();
</script>

<template>
  <ErrorMessage class="error-message" :name="name" />
</template>

<style lang="scss" scoped>
.error-message {
  margin-top: var(--gap-md);
  color: red;
  font-weight: 600;
  font-size: var(--xs);
}
</style>
