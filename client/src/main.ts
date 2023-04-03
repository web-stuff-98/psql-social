import { createApp } from "vue";
import { createPinia } from "pinia";
import router from "./router";
import "./styles.css";
import App from "./App.vue";
import { OhVueIcon, addIcons } from "oh-vue-icons";
import {
  IoClose,
  PrSpinner,
  MdErrorRound,
  HiSolidMenu,
  FaUserAlt,
  IoSend,
  IoSearch,
  IoAddCircleSharp,
  MdSensordoorRound,
  MdModeeditoutline,
  MdDeleteRound,
  BiPaperclip,
  HiPhoneMissedCall,
  HiPhoneIncoming,
  MdScreenshare,
  MdStopscreenshare,
  BiCameraVideo,
  BiCameraVideoOff,
  BiMicMuteFill,
  BiMicFill,
} from "oh-vue-icons/icons";

addIcons(
  IoClose,
  PrSpinner,
  MdErrorRound,
  HiSolidMenu,
  FaUserAlt,
  IoSend,
  IoSearch,
  IoAddCircleSharp,
  MdSensordoorRound,
  MdModeeditoutline,
  MdDeleteRound,
  BiPaperclip,
  HiPhoneMissedCall,
  HiPhoneIncoming,
  MdScreenshare,
  MdStopscreenshare,
  BiCameraVideo,
  BiCameraVideoOff,
  BiMicMuteFill,
  BiMicFill
);

const pinia = createPinia();

createApp(App)
  .use(pinia)
  .use(router)
  .component("v-icon", OhVueIcon)
  .mount("#app");
