<script lang="ts" setup>
import { ref } from "vue";
import EAsideSection from "../../../enums/EAsideSection";
import FindUser from "./sections/findUser/FindUser.vue";
import Profile from "./sections/profile/Profile.vue";
import Rooms from "./sections/rooms/Rooms.vue";
import Messages from "./sections/messages/Messages.vue";
import DeviceSettings from "./sections/deviceSettings/DeviceSettings.vue";
import Friends from "./sections/friends/Friends.vue";

const currentSection = ref<EAsideSection>(EAsideSection.FRIENDS);
const show = ref(false);
</script>

<template>
  <aside :style="show ? {} : { minWidth: 'fit-content' }">
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
      <FindUser v-if="currentSection === EAsideSection.FIND_USER" />
      <Rooms v-if="currentSection === EAsideSection.ROOMS" />
      <Messages v-if="currentSection === EAsideSection.MESSAGES" />
      <DeviceSettings
        :closeClicked="() => (currentSection = EAsideSection.FRIENDS)"
        v-if="currentSection === EAsideSection.DEVICE_SETTINGS"
      />
      <Friends v-if="currentSection === EAsideSection.FRIENDS"/>
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
  min-width: var(--aside-width);
  height: 100%;
  border-right: 2px solid var(--border-light);
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
      border: 2px solid var(--border-light);
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
