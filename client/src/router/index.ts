import Home from "../pages/Home.vue";
import { createRouter, createWebHashHistory } from "vue-router";

const router = createRouter({
  history: createWebHashHistory(),
  routes: [{ path: "/", component: Home }],
});

export default router
