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
  rules: {
    type: String,
    default: undefined,
  },
});

const { name, rules } = toRefs(props);
//@ts-ignore
const { checked, handleChange } = useField(name, rules, {
  type: "checkbox",
  checkedValue: true,
  uncheckedValue: false,
});
</script>

<template>
  <button type="button" @click="handleChange(!value)">
    <v-icon :name="checked ? 'md-radiobuttonchecked' : 'md-radiobuttonunchecked'"/>
  </button>
</template>

<style lang="scss" scoped>
button {
  background: none;
  font-size: 16px;
  width: 100%;
  color: var(--text-colour);
  text-shadow: none;
  border: none;
  height: fit-content;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 3px;
  .checkbox-container {
    height: 0.7rem;
    width: 0.7rem;
    border: 1px solid var(--border-medium);
    border-radius: 3px;
    position: relative;
  }
}
button:hover {
  background: none;
}
</style>
