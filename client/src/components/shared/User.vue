<script lang="ts" setup>
import useUserStore from "../../store/UserStore";
import useAuthStore from "../../store/AuthStore";
import { computed, onBeforeUnmount, onMounted, ref, toRefs } from "vue";
import { userdropdownStore } from "../../store/UserDropdownStore";

const userStore = useUserStore();
const authStore = useAuthStore();

const props = defineProps<{
  uid: string;
  noClick?: boolean;
  reverse?: boolean;
  date?: string;
  noPfp?: boolean;
  roomId?: string;
}>();
const { uid, date } = toRefs(props);

const container = ref<HTMLElement>();
const user = computed(() => userStore.getUser(uid.value));
const dateAsDate = computed(() => {
  if (!date?.value) return;
  return new Date(date.value);
});

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
  <div
    :style="reverse ? { flexDirection: 'row-reverse' } : {}"
    ref="container"
    class="user"
  >
    <button
      type="button"
      v-if="!noPfp"
      @click="
        {
          if (authStore.uid !== uid && !noClick)
            userdropdownStore.openOnSubject(uid, roomId);
        }
      "
      :style="{
        backgroundImage: `url(${user?.pfp})`,
        ...(authStore.uid === uid ? { cursor: 'default' } : {}),
        ...(date ? { width: '2.4rem', height: '2.4rem' } : {}),
      }"
      class="pfp"
    >
      <v-icon v-if="!user?.pfp" name="fa-user-alt" />
    </button>
    <div
      :style="reverse ? { alignItems: 'flex-end' } : {}"
      class="name-date-time"
    >
      <div class="name">
        {{ user?.username }}
      </div>
      <div v-if="date" class="date">
        {{ dateAsDate?.getFullYear() }}/{{ dateAsDate?.getMonth() }}/{{
          dateAsDate?.getDay()
        }}
      </div>
      <div v-if="date" class="time">
        {{ dateAsDate?.toTimeString().slice(0, 8) }}
      </div>
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
    background-color: var(--base-colour);
    padding: 0;
  }
  .name-date-time {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    .name {
      font-weight: 600;
      line-height: 0.866;
      font-size: var(--md);
      text-shadow: 0px 2px 1px rgba(0, 0, 0, 0.166);
    }
    .date,
    .time {
      line-height: 0.866;
      font-size: var(--xs);
    }
  }
}
</style>
