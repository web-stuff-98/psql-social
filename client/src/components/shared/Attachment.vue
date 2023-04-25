<script lang="ts" setup>
import { toRefs, ref, onMounted, onBeforeUnmount } from "vue";
import { IResMsg } from "../../interfaces/GeneralInterfaces";
import { baseURL, makeRequest } from "../../services/makeRequest";
import useAttachmentStore from "../../store/AttachmentStore";
import ResMsg from "./ResMsg.vue";
import ProgressBar from "../shared/Progress.vue";

const attachmentStore = useAttachmentStore();

const props = defineProps<{
  msgId: string;
  reverse?: boolean;
}>();
const { msgId } = toRefs(props);

const resMsg = ref<IResMsg>({ msg: "", err: false, pen: false });
const containerRef = ref<HTMLElement | null>(null);

const meta = attachmentStore.getAttachment(msgId.value);

const observer = new IntersectionObserver(([entry]) => {
  if (entry.isIntersecting) {
    attachmentStore.attachmentEnteredView(msgId.value);
  } else {
    attachmentStore.attachmentLeftView(msgId.value);
  }
});

onMounted(() => {
  observer.observe(containerRef.value!);
});
onBeforeUnmount(() => {
  observer.disconnect();
});

async function download() {
  const data = await makeRequest(`/api/attachment/${meta?.ID}`, {
    responseType: "arraybuffer",
  });
  const blob = new Blob([data], { type: meta?.mime });
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = `${baseURL}/api/attachment/${meta?.ID}`;
  link.click();
  URL.revokeObjectURL(link.href);
}
</script>

<template>
  <div
    :style="reverse ? { justifyContent: 'flex-end', textAlign: 'right' } : {}"
    ref="containerRef"
    class="attachment"
  >
    <div v-if="meta && meta.ratio === 1">
      <img
        :src="`${baseURL}/api/attachment/${msgId}`"
        v-if="
          meta.mime === 'image/jpeg' ||
          meta.mime === 'image/png' ||
          meta.mime === 'image/avif' ||
          meta.mime === 'image/webp'
        "
      />
      <video controls v-if="meta.mime === 'video/mp4'">
        <source type="video/mp4" :src="`${baseURL}/api/attachment/video/${msgId}`" />
      </video>
      <a
        :href="`${baseURL}/api/attachment/${meta.ID}`"
        @click.prevent="download"
        :style="{ flexDirection: 'row-reverse' }"
        type="button"
        v-else
      >
        <v-icon name="fa-download" />
        {{
          meta.name.length > 24 ? meta.name.slice(0, 24 - 1) + "..." : meta.name
        }}
      </a>
    </div>
    <ProgressBar v-if="meta && meta.ratio < 1" :ratio="meta.ratio" />
    <ResMsg :resMsg="resMsg" />
  </div>
</template>

<style lang="scss">
.attachment {
  display: flex;
  flex-direction: row;
  width: 100%;
  justify-content: flex-start;
  text-align: left;
  margin-top: 4px;
  img {
    border-radius: var(--border-radius-md);
    filter: drop-shadow(0px 2px 3px rgba(0, 0, 0, 0.166));
    max-width: 80%;
    border: 2px solid var(--border-medium);
  }
  button {
    padding: 3px;
    display: flex;
    gap: var(--gap-md);
    font-size: 0.666rem;
    line-height: 1;
    align-items: center;
    padding: var(--gap-md);
    box-shadow: none;
    border: none;
  }
  video {
    max-width: 70%;
    max-height: 10rem;
    border-radius: var(--border-radius-md);
  }
  button:hover {
    outline: 1px solid var(--border-medium);
  }
}
</style>
