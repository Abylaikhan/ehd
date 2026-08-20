<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <div class="page-header">
    <div class="page-header-text">
      <h2 class="page-title">{{ title }}</h2>
      <p v-if="description" class="page-description">{{ description }}</p>
    </div>
    <div class="page-actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1.5rem;
}
.page-title {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--ehd-ink);
}
.page-description {
  margin: 0.3rem 0 0;
  color: var(--ehd-ink-2);
  font-size: 0.9rem;
}
.page-actions {
  display: flex;
  gap: 0.5rem;
  flex-shrink: 0;
}
</style>
