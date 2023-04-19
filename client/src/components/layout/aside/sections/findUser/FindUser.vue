<script lang="ts" setup>
import { Field, Form } from "vee-validate";
import { ref } from "vue";
import { IResMsg } from "../../../../../interfaces/GeneralInterfaces";
import { searchUsers } from "../../../../../services/user";
import useUserStore from "../../../../../store/UserStore";
import ErrorMessage from "../../../../shared/ErrorMessage.vue";
import ResMsg from "../../../../shared/ResMsg.vue";
import User from "../../../../shared/User.vue";

const userStore = useUserStore();

const formRef = ref<HTMLFormElement>();
const searchTimeout = ref<NodeJS.Timeout>();
const username = ref("");
const result = ref<string[]>([]);

const resMsg = ref<IResMsg>({});

async function handleSubmit() {
  const abortController = new AbortController();
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    result.value = [];
    if (username.value.trim()) {
      const ids = await searchUsers(username.value);
      if (ids) {
        for await (const id of ids) {
          await userStore.cacheUser(id);
        }
      }
      result.value = ids || [];
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
  <div class="find-user">
    <div class="result-container">
      <div class="list">
        <User v-for="uid in result" :uid="uid" />
      </div>
    </div>
    <ErrorMessage name="username" />
    <ResMsg v-if="resMsg.msg" :resMsg="resMsg" />
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
  gap: var(--gap-sm);
  display: flex;
  flex-direction: column;
  height: 100%;
  .result-container {
    position: relative;
    flex-grow: 1;
    width: 100%;
    .list {
      position: absolute;
      overflow-y: auto;
      width: 100%;
      height: 100%;
      display: flex;
      justify-content: flex-start;
      align-items: flex-start;
      flex-direction: column;
      gap: var(--gap-md);
      padding: var(--gap-md);
    }
  }
  form {
    padding: var(--gap-sm);
  }
}
</style>
