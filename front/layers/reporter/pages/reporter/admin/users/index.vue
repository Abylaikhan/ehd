<script setup lang="ts">
import type { UserList, UserView, Role, Reference, ItemsResponse, UpdateUserRequest } from '~~/shared/api/types'

definePageMeta({ middleware: ['auth', 'admin'] })

const api = useApi()

const page = ref(1)
const pageSize = ref(50)
const q = ref('')
const status = ref<string | undefined>(undefined)

const statusOptions = [
  { label: 'Все статусы', value: undefined },
  { label: 'Ожидает', value: 'pending' },
  { label: 'Активен', value: 'active' },
  { label: 'Заблокирован', value: 'blocked' },
]

const { data, pending, error, refresh } = await useAsyncData(
  'admin-users',
  () =>
    api<UserList>('/api/v1/auth/admin/users', {
      query: {
        page: page.value,
        page_size: pageSize.value,
        q: q.value || undefined,
        status: status.value || undefined,
      },
    }),
  { watch: [page, pageSize] },
)

const { data: roles } = await useAsyncData('ref-roles', () => api<ItemsResponse<Role>>('/api/v1/auth/admin/roles'))
const { data: regions } = await useAsyncData('ref-regions', () => api<ItemsResponse<Reference>>('/api/v1/auth/admin/regions'))
const { data: departments } = await useAsyncData('ref-departments', () => api<ItemsResponse<Reference>>('/api/v1/auth/admin/departments'))

const items = computed(() => data.value?.items ?? [])
const total = computed(() => data.value?.total_count ?? 0)
const first = computed(() => (page.value - 1) * pageSize.value)

const actionError = ref('')

const applyFilters = () => {
  page.value = 1
  refresh()
}
const onPage = (e: { page: number; rows: number }) => {
  page.value = e.page + 1
  pageSize.value = e.rows
}

const statusSeverity = (s: string) => (s === 'active' ? 'success' : s === 'blocked' ? 'danger' : 'warn')
const statusLabel = (s: string) => (s === 'active' ? 'Активен' : s === 'blocked' ? 'Заблокирован' : 'Ожидает')
const fmtDate = (iso: string) => (iso ? iso.slice(0, 10) : '')

// --- карточка (Drawer) ---
const drawer = ref(false)
const editing = ref<UserView | null>(null)
const saving = ref(false)
const editForm = reactive<UpdateUserRequest>({
  iin_verified: false,
  status: 'active',
  role_ids: [],
  region_ids: [],
  department_ids: [],
})

const openEdit = (u: UserView) => {
  actionError.value = ''
  editing.value = u
  editForm.iin_verified = u.iin_verified
  editForm.status = u.status
  editForm.role_ids = (roles.value?.items ?? []).filter((r) => u.roles.includes(r.code)).map((r) => r.id)
  editForm.region_ids = (regions.value?.items ?? []).filter((r) => u.region_codes.includes(r.code)).map((r) => r.id)
  editForm.department_ids = (departments.value?.items ?? []).filter((d) => u.department_codes.includes(d.code)).map((d) => d.id)
  drawer.value = true
}

const saveEdit = async () => {
  if (!editing.value) return
  saving.value = true
  actionError.value = ''
  try {
    await api(`/api/v1/auth/admin/users/${editing.value.id}`, { method: 'PATCH', body: { ...editForm } })
    drawer.value = false
    await refresh()
  } catch (e) {
    actionError.value = apiErrorMessage(e)
  } finally {
    saving.value = false
  }
}

const unlock = async (u: UserView) => {
  actionError.value = ''
  try {
    await api(`/api/v1/auth/admin/users/${u.id}/unlock`, { method: 'POST' })
    await refresh()
  } catch (e) {
    actionError.value = apiErrorMessage(e)
  }
}

const tempDialog = ref(false)
const tempPw = ref('')
const issueTempPassword = async (u: UserView) => {
  actionError.value = ''
  try {
    const r = await api<{ temporary_password: string }>(`/api/v1/auth/admin/users/${u.id}/temp-password`, { method: 'POST' })
    tempPw.value = r.temporary_password
    tempDialog.value = true
  } catch (e) {
    actionError.value = apiErrorMessage(e)
  }
}
</script>

