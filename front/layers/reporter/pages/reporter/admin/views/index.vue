<script setup lang="ts">
import type { ViewSummary } from '~~/shared/api/types'
import { statusMeta } from '../../../../utils/viewForm'

definePageMeta({ middleware: ['auth', 'admin'] })

const admin = useReporterAdmin()
const { data, pending, error, refresh } = await useAsyncData('admin-views', () => admin.views.list())
const items = computed(() => data.value?.items ?? [])

const actionError = ref('')
const busyId = ref('')

async function run(id: string, fn: () => Promise<unknown>) {
  actionError.value = ''
  busyId.value = id
  try {
    await fn()
    await refresh()
  } catch (e) {
    actionError.value = apiErrorMessage(e)
  } finally {
    busyId.value = ''
  }
}

const publish = (v: ViewSummary) => run(v.id, () => admin.views.publish(v.id))
const disable = (v: ViewSummary) => run(v.id, () => admin.views.disable(v.id))
const remove = (v: ViewSummary) => {
  if (!confirm(`Удалить витрину «${v.name}»?`)) return
  return run(v.id, () => admin.views.remove(v.id))
}
const fmtDate = (iso: string) => (iso ? iso.slice(0, 10) : '')
</script>

<template>
  <div>
    <PageHeader title="Витрины (конструктор)" description="Создание, настройка и публикация представлений данных">
      <template #actions>
        <Button label="Создать витрину" icon="pi pi-plus" @click="navigateTo('/reporter/admin/views/create')" />
      </template>
    </PageHeader>

    <Message v-if="error" severity="error" :closable="false" class="mb">Не удалось загрузить список витрин</Message>
    <Message v-if="actionError" severity="error" :closable="true" class="mb">{{ actionError }}</Message>

    <DataTable :value="items" :loading="pending" data-key="id" size="small">
      <template #empty><EmptyState icon="pi pi-table" title="Витрин пока нет" hint="Нажмите «Создать витрину»." /></template>
      <Column header="Название">
        <template #body="{ data: v }">
          <NuxtLink :to="`/reporter/admin/views/${v.id}`" class="name-link">{{ v.name }}</NuxtLink>
          <div class="slug">/{{ v.slug }}</div>
        </template>
      </Column>
      <Column header="Источник">
        <template #body="{ data: v }">{{ v.database }}.{{ v.table }}</template>
      </Column>
      <Column header="Статус">
        <template #body="{ data: v }">
          <Tag :severity="statusMeta(v.status).severity" :value="statusMeta(v.status).label" />
        </template>
      </Column>
      <Column header="Изменена">
        <template #body="{ data: v }">{{ fmtDate(v.updated_at) }}</template>
      </Column>
      <Column header="Действия" style="width: 12rem">
        <template #body="{ data: v }">
          <div class="row-actions">
            <Button
              v-tooltip.top="'Открыть'"
              icon="pi pi-pencil"
              text
              rounded
              size="small"
              @click="navigateTo(`/reporter/admin/views/${v.id}`)"
            />
            <Button
              v-tooltip.top="'Опубликовать'"
              icon="pi pi-check-circle"
              text
              rounded
              severity="success"
              size="small"
              :loading="busyId === v.id"
              @click="publish(v)"
            />
            <Button
              v-if="v.status === 'published'"
              v-tooltip.top="'Отключить'"
              icon="pi pi-ban"
              text
              rounded
              severity="warn"
              size="small"
              :loading="busyId === v.id"
              @click="disable(v)"
            />
            <Button
              v-tooltip.top="'Удалить'"
              icon="pi pi-trash"
              text
              rounded
              severity="danger"
              size="small"
              :loading="busyId === v.id"
              @click="remove(v)"
            />
          </div>
        </template>
      </Column>
    </DataTable>
  </div>
</template>

<style scoped>
.mb {
  margin-bottom: 1rem;
}
.name-link {
  font-weight: 600;
  color: var(--ehd-brand);
  text-decoration: none;
}
.name-link:hover {
  text-decoration: underline;
}
.slug {
  font-size: 0.78rem;
  color: var(--p-text-muted-color);
}
.row-actions {
  display: flex;
  gap: 0.15rem;
}
</style>
