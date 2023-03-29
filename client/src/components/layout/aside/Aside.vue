<script lang="ts" setup>
import { ref } from "vue";
import EAsideSection from "../../../enums/EAsideSection";
import Profile from "./sections/Profile.vue";

const currentSection = ref<EAsideSection>(EAsideSection.FRIENDS);
const show = ref(false);
</script>

<template>
  <aside :style="show ? {} : { width: 'fit-content' }">
    <button v-if="!show" @click="show = true" type="button" class="show-button">
      <v-icon name="hi-solid-menu" />
    </button>
    <div v-show="show" class="inner">
      <!-- Aside section menu buttons -->
      <div class="buttons">
        <button
          @click="currentSection = section"
          v-for="section in EAsideSection"
        >
          {{ section }}
        </button>
      </div>
      <Profile
        :closeClicked="() => (currentSection = EAsideSection.FRIENDS)"
        v-if="currentSection === EAsideSection.PROFILE"
      />
      <button
        @click="show = false"
        type="button"
        class="aside-close-button close-button"
      >
        <v-icon name="io-close" />
      </button>
    </div>
  </aside>
</template>

<style lang="scss" scoped>
aside {
  width: var(--aside-width);
  height: 100%;
  border-right: 2px solid var(--border-pale);
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  .show-button {
    background: none;
    border: none;
    text-shadow: none;
    padding: 0;
    height: 100%;
    svg {
      width: 1rem;
      height: 1rem;
      fill: var(--border-light);
      transform: rotateZ(90deg);
    }
  }
  .inner {
    width: 100%;
    height: 100%;
    .buttons {
      display: flex;
      flex-direction: column;
      gap: var(--gap-sm);
      padding: var(--gap-sm);
      button {
        text-align: left;
        font-size: var(--xs);
        padding: 4px;
        font-weight: 600;
      }
    }
    .aside-close-button {
      position: absolute;
      bottom: 3px;
      right: 3px;
    }
  }
}
</style>
