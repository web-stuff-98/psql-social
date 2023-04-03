<script lang="ts" setup>
import useAuthStore from "../../store/AuthStore";
const authStore = useAuthStore();
</script>

<template>
  <div class="home">
    <h1>pSQL-Social</h1>
    <hr />
    <h2>Voip, video chat, screensharing & filesharing... faster and written better than previously.</h2>
    <p><b>Hint:</b> Click on a user to open an action dropdown menu</p>
  </div>
</template>

<style lang="scss" scoped>
.home {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  width: 100%;
  text-align: center;
  padding: var(--gap-lg);

  hr {
    border: none;
    border-bottom: 2px solid var(--border-heavy);
    width: 100%;
    max-width: 25rem;
    margin: 4px;
    mask-image: linear-gradient(
      to right,
      transparent 0%,
      black 50%,
      transparent 100%
    );
    -webkit-mask-image: linear-gradient(
      to right,
      transparent 0%,
      black 50%,
      transparent 100%
    );
  }

  p {
    margin: 1rem;
  }
}
</style>
