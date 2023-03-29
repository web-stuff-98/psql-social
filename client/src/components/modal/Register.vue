<script lang="ts" setup>
import { IResMsg } from "../../interfaces/GeneralInterfaces";
import useAuthStore from "../../store/AuthStore";
import { Form, Field } from "vee-validate";
import ErrorMessage from "../shared/ErrorMessage.vue";
import {
  validateUsername,
  validatePassword,
} from "../../validators/validators";
import { ref } from "vue";

const authStore = useAuthStore();

const resMsg = ref<IResMsg>({});

async function handleSubmit(values: any) {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    await authStore.register(values.username, values.spassword);
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
}
</script>

<template>
  <Form @submit="handleSubmit" class="login">
    <div class="input-label">
      <label for="username">Username</label>
      <Field
        type="text"
        name="username"
        id="username"
        :rules="validateUsername as any"
      />
      <ErrorMessage name="username" />
    </div>
    <div class="input-label">
      <label for="password">Password</label>
      <Field
        type="text"
        name="password"
        id="password"
        :rules="validatePassword as any"
      />
      <ErrorMessage name="password" />
    </div>
    <button type="submit">Register</button>
  </Form>
</template>

<style lang="scss" scoped>
.login {
  display: flex;
  flex-direction: column;
  gap: var(--gap-md);
  .input-label {
    display: flex;
    flex-direction: column;
    text-align: center;
    align-items: center;
    justify-content: center;
    input {
      text-align: center;
    }
  }
  button {
    border: 2px solid var(--border-heavy);
    font-weight: 600;
    font-size: var(--md);
  }
}
</style>
