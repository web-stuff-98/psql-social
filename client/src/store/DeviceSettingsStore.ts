import { ref } from "vue";
import { IMediaOptions } from "../interfaces/GeneralInterfaces";

export const showDeviceSettings = ref(false);
export const selectedAudioInputDevice = ref("");
export const selectedVideoInputDevice = ref("");

export const mediaOptions = ref<IMediaOptions>({
    userMedia: {
      audio: true,
      video: false,
    },
    displayMedia: {
      video: false,
    },
  });