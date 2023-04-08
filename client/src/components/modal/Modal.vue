<template>
  <Teleport to="body">
    <div class="modal-container">
      <div class="modal">
        <slot> </slot>
      </div>
    </div>
  </Teleport>
</template>
