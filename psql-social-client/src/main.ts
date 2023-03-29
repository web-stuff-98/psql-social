import { createApp } from "vue";
import { createPinia } from "pinia";
import router from "./router";
import "./styles.css";
import App from "./App.vue";
import { OhVueIcon, addIcons } from "oh-vue-icons";
import { IoClose } from "oh-vue-icons/icons";

addIcons(IoClose);

const pinia = createPinia();

createApp(App)
  .use(pinia)
  .use(router)
  .component("v-icon", OhVueIcon)
  .mount("#app");
