<template>
  <div class="auth-plane">
    <slot />
  </div>
</template>

<style scoped>
.auth-plane {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 2rem;
  background:
    radial-gradient(1200px 600px at 15% -10%, rgba(11, 37, 69, 0.10), transparent 60%),
    radial-gradient(900px 500px at 100% 110%, rgba(39, 75, 120, 0.12), transparent 55%),
    var(--ehd-page);
}
</style>
