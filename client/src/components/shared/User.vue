<script lang="ts" setup>
import useUserStore from "../../store/UserStore";
import { computed, onBeforeUnmount, onMounted, ref, toRefs } from "vue";

const userStore = useUserStore();

const props = defineProps<{ uid: string }>();
const { uid } = toRefs(props);

const container = ref<HTMLElement>();
const user = computed(() => userStore.getUser(uid.value));

const observer = new IntersectionObserver(([entry]) => {
  if (entry.isIntersecting) userStore.userEnteredView(uid.value);
  else userStore.userLeftView(uid.value);
});

onMounted(() => {
  observer.observe(container.value!);
});

onBeforeUnmount(() => {
  observer.disconnect();
});
</script>

<template>
  <div ref="container" class="user">
    <div :style="{ backgroundImage: `url(${user?.pfp})` }" class="pfp">
      <v-icon v-if="!user?.pfp" name="fa-user-alt" />
    </div>
    <div class="name">
      {{ user?.username }}
    </div>
  </div>
</template>

<style lang="scss" scoped>
.user {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 3px;
  .pfp {
    border: 2px outset var(--text-colour);
    border-radius: var(--border-radius-md);
    width: 2rem;
    height: 2rem;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0px 3px 3px rgba(0, 0, 0, 0.33);
    background-size: cover;
    background-position: center;
  }
  .name {
    font-weight: 600;
    font-size: var(--md);
    text-shadow: 0px 2px 1px rgba(0, 0, 0, 0.166);
  }
}
</style>
