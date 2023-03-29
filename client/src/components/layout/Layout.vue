<script lang="ts" setup>
import { ref } from "vue";
import { IResMsg } from "../../interfaces/GeneralInterfaces";
import useAuthStore from "../../store/AuthStore";
import Modal from "../modal/Modal.vue";
import ModalCloseButton from "../shared/ModalCloseButton.vue";
import ResMsg from "../shared/ResMsg.vue";
const authStore = useAuthStore();

const resMsg = ref<IResMsg>({});

async function logout() {
  try {
    await authStore.logout();
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
}
</script>

<template>
  <Modal v-if="resMsg.msg">
    <ModalCloseButton @click="resMsg = {}" />
    <ResMsg :resMsg="resMsg" />
  </Modal>
  <div v-if="authStore.user" class="layout">
    <nav>
      <div class="nav-items">
        <button class="nav-item">Settings</button>
        <button class="nav-item" @click="logout">Logout</button>
      </div>
    </nav>
    <router-view :key="$route.fullPath" />
  </div>
</template>

<style lang="scss" scoped>
.layout {
  height: 100vh;
  width: 100vw;
  display: flex;
  flex-direction: column;
  nav {
    width: 100%;
    height: var(--nav-height);
    background: var(--nav-colour);
    display: flex;
    justify-content: flex-end;
    align-items: center;
    padding: var(--gap-md);
    .nav-items {
      display: flex;
      gap: var(--gap-md);
      .nav-item {
        color: white;
        font-weight: 600;
        padding: 0;
        margin: 0;
        background: none;
        font-size: var(--md);
        border: none;
        filter: opacity(0.866);
        cursor: pointer;
        transition: filter 100ms ease;
      }
      .nav-item:hover {
        text-decoration: underline;
        filter: opacity(1);
      }
    }
  }
}
</style>
