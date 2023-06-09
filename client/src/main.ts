import { createApp, markRaw } from "vue";
import { createPinia } from "pinia";
import router from "./router";
import "./styles.css";
import App from "./App.vue";
import { OhVueIcon, addIcons } from "oh-vue-icons";
import {
  IoClose,
  PrSpinner,
  MdErrorTwotone,
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
  FaDownload,
  MdRadiobuttonchecked,
  MdRadiobuttonunchecked,
  GiExpand,
  BiCaretLeftFill,
  BiCaretRightFill,
} from "oh-vue-icons/icons";

addIcons(
  IoClose,
  PrSpinner,
  MdErrorTwotone,
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
  FaDownload,
  MdRadiobuttonchecked,
  MdRadiobuttonunchecked,
  GiExpand,
  BiCaretLeftFill,
  BiCaretRightFill
);

const pinia = createPinia();
pinia.use(({ store }) => {
  store.router = markRaw(router);
});

createApp(App)
  .use(pinia)
  .use(router)
  .component("v-icon", OhVueIcon)
  .mount("#app");
