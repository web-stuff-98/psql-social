<script lang="ts" setup>
import { computed } from "vue";
import useAuthStore from "../../../../store/AuthStore";
import Modal from "../../../modal/Modal.vue";
import ModalCloseButton from "../../../shared/ModalCloseButton.vue";
defineProps<{ closeClicked: Function }>();
const authStore = useAuthStore();
const user = computed(() => authStore?.user);
</script>

<template>
  <Modal>
    <div class="profile-section">
      <ModalCloseButton @click="closeClicked()" />
      <div class="pfp-name">
        <div class="pfp" />
        <div class="name">
          {{ user?.username }}
        </div>
      </div>
      <div class="bio-input-area">
        <label>Introduce yourself:</label>
        <textarea />
      </div>
    </div>
  </Modal>
</template>

<style lang="scss" scoped>
.profile-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  .pfp-name {
    display: flex;
    align-items: center;
    justify-content: center;
    margin: var(--sm);
    gap: 4px;
    filter: drop-shadow(0px 2px 3px rgba(0, 0, 0, 0.166));
    .pfp {
      border: 2px outset var(--border-pale);
      height: 3rem;
      width: 3rem;
      border-radius: var(--border-radius-md);
    }
    .name {
      font-weight: 600;
      font-size: var(--lg);
    }
  }

  .bio-input-area {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    label {
      font-weight: 600;
      padding: 3px;
    }
  }
}
</style>
