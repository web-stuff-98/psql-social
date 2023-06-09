import Home from "../components/pages/Home.vue";
import Policy from "../components/pages/Policy.vue";
import Call from "../components/pages/Call.vue";
import Room from "../components/pages/Room/Room.vue";
import { createRouter, createWebHashHistory } from "vue-router";

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: "/", component: Home, name: "home" },
    { path: "/policy", component: Policy, name: "policy" },
    { path: "/room/:id", component: Room, name: "room" },
    { path: "/call/:id", component: Call, name: "call" },
  ],
});

export default router;
