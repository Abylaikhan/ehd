<script setup lang="ts">
// Переиспользуемое состояние ошибки ЕХД (ТЗ, Принцип 6): Error / Access denied / Source unavailable.
defineProps<{ title?: string; message: string; retryable?: boolean }>()
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <div class="error-state">
    <i class="pi pi-exclamation-triangle error-icon" aria-hidden="true" />
    <p class="error-title">{{ title || 'Ошибка' }}</p>
    <p class="error-message">{{ message }}</p>
    <Button
      v-if="retryable"
      label="Повторить"
      icon="pi pi-refresh"
      outlined
      size="small"
      @click="emit('retry')"
    />
  </div>
</template>

<style scoped>
.error-state {
  text-align: center;
  padding: 2.5rem 1rem;
  color: var(--ehd-ink-2);
}
.error-icon {
  font-size: 2rem;
  display: block;
  margin-bottom: 0.75rem;
  color: var(--p-red-500);
}
.error-title {
  margin: 0 0 0.25rem;
  font-weight: 600;
}
.error-message {
  font-size: 0.9rem;
  margin: 0 0 1rem;
  color: var(--p-text-muted-color);
}
</style>
