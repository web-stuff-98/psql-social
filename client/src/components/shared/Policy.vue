<template>
  <div class="policy-container">
    <h1>Cookies Policy</h1>
    <p>
      When you create an account, a cookie called "session_token" will be
      created. This contains an encrypted ID used to identify your session and
      facilitate your login. This cookie is protected by your browser. The
      cookie is regularly refreshed by the server.
    </p>
    <h2>Privacy Policy</h2>
    <p>
      None of your information is shared with any 3rd party, and your account
      will be automatically deleted after 20 minutes.
    </p>
  </div>
</template>

<style lang="scss" scoped>
.policy-container {
  max-width: 15rem;
  text-align: center;
  h1,
  h2 {
    margin-bottom: var(--gap-md);
  }
  p {
    font-size: var(--xs);
    margin-bottom: var(--gap-md);
  }
  p:last-of-type {
    margin: 0;
  }
}
</style>
