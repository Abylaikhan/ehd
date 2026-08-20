<script setup lang="ts">
// Каталог доступных витрин. Данные появятся с GET /api/v1/reporter/views.
</script>

<template>
  <div>
    <PageHeader
      title="Витрины"
      description="Каталог доступных представлений данных"
    />
    <Card>
      <template #content>
        <div class="empty-state">
          <i class="pi pi-table empty-icon" aria-hidden="true" />
          <p>Опубликованных витрин пока нет.</p>
          <p class="empty-hint">
            Витрины создаются администратором в конструкторе Reporter.
          </p>
        </div>
      </template>
    </Card>
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
.empty-hint {
  font-size: 0.9rem;
}
</style>
