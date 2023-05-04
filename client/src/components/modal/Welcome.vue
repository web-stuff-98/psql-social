<script lang="ts" setup>
import { Cloudinary } from "@cloudinary/url-gen";
import { AdvancedVideo } from "@cloudinary/vue";
const cld = new Cloudinary({
  cloud: {
    cloudName: "dr7whzm5r",
  },
});
const vid = cld.video("psql-social_oi9gv6");
defineProps<{ onLoginClicked: Function; onRegisterClicked: Function }>();
</script>

<template>
  <div class="welcome">
    <h1>Welcome</h1>
    <hr />
    <AdvancedVideo controls :cld-vid="vid" />
    <p>
      This is my new Vue & Go social app, it has all the features of Go-Vue-Chat
      but it's written better and uses postgres. The performance and UI is
      better. You can login and try it out, or just watch the demo video. The
      video compression is heavy but it shows all features.
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
    font-size: var(--sm);
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
    margin-bottom: var(--gap-sm);
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
  video {
    border-radius: var(--border-radius-md);
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
    }
  }
}
</style>
