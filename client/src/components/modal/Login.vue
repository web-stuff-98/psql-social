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
import ResMsg from "../shared/ResMsg.vue";
import useSocketStore from "../../store/SocketStore";
import useUserStore from "../../store/UserStore";

const authStore = useAuthStore();
const socketStore = useSocketStore();
const userStore = useUserStore();

const resMsg = ref<IResMsg>({});

async function handleSubmit(values: any) {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    await authStore.login(values.username, values.password);
    await socketStore.connectSocket();
    await userStore.cacheUser(authStore.uid!, true);
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
    <button type="submit">Login</button>
    <ResMsg :resMsg="resMsg" />
  </Form>
</template>

<style lang="scss" scoped>
.login {
  display: flex;
  flex-direction: column;
  gap: var(--gap-md);
  width: 12rem;
  max-width: 12rem;
  button {
    border: 2px solid var(--border-heavy);
    font-weight: 600;
    font-size: var(--md);
  }
}
</style>
