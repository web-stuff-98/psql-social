<script lang="ts" setup>
import { ref } from "vue";
import Modal from "./components/modal/Modal.vue";
import Login from "./components/modal/Login.vue";

const noUserModalSection = ref<"WELCOME" | "LOGIN" | "REGISTER">("WELCOME");
</script>

<template>
  <div class="container">
    <h1>Container</h1>
    <router-view :key="$route.fullPath" />
    <Modal>
      <Login />
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
