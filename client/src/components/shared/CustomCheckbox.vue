<script lang="ts" setup>
import { toRefs } from "vue";
import { useField } from "vee-validate";

const props = defineProps({
  modelValue: {
    type: null,
  },
  name: String,
  value: {
    type: Boolean,
  },
});

const { name } = toRefs(props);
//@ts-ignore
const { checked, handleChange } = useField(name, undefined, {
  type: "checkbox",
  checkedValue: true,
  uncheckedValue: false,
});
</script>

<template>
  <button type="button" @click="handleChange(!value)">
    {{ checked ? "TRUE" : "FALSE" }}
  </button>
</template>

<style scoped>
button {
  background: none;
  font-size: 16px;
  width: 100%;
  color: var(--text-colour);
  text-shadow: none;
}
button:hover {
  background: none;
}
</style>
