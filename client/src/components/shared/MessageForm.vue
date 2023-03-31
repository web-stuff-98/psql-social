<script lang="ts" setup>
import { Field, Form } from "vee-validate";
defineProps<{ handleSubmit: (values: any) => void }>();
</script>

<template>
  <Form @submit="handleSubmit">
    <Field ref="field" name="message" />
    <button type="submit">
      <v-icon name="io-send" />
    </button>
    <button type="button">
      <v-icon name="bi-paperclip" />
    </button>
  </Form>
</template>

<style lang="scss" scoped>
form {
  width: 100%;
  gap: var(--gap-sm);
  margin: 0;
  display: flex;
  align-items: center;
  input {
    width: 100%;
    height: 100%;
  }
  button {
    background: none;
    border: none;
  }
  button:hover {
    background: none;
  }
  button:first-of-type {
    padding-right: 0;
  }
  button:last-of-type {
    padding: 0;
    svg {
      width: 1.5rem;
      height: 1.5rem;
    }
  }
}
</style>
