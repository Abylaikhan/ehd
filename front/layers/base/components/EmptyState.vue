<script setup lang="ts">
// Переиспользуемое пустое состояние ЕХД (ТЗ, Принцип 6).
defineProps<{ icon?: string; title: string; hint?: string }>()
</script>

<template>
  <div class="empty-state">
    <i :class="icon || 'pi pi-inbox'" class="empty-icon" aria-hidden="true" />
    <p class="empty-title">{{ title }}</p>
    <p v-if="hint" class="empty-hint">{{ hint }}</p>
    <slot />
  </div>
</template>

<style scoped>
.empty-state {
  text-align: center;
  padding: 2.5rem 1rem;
  color: var(--p-text-muted-color);
}
.empty-icon {
  font-size: 2rem;
  display: block;
  margin-bottom: 0.75rem;
}
.empty-title {
  margin: 0 0 0.25rem;
  font-weight: 600;
  color: var(--ehd-ink-2);
}
.empty-hint {
  font-size: 0.9rem;
  margin: 0;
}
</style>
