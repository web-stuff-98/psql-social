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
        <button id="select profile picture" type="button" class="pfp" />
        <div class="name">
          <div>
            {{ user?.username }}
          </div>
          <label for="select profile picture">Select an image</label>
        </div>
      </div>
      <div class="bio-input-area">
        <label>Introduce yourself:</label>
        <textarea />
      </div>
      <button type="submit">Update profile</button>
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
      background: none;
    }
    .name {
      font-weight: 600;
      font-size: var(--lg);
      text-align: left;
      line-height: 0.7;
      div {
        margin: 0;
        padding: 0;
      }
      label {
        margin: 0;
        padding: 0;
        font-size: var(--xs);
        font-style: italic;
        filter: opacity(0.88);
      }
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
    textarea {
      max-width: 15rem;
      max-height: 15rem;
      min-width: 12rem;
      min-height: 8rem;
    }
  }

  button[type="submit"] {
    margin-top: var(--gap-md);
    width: 100%;
  }
}
</style>
