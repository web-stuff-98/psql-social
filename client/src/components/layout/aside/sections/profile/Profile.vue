<script lang="ts" setup>
import { Field, Form } from "vee-validate";
import { onMounted, ref } from "vue";
import { IResMsg } from "../../../../../interfaces/GeneralInterfaces";
import { getUserBio } from "../../../../../services/user";
import { validateBio } from "../../../../../validators/validators";
import { uploadBio, uploadPfp } from "../../../../../services/account";
import useUserStore from "../../../../../store/UserStore";
import useAuthStore from "../../../../../store/AuthStore";
import Modal from "../../../../modal/Modal.vue";
import ErrorMessage from "../../../../shared/ErrorMessage.vue";
import ModalCloseButton from "../../../../shared/ModalCloseButton.vue";
import ResMsg from "../../../../shared/ResMsg.vue";
defineProps<{ closeClicked: Function }>();

const authStore = useAuthStore();
const userStore = useUserStore();
const user = userStore.getUser(authStore.uid as string);

const bio = ref("");
const bioInput = ref<HTMLElement>();
const pfpInput = ref<HTMLInputElement>();

const pfpFile = ref<File>();
const pfpUrl = ref<string>("");

const resMsg = ref<IResMsg>({});

onMounted(async () => {
  if (user?.pfp) pfpUrl.value = user.pfp;
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    const content = await getUserBio(user?.ID!);
    bio.value = content;
    // @ts-ignore
    bioInput.value = content;
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    const notFound = e === "Bio not found";
    resMsg.value = { msg: notFound ? "" : `${e}`, err: !notFound, pen: false };
    bio.value = "";
    // @ts-ignore
    bioInput.value = "";
  }
});

async function handleSubmit() {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    await uploadBio(bio.value);
    if (pfpFile.value) uploadPfp(pfpFile.value);
    resMsg.value = { msg: "", err: false, pen: false };
    const i = userStore.users.findIndex((u) => u.ID === authStore.uid);
    if (i !== -1) userStore.users[i].pfp = pfpUrl.value;
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
}

function selectImage(e: Event) {
  const target = e.target as HTMLInputElement;
  if (!target.files || !target.files[0]) return;
  if (pfpUrl.value && pfpFile.value) URL.revokeObjectURL(pfpUrl.value);
  const file = target.files[0];
  pfpFile.value = file;
  pfpUrl.value = URL.createObjectURL(file);
}
</script>

<template>
  <Modal>
    <Form @submit="handleSubmit" class="profile-section">
      <ModalCloseButton @click="closeClicked()" />
      <div class="pfp-name">
        <button
          :style="{ backgroundImage: `url(${pfpUrl})` }"
          @click="pfpInput?.click()"
          id="select profile picture"
          type="button"
          class="pfp"
        >
          <v-icon v-if="!pfpUrl" name="fa-user-alt" />
        </button>
        <!-- Hidden file input -->
        <input
          accept=".png,.jpeg.jpg"
          @change="selectImage"
          ref="pfpInput"
          type="file"
        />
        <div class="name">
          <div>
            {{ user?.username }}
          </div>
          <label for="select profile picture"
            >Click the image to select a new picture</label
          >
        </div>
      </div>
      <div class="bio-input-area">
        <label for="bio">Bio (can be left blank)</label>
        <Field
          as="textarea"
          type="textarea"
          name="bio"
          v-model="bio"
          ref="bioInput"
          id="bio"
          :rules="validateBio as any"
        />
        <ErrorMessage name="bio" />
      </div>
      <button type="submit">Update profile</button>
      <ResMsg :resMsg="resMsg" />
    </Form>
  </Modal>
</template>

<style lang="scss" scoped>
.profile-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  max-width: 12rem;
  .pfp-name {
    display: flex;
    align-items: center;
    justify-content: center;
    margin: var(--sm);
    gap: 4px;
    filter: drop-shadow(0px 2px 3px rgba(0, 0, 0, 0.166));
    .pfp {
      border: 2px outset var(--border-pale);
      min-height: 3.75rem;
      min-width: 3.75rem;
      background: var(--foreground-colour);
      border-radius: var(--border-radius-md);
      background-size: cover;
      background-position: center;
      svg {
        fill: var(--text-colour);
      }
    }
    .name {
      font-size: var(--lg);
      text-align: left;
      line-height: 0.3;
      div {
        margin: 0;
        padding: 0;
        line-height: 1;
        font-weight: 600;
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
      max-width: 12rem;
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
