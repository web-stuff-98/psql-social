<script lang="ts" setup>
import useAuthStore from "../store/AuthStore";
const authStore = useAuthStore();
</script>

<template>
  <h1>Home</h1>
  {{ authStore.user ? "No user" : "User" }}
</template>
