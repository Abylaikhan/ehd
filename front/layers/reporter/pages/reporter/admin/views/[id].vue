<script setup lang="ts">
import type { ViewColumnDetail, QueryResult, Role } from '~~/shared/api/types'
import { DISPLAY_TYPES, ROW_SCOPE_MODES, columnsToPayload, statusMeta } from '../../../../utils/viewForm'
import { formatCell } from '../../../../utils/format'

definePageMeta({ middleware: ['auth', 'admin'] })

const route = useRoute()
const admin = useReporterAdmin()
const id = computed(() => String(route.params.id))

const { data: detail, pending, error, refresh } = await useAsyncData(`admin-view-${id.value}`, () =>
  admin.views.get(id.value),
)
const { data: rolesData } = await useAsyncData('admin-roles', () => admin.roles())
const roleOptions = computed<Role[]>(() => rolesData.value?.items ?? [])

// --- локальные редактируемые копии (draft до «Сохранить») ---
const columns = ref<ViewColumnDetail[]>([])
const roleCodes = ref<string[]>([])
const meta = reactive({
  name: '',
  slug: '',
  description: '',
  page_size_default: 50,
  page_size_min: 20,
  page_size_max: 200,
  export_row_limit: 100000,
  row_scope_mode: 'by_profile',
  keyset_column: '',
  keyset_dir: 'asc',
  row_scope_region_column: '',
  row_scope_department_column: '',
})

watch(
  detail,
  (d) => {
    if (!d) return
    columns.value = d.columns.map((c) => ({ ...c }))
    roleCodes.value = [...d.role_codes]
    Object.assign(meta, {
      name: d.name,
      slug: d.slug,
      description: d.description,
      page_size_default: d.page_size_default,
      page_size_min: d.page_size_min,
      page_size_max: d.page_size_max,
      export_row_limit: d.export_row_limit,
      row_scope_mode: d.row_scope_mode,
      keyset_column: d.keyset_column,
      keyset_dir: d.keyset_dir,
      row_scope_region_column: d.row_scope_region_column,
      row_scope_department_column: d.row_scope_department_column,
    })
  },
  { immediate: true },
)

const colNames = computed(() => columns.value.map((c) => c.source_name))
const displayTypeOptions = DISPLAY_TYPES.map((t) => ({ label: t, value: t }))
const sortDirOptions = [
  { label: 'по возрастанию', value: 'asc' },
  { label: 'по убыванию', value: 'desc' },
]

// --- сохранение секций ---
const msg = ref<{ severity: string; text: string } | null>(null)
const saving = ref('')

async function section(kind: string, fn: () => Promise<unknown>, ok: string) {
  msg.value = null
  saving.value = kind
  try {
    await fn()
    await refresh()
    msg.value = { severity: 'success', text: ok }
  } catch (e) {
    msg.value = { severity: 'error', text: apiErrorMessage(e) }
  } finally {
    saving.value = ''
  }
}

const saveColumns = () =>
  section('columns', () => admin.views.updateColumns(id.value, columnsToPayload(columns.value)), 'Колонки сохранены')
const savePermissions = () =>
  section('perms', () => admin.views.setPermissions(id.value, roleCodes.value), 'Права сохранены')
const saveMeta = () =>
  section(
    'meta',
    () =>
      admin.views.updateMeta(id.value, {
        name: meta.name,
        slug: meta.slug,
        description: meta.description || undefined,
        page_size_default: meta.page_size_default,
        page_size_min: meta.page_size_min,
        page_size_max: meta.page_size_max,
        export_row_limit: meta.export_row_limit,
        row_scope_mode: meta.row_scope_mode,
        keyset_column: meta.keyset_column || undefined,
        keyset_dir: meta.keyset_dir || undefined,
        row_scope_region_column: meta.row_scope_region_column || undefined,
        row_scope_department_column: meta.row_scope_department_column || undefined,
      }),
    'Параметры сохранены',
  )
const publish = () => section('publish', () => admin.views.publish(id.value), 'Витрина опубликована')
const disable = () => section('disable', () => admin.views.disable(id.value), 'Витрина отключена')
async function remove() {
  if (!confirm('Удалить витрину?')) return
  try {
    await admin.views.remove(id.value)
    await navigateTo('/reporter/admin/views')
  } catch (e) {
    msg.value = { severity: 'error', text: apiErrorMessage(e) }
  }
}

