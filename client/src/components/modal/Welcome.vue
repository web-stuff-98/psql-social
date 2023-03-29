<script lang="ts" setup>
defineProps<{ onLoginClicked: Function; onRegisterClicked: Function }>();
</script>

<template>
  <div class="welcome">
    <h1>Welcome</h1>
    <hr />
    <p>
      This is my new Vue & Go social app, it's basically the same as Go-Vue-Chat
      but uses postgres and Pinia, and has been completely rewritten.<p>The
      backend code is faster, the client code is much neater and handles
      events differently. I am also using postgres instead of MongoDB. This version
      also has more features, along with the standard video chat, screen sharing
      & file sharing.
    </p>
    </p>
    <div class="buttons">
      <button type="button" @click="onLoginClicked()">Login</button>
      <button type="button" @click="onRegisterClicked()">Register</button>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.welcome {
  display: flex;
  flex-direction: column;
  max-width: 20rem;
  padding: var(--gap-sm);
  * {
    text-align: center;
  }
  p {
    margin-top: var(--gap-md);
  }
  h1 {
    padding: var(--gap-sm);
  }
  hr {
    width: 100%;
    margin: 0;
    padding: 0;
    border: none;
    border-bottom: 2px solid var(--border-heavy);
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
  .buttons {
    margin-top: var(--gap-md);
    display: flex;
    flex-grow: 1;
    width: 100%;
    gap: var(--gap-md);
    button {
      flex-grow: 1;
      width: 100%;
      font-size: var(--md);
      font-weight: 600;
      border: 2px solid var(--border-heavy);
    }
  }
}
</style>
