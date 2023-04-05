import Home from "../components/pages/Home.vue";
import Policy from "../components/pages/Policy.vue";
import Call from "../components/pages/Call.vue";
import Room from "../components/pages/Room/Room.vue";
import { createRouter, createWebHashHistory } from "vue-router";
import useAuthStore from "../store/AuthStore";

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: "/", component: Home },
    { path: "/policy", component: Policy },
    { path: "/room/:id", component: Room },
    { path: "/call/:id", component: Call, name: "call" },
  ],
});

router.beforeEach((to) => {
  const store = useAuthStore();
  if (!store.uid) return "/";
});

export default router;