// --- предпросмотр ---
const preview = ref<QueryResult | null>(null)
const previewing = ref(false)
async function doPreview() {
  previewing.value = true
  msg.value = null
  try {
    preview.value = await admin.views.preview(id.value, { page_size: 20 })
  } catch (e) {
    msg.value = { severity: 'error', text: apiErrorMessage(e) }
  } finally {
    previewing.value = false
  }
}
</script>

<template>
  <div>
    <PageHeader :title="detail?.name || 'Витрина'" :description="detail ? `/${detail.slug}` : undefined">
      <template #actions>
        <Button label="К списку" icon="pi pi-arrow-left" text @click="navigateTo('/reporter/admin/views')" />
      </template>
    </PageHeader>

    <div v-if="pending" class="center"><ProgressSpinner style="width: 2.5rem; height: 2.5rem" /></div>
    <ErrorState v-else-if="error || !detail" message="Не удалось загрузить витрину." retryable @retry="refresh" />

    <template v-else>
      <Message v-if="msg" :severity="msg.severity" :closable="true" class="mb">{{ msg.text }}</Message>

      <!-- Источник -->
      <Card class="mb">
        <template #content>
          <div class="source-row">
            <div><span class="k">Подключение</span><span class="v">{{ detail.data_source_id.slice(0, 8) }}…</span></div>
            <div><span class="k">База</span><span class="v">{{ detail.database }}</span></div>
            <div><span class="k">Таблица</span><span class="v">{{ detail.table }}</span></div>
            <div>
              <span class="k">Статус</span>
              <Tag :severity="statusMeta(detail.status).severity" :value="statusMeta(detail.status).label" />
            </div>
          </div>
        </template>
      </Card>

      <!-- Колонки -->
      <Card class="mb">
        <template #title><div class="card-title">Колонки<Button label="Сохранить колонки" icon="pi pi-save" size="small" :loading="saving === 'columns'" @click="saveColumns" /></div></template>
        <template #content>
          <DataTable :value="columns" data-key="source_name" size="small" scrollable>
            <Column header="Код (ClickHouse)"><template #body="{ data: c }"><code>{{ c.source_name }}</code></template></Column>
            <Column header="Тип источника"><template #body="{ data: c }"><span class="muted">{{ c.source_type }}</span></template></Column>
            <Column header="Наименование" style="min-width: 12rem">
              <template #body="{ data: c }"><InputText v-model="c.label" class="cell-input" /></template>
            </Column>
            <Column header="Тип отображения">
              <template #body="{ data: c }"><Select v-model="c.display_type" :options="displayTypeOptions" option-label="label" option-value="value" class="cell-input" /></template>
            </Column>
            <Column header="Порядок" style="width: 6rem">
              <template #body="{ data: c }"><InputNumber v-model="c.position" :min="0" :useGrouping="false" class="cell-num" /></template>
            </Column>
            <Column header="Видима"><template #body="{ data: c }"><Checkbox v-model="c.visible" :binary="true" /></template></Column>
            <Column header="Поиск"><template #body="{ data: c }"><Checkbox v-model="c.searchable" :binary="true" /></template></Column>
            <Column header="Фильтр"><template #body="{ data: c }"><Checkbox v-model="c.filterable" :binary="true" /></template></Column>
            <Column header="Сортировка"><template #body="{ data: c }"><Checkbox v-model="c.sortable" :binary="true" /></template></Column>
            <Column header="Экспорт"><template #body="{ data: c }"><Checkbox v-model="c.exportable" :binary="true" /></template></Column>
          </DataTable>
        </template>
      </Card>

      <!-- Права -->
      <Card class="mb">
        <template #title><div class="card-title">Права (роли)<Button label="Сохранить права" icon="pi pi-save" size="small" :loading="saving === 'perms'" @click="savePermissions" /></div></template>
        <template #content>
          <MultiSelect
            v-model="roleCodes"
            :options="roleOptions"
            option-label="name_ru"
            option-value="code"
            placeholder="Выберите роли, которым доступна витрина"
            display="chip"
            class="full"
          />
          <p class="hint">Администратор имеет доступ всегда. Пользователь видит витрину, если у него есть хотя бы одна из ролей.</p>
        </template>
      </Card>

      <!-- Ограничение строк и параметры -->
      <Card class="mb">
        <template #title><div class="card-title">Ограничение строк и параметры<Button label="Сохранить параметры" icon="pi pi-save" size="small" :loading="saving === 'meta'" @click="saveMeta" /></div></template>
        <template #content>
          <div class="grid3">
            <div class="field">
              <label>Режим ограничения строк</label>
              <Select v-model="meta.row_scope_mode" :options="ROW_SCOPE_MODES" option-label="label" option-value="value" />
            </div>
            <div class="field">
              <label>Колонка региона (RLS)</label>
              <Select v-model="meta.row_scope_region_column" :options="colNames" show-clear placeholder="— не задано —" />
            </div>
            <div class="field">
              <label>Колонка подразделения (RLS)</label>
              <Select v-model="meta.row_scope_department_column" :options="colNames" show-clear placeholder="— не задано —" />
            </div>
            <div class="field">
              <label>Ключ пагинации (keyset)</label>
              <Select v-model="meta.keyset_column" :options="colNames" placeholder="стабильный ключ" />
            </div>
            <div class="field">
              <label>Направление ключа</label>
              <Select v-model="meta.keyset_dir" :options="sortDirOptions" option-label="label" option-value="value" />
            </div>
            <div class="field">
              <label>Размер страницы (по умолч.)</label>
              <InputNumber v-model="meta.page_size_default" :min="20" :max="200" :useGrouping="false" />
            </div>
            <div class="field">
              <label>Лимит экспорта</label>
              <InputNumber v-model="meta.export_row_limit" :min="1" :max="100000" :useGrouping="false" />
            </div>
          </div>
        </template>
      </Card>

      <!-- Предпросмотр -->
      <Card class="mb">
        <template #title><div class="card-title">Предпросмотр<Button label="Обновить предпросмотр" icon="pi pi-eye" size="small" :loading="previewing" @click="doPreview" /></div></template>
        <template #content>
          <EmptyState v-if="!preview" icon="pi pi-eye" title="Нажмите «Обновить предпросмотр»" hint="Первые строки текущей конфигурации (как администратор, без RLS)." />
          <DataTable v-else :value="preview.rows" size="small" scrollable>
            <Column v-for="col in preview.columns" :key="col.source_name" :header="col.label">
              <template #body="{ data }">{{ formatCell(data[col.source_name], col.display_type) }}</template>
            </Column>
          </DataTable>
        </template>
      </Card>

      <!-- Действия -->
      <div class="publish-bar">
        <Button label="Опубликовать" icon="pi pi-check-circle" severity="success" :loading="saving === 'publish'" @click="publish" />
        <Button v-if="detail.status === 'published'" label="Отключить" icon="pi pi-ban" severity="warn" outlined :loading="saving === 'disable'" @click="disable" />
        <Button label="Удалить" icon="pi pi-trash" severity="danger" outlined @click="remove" />
      </div>
    </template>
  </div>
</template>

<style scoped>
.center { display: flex; justify-content: center; padding: 2.5rem 0; }
.mb { margin-bottom: 1rem; }
.card-title { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
.source-row { display: flex; flex-wrap: wrap; gap: 2rem; }
.source-row .k { display: block; font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--ehd-muted); }
.source-row .v { font-weight: 600; }
.cell-input { width: 100%; }
.cell-num :deep(.p-inputtext) { width: 5rem; }
.muted { color: var(--p-text-muted-color); font-size: 0.85rem; }
.full { width: 100%; max-width: 640px; }
.hint { font-size: 0.82rem; color: var(--p-text-muted-color); margin: 0.5rem 0 0; }
.grid3 { display: grid; grid-template-columns: repeat(3, 1fr); gap: 1rem; }
.field { display: flex; flex-direction: column; gap: 0.35rem; }
.field label { font-size: 0.82rem; font-weight: 500; color: var(--ehd-ink-2); }
.field :deep(.p-select), .field :deep(.p-inputnumber) { width: 100%; }
.publish-bar { display: flex; gap: 0.75rem; padding: 1rem 0; }
</style>
