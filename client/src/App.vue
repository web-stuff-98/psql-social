<script lang="ts" setup>
import { onMounted, ref, watch, onUnmounted } from "vue";
import useBackgroundProcess from "./composables/useBackgroundProcess";
import useAuthStore from "./store/AuthStore";
import useInterfaceStore from "./store/InterfaceStore";
import Modal from "./components/modal/Modal.vue";
import Login from "./components/modal/Login.vue";
import Register from "./components/modal/Register.vue";
import Welcome from "./components/modal/Welcome.vue";
import ModalCloseButton from "./components/shared/ModalCloseButton.vue";
import Layout from "./components/layout/Layout.vue";
import { IResMsg } from "./interfaces/GeneralInterfaces";
import ResMsg from "./components/shared/ResMsg.vue";
import UserDropdown from "./components/userDropdown/UserDropdown.vue";
import PendingCalls from "./components/pendingCalls/PendingCalls.vue";
import Bio from "./components/modal/Bio.vue";
import MessageModal from "./components/modal/MessageModal.vue";
import VidFullscreenModal from "./components/shared/vidChat/VidFullscreenModal.vue";
import * as THREE from "three";

const authStore = useAuthStore();
const interfaceStore = useInterfaceStore();

const backgroundProcessResMsg = ref<IResMsg>();
useBackgroundProcess({ resMsg: backgroundProcessResMsg });

const noUserModalSection = ref<"WELCOME" | "LOGIN" | "REGISTER">("WELCOME");

watch(authStore, (_, newVal) => {
  if (!newVal.uid) noUserModalSection.value = "WELCOME";
});

let camera: THREE.PerspectiveCamera;
let scene: THREE.Scene;
let renderer: THREE.WebGLRenderer;
let mesh: THREE.Mesh;
let animationFrameId: number;

onMounted(() => {
  init();
  animate();
});

onUnmounted(() => {
  cancelAnimationFrame(animationFrameId);
});

watch(interfaceStore, (_, newVal) => {
  if (newVal.darkMode) {
    changeSphereColor(0xffffff);
  } else {
    changeSphereColor(0x000000);
  }
});

function changeSphereColor(color: number) {
  const sphere = scene.getObjectByName("sphere");
  if (sphere && sphere instanceof THREE.Mesh) {
    const material = sphere.material;
    if (material instanceof THREE.MeshBasicMaterial) {
      material.color.set(color);
    }
  }
}

function init() {
  const container = document.querySelector("#sphere") as HTMLElement;

  camera = new THREE.PerspectiveCamera(
    70,
    container.clientWidth / container.clientHeight,
    0.01,
    10
  );
  camera.position.z = 1;

  scene = new THREE.Scene();

  const geometry = new THREE.SphereGeometry(0.666, 32, 32);
  const material = new THREE.MeshBasicMaterial({
    color: 0xffffff,
    wireframe: true,
  });
  mesh = new THREE.Mesh(geometry, material);
  mesh.name = "sphere";
  scene.add(mesh);

  renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
  renderer.setClearColor(0x000000, 0);
  renderer.setSize(container.clientWidth, container.clientHeight);
  container.appendChild(renderer.domElement);
}

function animate() {
  animationFrameId = requestAnimationFrame(animate);

  mesh.rotation.x += 0.000002;
  mesh.rotation.y += 0.0003;

  renderer.render(scene, camera);
}
</script>

<template>
  <div class="container">
    <div id="sphere" class="sphere-background" />
    <Layout />
    <!-- Background process response message modal (eg, when refreshing token failed) -->
    <Modal v-if="backgroundProcessResMsg?.msg">
      <ModalCloseButton @click="() => (backgroundProcessResMsg = {})" />
      <ResMsg :resMsg="backgroundProcessResMsg" />
    </Modal>
    <!-- Welcome / Login / Register modal -->
    <Modal
      :noExtraTopPadding="noUserModalSection === 'WELCOME'"
      v-if="!authStore.uid"
    >
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
    <PendingCalls />
    <Bio />
    <MessageModal />
    <VidFullscreenModal />
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
  .sphere-background {
    position: fixed;
    left: 0;
    top: 0;
    width: 100vw;
    height: 100vh;
    pointer-events: none;
    z-index: 2;
    filter: opacity(0.02);
  }
  .dark-mode {
    .sphere-background {
      filter: invert(1);
    }
  }
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
