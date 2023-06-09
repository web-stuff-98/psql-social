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
import Policy from "../shared/Policy.vue";
import Modal from "./Modal.vue";
import ModalCloseButton from "../shared/ModalCloseButton.vue";

const authStore = useAuthStore();
const socketStore = useSocketStore();
const userStore = useUserStore();

const resMsg = ref<IResMsg>({});
const showPolicy = ref(false);

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
  <Form v-if="!resMsg.pen" @submit="handleSubmit" class="login">
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
        type="password"
        name="password"
        id="password"
        :rules="validatePassword as any"
      />
      <ErrorMessage name="password" />
    </div>
    <button type="submit">Login</button>
    <Modal v-if="showPolicy">
      <ModalCloseButton @click="showPolicy = false" />
      <Policy />
    </Modal>
  </Form>
  <ResMsg :resMsg="resMsg" />
</template>

<style lang="scss" scoped>
.login {
  display: flex;
  flex-direction: column;
  gap: var(--gap-md);
  width: 10rem;
  max-width: 10rem;
  justify-content: center;
  align-items: center;
  button {
    width: 100%;
    font-weight: 600;
    font-size: var(--md);
  }
}
</style>
