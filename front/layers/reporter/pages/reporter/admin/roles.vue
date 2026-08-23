<script setup lang="ts">
import type { ItemsResponse, Role } from '~~/shared/api/types'

definePageMeta({ middleware: ['auth', 'admin'] })

const api = useApi()
const { data, pending, error, refresh } = await useAsyncData('roles-page', () =>
  api<ItemsResponse<Role>>('/api/v1/auth/admin/roles'),
)
const items = computed(() => data.value?.items ?? [])

const dialog = ref(false)
const saving = ref(false)
const formError = ref('')
const form = reactive({ code: '', name_ru: '', name_kk: '' })

const open = () => {
  form.code = ''
  form.name_ru = ''
  form.name_kk = ''
  formError.value = ''
  dialog.value = true
}

const save = async () => {
  if (!form.code.trim() || !form.name_ru.trim()) {
    formError.value = 'Код и название (RU) обязательны'
    return
  }
  saving.value = true
  formError.value = ''
  try {
    await api('/api/v1/auth/admin/roles', { method: 'POST', body: { ...form } })
    dialog.value = false
    await refresh()
  } catch (e) {
    formError.value = apiErrorMessage(e)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <PageHeader title="Роли" description="Пользовательские роли и доступ к витринам">
      <template #actions>
        <Button label="Создать роль" icon="pi pi-plus" @click="open" />
      </template>
    </PageHeader>

    <Message v-if="error" severity="error" :closable="false" class="mb">Не удалось загрузить роли</Message>

    <DataTable :value="items" :loading="pending" data-key="id">
      <template #empty><div class="empty">Роли не найдены</div></template>
      <Column field="code" header="Код" />
      <Column field="name_ru" header="Название (RU)" />
      <Column field="name_kk" header="Название (KK)" />
      <Column header="Статус">
        <template #body="{ data: r }">
          <Tag :severity="r.status === 'active' ? 'success' : 'secondary'" :value="r.status" />
        </template>
      </Column>
    </DataTable>

    <Dialog v-model:visible="dialog" modal header="Новая роль" :style="{ width: '420px' }">
      <div class="form">
        <div class="field">
          <label for="code">Код</label>
          <InputText id="code" v-model="form.code" placeholder="например, auditor" />
        </div>
        <div class="field">
          <label for="ru">Название (RU)</label>
          <InputText id="ru" v-model="form.name_ru" />
        </div>
        <div class="field">
          <label for="kk">Название (KK)</label>
          <InputText id="kk" v-model="form.name_kk" />
        </div>
        <Message v-if="formError" severity="error" :closable="false">{{ formError }}</Message>
        <div class="actions">
          <Button label="Отмена" severity="secondary" text @click="dialog = false" />
          <Button label="Создать" icon="pi pi-check" :loading="saving" @click="save" />
        </div>
      </div>
    </Dialog>
  </div>
</template>

<style scoped>
.mb {
  margin-bottom: 1rem;
}
.empty {
  padding: 2rem;
  text-align: center;
  color: var(--ehd-muted);
}
.form {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}
.field label {
  font-size: 0.82rem;
  font-weight: 500;
  color: var(--ehd-ink-2);
}
.field :deep(.p-inputtext) {
  width: 100%;
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}
</style>
