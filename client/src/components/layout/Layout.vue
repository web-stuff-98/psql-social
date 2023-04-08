<script lang="ts" setup>
import { computed, ref } from "vue";
import { IResMsg } from "../../interfaces/GeneralInterfaces";
import useAuthStore from "../../store/AuthStore";
import useUserStore from "../../store/UserStore";
import Modal from "../modal/Modal.vue";
import Aside from "./aside/Aside.vue";
import ModalCloseButton from "../shared/ModalCloseButton.vue";
import ResMsg from "../shared/ResMsg.vue";
import { useRouter } from "vue-router";

const authStore = useAuthStore();
const userStore = useUserStore();

const router = useRouter();

const resMsg = ref<IResMsg>({});

async function logout() {
  try {
    await authStore.logout();
    router.push("/");
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
}

function toggleDarkMode() {
  document.body.classList.toggle("dark-mode");
}

const username = computed(
  () => userStore.getUser(authStore.uid as string)?.username
);
</script>

<template>
  <Modal v-if="resMsg.msg">
    <ModalCloseButton @click="resMsg = {}" />
    <ResMsg :resMsg="resMsg" />
  </Modal>
  <div v-if="authStore.uid" class="layout">
    <nav>
      <div class="name">{{ username }}</div>
      <div class="nav-items">
        <RouterLink to="/policy">
          <button type="button" class="nav-item">Policy</button>
        </RouterLink>
        <RouterLink to="/">
          <button type="button" class="nav-item">Home</button>
        </RouterLink>
        <button type="button" class="nav-item" @click="toggleDarkMode">
          Darkmode
        </button>
        <button type="button" class="nav-item" @click="logout">Logout</button>
      </div>
    </nav>
    <div class="aside-main">
      <Aside />
      <main>
        <router-view :key="$route.fullPath" />
      </main>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.layout {
  height: min(30rem, calc(100% - var(--gap-lg)));
  width: min(40rem, calc(100% - var(--gap-lg)));
  transition: height 100ms ease;
  display: flex;
  flex-direction: column;
  border-radius: var(--border-radius-lg);
  overflow: hidden;
  border: 2px solid var(--border-heavy);
  box-shadow: 0px 2px 16px rgba(0, 0, 0, 0.33);
  background: var(--base-colour);
  nav {
    width: 100%;
    height: var(--nav-height);
    background: var(--nav-colour);
    display: flex;
    justify-content: flex-end;
    align-items: center;
    padding: var(--gap-md);
    border-bottom: 2px solid var(--border-medium);
    .name {
      flex-grow: 1;
      text-align: left;
      color: white;
      font-size: var(--xs);
    }
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
        text-shadow: none;
        transition: filter 100ms ease;
        cursor: pointer;
      }
      .nav-item:hover {
        text-decoration: underline;
      }
    }
  }
  .aside-main {
    display: flex;
    flex-grow: 1;
    height: 100%;
    width: 100%;
    main {
      width: 100%;
      display: flex;
      height: 100%;
    }
  }
}

@media (max-width: 600px) {
  .layout {
    height: calc(100% - var(--gap-lg));
  }
}
</style>