<template>
  <div>
    <PageHeader title="Пользователи" description="Реестр учётных записей, проверка ИИН, роли и профиль доступа" />

    <div class="toolbar">
      <IconField class="grow">
        <InputIcon class="pi pi-search" />
        <InputText v-model="q" placeholder="Поиск по логину или email" @keyup.enter="applyFilters" />
      </IconField>
      <Select v-model="status" :options="statusOptions" option-label="label" option-value="value" placeholder="Статус" show-clear />
      <Button label="Применить" icon="pi pi-filter" @click="applyFilters" />
      <Button icon="pi pi-refresh" severity="secondary" outlined aria-label="Обновить" :loading="pending" @click="refresh" />
    </div>

    <Message v-if="error" severity="error" :closable="false" class="mb">Не удалось загрузить пользователей</Message>
    <Message v-if="actionError" severity="error" :closable="true" class="mb">{{ actionError }}</Message>

    <DataTable
      :value="items"
      lazy
      paginator
      :rows="pageSize"
      :first="first"
      :total-records="total"
      :rows-per-page-options="[20, 50, 100, 200]"
      :loading="pending"
      data-key="id"
      class="card-table"
      @page="onPage"
    >
      <template #empty>
        <div class="empty">Пользователи не найдены</div>
      </template>
      <Column field="login" header="Логин" />
      <Column field="full_name" header="ФИО" />
      <Column field="iin_masked" header="ИИН" />
      <Column header="Статус">
        <template #body="{ data: u }">
          <Tag :severity="statusSeverity(u.status)" :value="statusLabel(u.status)" />
        </template>
      </Column>
      <Column header="Роли">
        <template #body="{ data: u }">
          <span v-if="!u.roles.length" class="muted">—</span>
          <Chip v-for="r in u.roles" :key="r" :label="r" class="role-chip" />
        </template>
      </Column>
      <Column header="Создан">
        <template #body="{ data: u }">{{ fmtDate(u.created_at) }}</template>
      </Column>
      <Column header="" :style="{ width: '150px' }">
        <template #body="{ data: u }">
          <div class="row-actions">
            <Button icon="pi pi-pencil" text rounded size="small" aria-label="Открыть" v-tooltip.top="'Открыть'" @click="openEdit(u)" />
            <Button icon="pi pi-lock-open" text rounded size="small" severity="secondary" aria-label="Разблокировать"
              v-tooltip.top="'Разблокировать'" :disabled="u.status !== 'blocked'" @click="unlock(u)" />
            <Button icon="pi pi-key" text rounded size="small" severity="secondary" aria-label="Временный пароль"
              v-tooltip.top="'Временный пароль'" @click="issueTempPassword(u)" />
          </div>
        </template>
      </Column>
    </DataTable>

    <!-- Карточка пользователя -->
    <Drawer v-model:visible="drawer" position="right" :style="{ width: '440px' }" :header="editing?.login ?? 'Пользователь'">
      <div v-if="editing" class="edit">
        <div class="edit-row"><span class="edit-k">ФИО</span><span>{{ editing.full_name }}</span></div>
        <div class="edit-row"><span class="edit-k">ИИН</span><span>{{ editing.iin_masked }}</span></div>
        <div class="edit-row"><span class="edit-k">Email</span><span>{{ editing.email }}</span></div>

        <div class="field">
          <label>Статус</label>
          <Select v-model="editForm.status" :options="statusOptions.filter(o => o.value)" option-label="label" option-value="value" />
        </div>
        <div class="field field-row">
          <label>ИИН подтверждён</label>
          <ToggleSwitch v-model="editForm.iin_verified" />
        </div>
        <div class="field">
          <label>Роли</label>
          <MultiSelect v-model="editForm.role_ids" :options="roles?.items ?? []" option-label="code" option-value="id"
            placeholder="Выберите роли" display="chip" filter />
        </div>
        <div class="field">
          <label>Регионы</label>
          <MultiSelect v-model="editForm.region_ids" :options="regions?.items ?? []" option-label="name_ru" option-value="id"
            placeholder="Регионы (row scope)" display="chip" filter />
        </div>
        <div class="field">
          <label>Подразделения</label>
          <MultiSelect v-model="editForm.department_ids" :options="departments?.items ?? []" option-label="name_ru" option-value="id"
            placeholder="Подразделения (row scope)" display="chip" filter />
        </div>

        <Message v-if="actionError" severity="error" :closable="false">{{ actionError }}</Message>

        <div class="edit-actions">
          <Button label="Отмена" severity="secondary" text @click="drawer = false" />
          <Button label="Сохранить" icon="pi pi-check" :loading="saving" @click="saveEdit" />
        </div>
      </div>
    </Drawer>

    <!-- Временный пароль -->
    <Dialog v-model:visible="tempDialog" modal header="Временный пароль" :style="{ width: '380px' }">
      <p class="temp-hint">Передайте пользователю. Пароль действует ограниченное время; после входа обязательна смена.</p>
      <div class="temp-box">{{ tempPw }}</div>
    </Dialog>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-bottom: 1rem;
}
.grow {
  flex: 1;
}
.grow :deep(.p-inputtext) {
  width: 100%;
}
.mb {
  margin-bottom: 1rem;
}
.card-table :deep(.p-datatable-header) {
  border: none;
}
.empty {
  padding: 2rem;
  text-align: center;
  color: var(--ehd-muted);
}
.muted {
  color: var(--ehd-muted);
}
.role-chip {
  margin-right: 0.3rem;
  font-size: 0.78rem;
}
.row-actions {
  display: flex;
  gap: 0.15rem;
}
.edit {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.edit-row {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  font-size: 0.9rem;
  border-bottom: 1px solid var(--ehd-border);
  padding-bottom: 0.5rem;
}
.edit-k {
  color: var(--ehd-muted);
}
.field {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}
.field label {
  font-size: 0.82rem;
  font-weight: 500;
  color: var(--ehd-ink-2);
}
.field-row {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
}
.field :deep(.p-select),
.field :deep(.p-multiselect) {
  width: 100%;
}
.edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 0.5rem;
}
.temp-hint {
  font-size: 0.85rem;
  color: var(--ehd-ink-2);
  margin: 0 0 0.75rem;
}
.temp-box {
  font-family: monospace;
  font-size: 1.1rem;
  padding: 0.75rem;
  background: var(--ehd-page);
  border: 1px solid var(--ehd-border);
  border-radius: var(--ehd-radius-sm);
  text-align: center;
  user-select: all;
}
</style>
