<template>
  <div class="home">
    <h1>pSQL-Social</h1>
    <hr />
    <h3>
      Voip, video chat, screensharing & filesharing. Click on the menu sidebar
      to get started. Your account will be deleted after 20 minutes, or when the
      server enters sleep mode.
    </h3>
    <p><b>Hint:</b> Click on a user to open an action dropdown menu</p>
    <h2>By Jason</h2>
    <a href="https://github.com/web-stuff-98/psql-social"
      >https://github.com/web-stuff-98/psql-social</a
    >
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
    border-bottom: 2px solid var(--text-colour);
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
