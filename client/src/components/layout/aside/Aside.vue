<script lang="ts" setup>
import EAsideSection from "../../../enums/EAsideSection";
import { ref } from "vue";
import Profile from "./sections/Profile.vue";

const currentSection = ref<EAsideSection>(EAsideSection.FRIENDS);
</script>

<template>
  <aside>
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
  </aside>
</template>

<style lang="scss" scoped>
aside {
  width: var(--aside-width);
  height: 100%;
  border-right: 2px solid var(--border-pale);
  box-shadow: 1px 0px 3px rgba(0, 0, 0, 0.33);
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
}
</style>
