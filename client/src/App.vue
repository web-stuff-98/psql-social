<script lang="ts" setup>
import { ref } from "vue";
import useAuthStore from "./store/AuthStore";
import Modal from "./components/modal/Modal.vue";
import Login from "./components/modal/Login.vue";
import Register from "./components/modal/Register.vue";
import Welcome from "./components/modal/Welcome.vue";
import ModalCloseButton from "./components/shared/ModalCloseButton.vue";

const authStore = useAuthStore();

const noUserModalSection = ref<"WELCOME" | "LOGIN" | "REGISTER">("WELCOME");
</script>

<template>
  <div class="container">
    <router-view :key="$route.fullPath" />
    <Modal v-if="!authStore.user">
      <ModalCloseButton
        v-if="noUserModalSection !== 'WELCOME'"
        @click="() => (noUserModalSection = 'WELCOME')"
      />
      <Login v-if="noUserModalSection === 'LOGIN'" />
      <Register v-if="noUserModalSection === 'REGISTER'" />
      <Welcome
        :onLoginClicked="() => (noUserModalSection = 'LOGIN')"
        :onRegisterClicked="() => (noUserModalSection = 'REGISTER')"
        v-if="noUserModalSection === 'WELCOME'"
      />
    </Modal>
  </div>
</template>

<style lang="scss" scoped>
.container {
  width: 100vw;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  .welcome-modal {
    display: flex;
    gap: var(--gap-md);
    flex-direction: column;
    input,
    button {
      border-radius: var(--border-radius-md);
    }
  }
}
</style>
