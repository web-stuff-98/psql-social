<script lang="ts" setup>
import { Field, Form } from "vee-validate";
import { ref } from "vue";
import { IResMsg } from "../../../../../interfaces/GeneralInterfaces";
import { getUserByName } from "../../../../../services/user";
import useUserStore from "../../../../../store/UserStore";
import ErrorMessage from "../../../../shared/ErrorMessage.vue";
import ResMsg from "../../../../shared/ResMsg.vue";
import User from "../../../../shared/User.vue";

const userStore = useUserStore();

const formRef = ref<HTMLFormElement>();
const searchTimeout = ref<NodeJS.Timeout>();
const username = ref("");
const result = ref("");

const resMsg = ref<IResMsg>({});

async function handleSubmit() {
  const abortController = new AbortController();
  try {
    result.value = "";
    resMsg.value = { msg: "", err: false, pen: true };
    if (username.value.trim()) {
      const uid = await getUserByName(username.value);
      result.value = uid;
      if (uid) await userStore.cacheUser(uid);
    }
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
  return () => {
    abortController.abort();
  };
}

function handleInput(e: Event) {
  const target = e.target as HTMLInputElement;
  if (!target) return;
  username.value = target.value;
  if (searchTimeout.value) clearTimeout(searchTimeout.value);
  searchTimeout.value = setTimeout(handleSubmit, 300);
}
</script>

<template>
  <div :style="result ? { height: '100%' } : {}" class="find-user">
    <div v-if="result" class="result"><User :uid="result" /></div>
    <ErrorMessage name="username" />
    <ResMsg
      v-if="resMsg.msg && resMsg.msg !== 'User not found'"
      :resMsg="resMsg"
    />
    <Form ref="formRef" @submit="handleSubmit" class="search-container">
      <Field name="username" id="username" @change="handleInput" type="text" />
      <v-icon
        :class="resMsg.pen ? 'spin' : ''"
        :name="resMsg.pen ? 'pr-spinner' : 'io-search'"
      />
    </Form>
  </div>
</template>

<style lang="scss" scoped>
.find-user {
  border: 2px solid var(--border-light);
  width: 100%;
  position: relative;
  border-radius: var(--border-radius-sm);
  padding: var(--gap-sm);
  gap: var(--gap-sm);
  display: flex;
  flex-direction: column;
  .result {
    flex-grow: 1;
    width: 100%;
    display: flex;
    justify-content: flex-start;
    align-items: flex-start;
  }
  .search-container {
    padding: 0;
    margin: 0;
    display: flex;
    align-items: center;
    gap: 2px;
    input {
      padding: 2px 4px;
      width: 100%;
      border-radius: var(--border-radius-sm);
    }
    button {
      border: none;
      background: none;
      text-shadow: none;
      padding: 0;
    }
  }
}
</style>
