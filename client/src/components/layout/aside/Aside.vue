<script lang="ts" setup>
import { ref } from "vue";
import EAsideSection from "../../../enums/EAsideSection";
import FindUser from "./sections/FindUser.vue";
import Profile from "./sections/Profile.vue";

const currentSection = ref<EAsideSection>(EAsideSection.FRIENDS);
const show = ref(false);
</script>

<template>
  <aside :style="show ? {} : { width: 'fit-content' }">
    <button v-if="!show" @click="show = true" type="button" class="show-button">
      <v-icon name="hi-solid-menu" />
    </button>
    <!-- Aside section menu buttons -->
    <div v-show="show" class="buttons">
      <button
        @click="currentSection = section"
        v-for="section in EAsideSection"
      >
        {{ section }}
      </button>
    </div>
    <div v-show="show" class="inner">
      <Profile
        :closeClicked="() => (currentSection = EAsideSection.FRIENDS)"
        v-if="currentSection === EAsideSection.PROFILE"
      />
      <FindUser
        :closeClicked="() => (currentSection = EAsideSection.FIND_USER)"
        v-if="currentSection === EAsideSection.FIND_USER"
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
  flex-direction: column;
  .buttons {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: var(--gap-sm);
    padding: var(--gap-sm);
    padding-bottom: 0;
    button {
      text-align: left;
      font-size: var(--sm);
      padding: 4px 5px;
      font-weight: 600;
      background: none;
      border: 2px solid var(--border-pale);
      color: var(--text-colour);
      text-shadow: none;
    }
  }
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
    padding: var(--gap-sm);
    padding-bottom: calc(6px + 1rem);
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    .aside-close-button {
      position: absolute;
      bottom: 3px;
      right: 3px;
    }
  }
}
</style>
