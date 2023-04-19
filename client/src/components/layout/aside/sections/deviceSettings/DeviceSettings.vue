<script lang="ts" setup>
import { onMounted, ref } from "vue";
import {
  selectedAudioInputDevice,
  selectedVideoInputDevice,
} from "../../../../../store/DeviceSettingsStore";
import { IResMsg } from "../../../../../interfaces/GeneralInterfaces";
import ResMsg from "../../../../shared/ResMsg.vue";
import Modal from "../../../../modal/Modal.vue";
import ModalCloseButton from "../../../../shared/ModalCloseButton.vue";

defineProps<{ closeClicked: Function }>();

const videoInputDevices = ref<MediaDeviceInfo[]>([]);
const audioInputDevices = ref<MediaDeviceInfo[]>([]);

const resMsg = ref<IResMsg>({ msg: "", err: false, pen: false });

async function getDeviceList() {
  const devices = await navigator.mediaDevices.enumerateDevices();
  devices.forEach((device) => {
    if (device.kind === "audioinput") audioInputDevices.value.push(device);
    if (device.kind === "videoinput") videoInputDevices.value.push(device);
  });
  if (
    !audioInputDevices.value.find(
      (d) => d.deviceId === selectedAudioInputDevice.value
    )
  )
    selectedAudioInputDevice.value = "";
  if (
    !videoInputDevices.value.find(
      (d) => d.deviceId === selectedVideoInputDevice.value
    )
  )
    selectedVideoInputDevice.value = "";
}

function handleAudioInputDeviceChange(e: Event) {
  const target = e.target as HTMLSelectElement;
  selectedAudioInputDevice.value = target.value;
}

function handleVideoInputDeviceChange(e:Event) {
  const target = e.target as HTMLSelectElement;
  selectedVideoInputDevice.value = target.value;
}

onMounted(async () => {
  try {
    resMsg.value = { msg: "", err: false, pen: true };
    await getDeviceList();
    navigator.mediaDevices.ondevicechange = () =>
      getDeviceList().catch((e) => {
        resMsg.value = {
          msg: `${e}`,
          err: true,
          pen: false,
        };
      });
    resMsg.value = { msg: "", err: false, pen: false };
  } catch (e) {
    resMsg.value = { msg: `${e}`, err: true, pen: false };
  }
});

const removeParenthesis = (str?: string) =>
  str ||
  ""
    .replace(/ *\([^)]*\) */g, "")
    .replaceAll(")", "")
    .replaceAll("(", "");
</script>

<template>
  <Modal>
    <ModalCloseButton @click="closeClicked()" />
    <div class="device-settings-container">
      <div class="select-container">
        <label for="audio">Audio input device</label>
        <select
          @change="handleAudioInputDeviceChange"
          v-model="selectedAudioInputDevice"
          id="audio"
        >
          <option
            :value="selectedAudioInputDevice"
            v-if="selectedAudioInputDevice"
          >
            {{
              removeParenthesis(
                audioInputDevices.find(
                  (d) => d.deviceId === selectedAudioInputDevice
                )?.label
              )
            }}
          </option>
          <option
            :value="device.deviceId"
            :key="device.deviceId"
            v-for="device in audioInputDevices.filter(
              (d) => d.deviceId !== selectedAudioInputDevice
            )"
          >
            {{ removeParenthesis(device.label) }}
          </option>
        </select>
      </div>
      <div class="select-container">
        <label for="video">Video input device</label>
        <select @change="handleVideoInputDeviceChange" v-model="selectedVideoInputDevice" id="video">
          <option
            :value="selectedVideoInputDevice"
            v-if="selectedVideoInputDevice"
          >
            {{
              removeParenthesis(
                videoInputDevices.find(
                  (d) => d.deviceId === selectedVideoInputDevice
                )?.label
              )
            }}
          </option>
          <option
            :value="device.deviceId"
            :key="device.deviceId"
            v-for="device in videoInputDevices.filter(
              (d) => d.deviceId !== selectedVideoInputDevice
            )"
          >
            {{ removeParenthesis(device.label) }}
          </option>
        </select>
      </div>
      <ResMsg :resMsg="resMsg" />
    </div>
  </Modal>
</template>

<style lang="scss" scoped>
.device-settings-container {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: var(--padding);
  box-sizing: border-box;
  .select-container {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--gap-md);
    label {
      width: 100%;
      text-align: center;
      font-weight: 600;
      margin-top: var(--gap-lg);
      padding: 0 var(--gap-md);
    }
    select,
    option {
      width: 100%;
      border-radius: var(--border-radius-md);
      background: var(--base-colour);
    }
    select:focus,
    option:focus {
      background: var(--base-colour-hover);
    }
  }
}
</style>
