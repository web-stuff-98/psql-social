import Home from "../pages/Home.vue";
import Call from "../pages/Call.vue";
import Room from "../pages/Room/Room.vue";
import { createRouter, createWebHashHistory } from "vue-router";

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: "/", component: Home },
    { path: "/room/:id", component: Room },
    { path: "/call/:id", component: Call, name: "call" },
  ],
});

export default router;
