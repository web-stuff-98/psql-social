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
import Modal from "./Modal.vue";
import ModalCloseButton from "../shared/ModalCloseButton.vue";
import Policy from "../shared/Policy.vue";
import CustomCheckbox from "../shared/CustomCheckbox.vue";

const authStore = useAuthStore();
const socketStore = useSocketStore();
const userStore = useUserStore();

const resMsg = ref<IResMsg>({});
const showPolicy = ref(false);

async function handleSubmit(values: any) {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    await authStore.register(values.username, values.password, values.policy);
    await socketStore.connectSocket();
    await userStore.cacheUser(authStore.uid!, true);
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
}
</script>

<template>
  <Form v-if="!resMsg.pen" @submit="handleSubmit" class="register">
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
    <div class="input-label">
      <label class="underlined" @click="showPolicy = true" for="password"
        >You agree to the policy</label
      >
      <CustomCheckbox
        :rules="((v:boolean) => v ? true : 'You must agree to the policy') as any"
        name="policy"
      />
      <ErrorMessage name="policy" />
    </div>
    <Modal v-if="showPolicy">
      <ModalCloseButton @click="showPolicy = false" />
      <Policy />
    </Modal>
  </Form>
  <ResMsg :resMsg="resMsg" />
</template>

<style lang="scss" scoped>
.register {
  display: flex;
  flex-direction: column;
  gap: var(--gap-md);
  width: 12rem;
  max-width: 12rem;
  button {
    font-weight: 600;
    font-size: var(--md);
  }
}
</style>
