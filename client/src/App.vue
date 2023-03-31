<script lang="ts" setup>
import { ref, watch } from "vue";
import useBackgroundProcess from "./composables/useBackgroundProcess";
import useAuthStore from "./store/AuthStore";
import Modal from "./components/modal/Modal.vue";
import Login from "./components/modal/Login.vue";
import Register from "./components/modal/Register.vue";
import Welcome from "./components/modal/Welcome.vue";
import ModalCloseButton from "./components/shared/ModalCloseButton.vue";
import Layout from "./components/layout/Layout.vue";
import { IResMsg } from "./interfaces/GeneralInterfaces";
import ResMsg from "./components/shared/ResMsg.vue";
import UserDropdown from "./components/userDropdown/UserDropdown.vue";

const authStore = useAuthStore();

const backgroundProcessResMsg = ref<IResMsg>();
useBackgroundProcess({ resMsg: backgroundProcessResMsg });

const noUserModalSection = ref<"WELCOME" | "LOGIN" | "REGISTER">("WELCOME");

watch(authStore, (_, newVal) => {
  if (!newVal.uid) noUserModalSection.value = "WELCOME";
});
</script>

<template>
  <div class="container">
    <Layout />
    <!-- Intervals response message modal (eg, when refreshing token failed) -->
    <Modal v-if="backgroundProcessResMsg?.msg">
      <ModalCloseButton @click="() => (backgroundProcessResMsg = {})" />
      <ResMsg :resMsg="backgroundProcessResMsg" />
    </Modal>
    <!-- Welcome / Login / Register modal -->
    <Modal v-if="!authStore.uid">
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
    <UserDropdown />
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
