<script setup lang="ts">
// Каталог доступных пользователю опубликованных витрин (REP-FR-050).
definePageMeta({ middleware: 'auth' })

const views = useReporterViews()
const { data, pending, error, refresh } = await useAsyncData('reporter-views', () => views.list())
const items = computed(() => data.value?.items ?? [])
</script>

<template>
  <div>
    <PageHeader title="Витрины" description="Каталог доступных представлений данных" />
    <Card>
      <template #content>
        <div v-if="pending" class="center">
          <ProgressSpinner style="width: 2.5rem; height: 2.5rem" />
        </div>
        <ErrorState
          v-else-if="error"
          message="Не удалось загрузить каталог витрин."
          retryable
          @retry="refresh"
        />
        <EmptyState
          v-else-if="items.length === 0"
          icon="pi pi-table"
          title="Опубликованных витрин пока нет"
          hint="Витрины создаются администратором в конструкторе Reporter."
        />
        <ul v-else class="view-list">
          <li v-for="v in items" :key="v.slug">
            <NuxtLink :to="`/reporter/${v.slug}`" class="view-item">
              <i class="pi pi-table view-icon" aria-hidden="true" />
              <span class="view-body">
                <span class="view-name">{{ v.name }}</span>
                <span v-if="v.description" class="view-desc">{{ v.description }}</span>
              </span>
              <i class="pi pi-angle-right view-arrow" aria-hidden="true" />
            </NuxtLink>
          </li>
        </ul>
      </template>
    </Card>
  </div>
</template>

<style scoped>
.center {
  display: flex;
  justify-content: center;
  padding: 2.5rem 0;
}
.view-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.view-item {
  display: flex;
  align-items: center;
  gap: 0.9rem;
  padding: 0.85rem 1rem;
  border: 1px solid var(--ehd-border);
  border-radius: var(--ehd-radius-sm);
  text-decoration: none;
  color: var(--ehd-ink);
  transition: border-color 0.12s ease, box-shadow 0.12s ease;
}
.view-item:hover {
  border-color: var(--p-primary-300);
  box-shadow: var(--ehd-shadow-sm);
}
.view-icon {
  font-size: 1.2rem;
  color: var(--ehd-brand);
}
.view-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.view-name {
  font-weight: 600;
}
.view-desc {
  font-size: 0.85rem;
  color: var(--p-text-muted-color);
}
.view-arrow {
  margin-left: auto;
  color: var(--ehd-muted);
}
</style>
