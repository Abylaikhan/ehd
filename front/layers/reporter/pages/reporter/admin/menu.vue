<script setup lang="ts">
import type { MenuItem, ViewSummary, Role, MenuItemPayload } from '~~/shared/api/types'

definePageMeta({ middleware: ['auth', 'admin'] })

const admin = useReporterAdmin()

const { data: menuData, pending, error, refresh } = await useAsyncData('admin-menu', () => admin.menu.list())
const { data: viewsData } = await useAsyncData('admin-menu-views', () => admin.views.list())
const { data: rolesData } = await useAsyncData('admin-menu-roles', () => admin.roles())

const items = computed<MenuItem[]>(() => menuData.value?.items ?? [])
const views = computed<ViewSummary[]>(() => viewsData.value?.items ?? [])
const roleOptions = computed<Role[]>(() => rolesData.value?.items ?? [])

const nameById = computed<Record<string, string>>(() => {
  const m: Record<string, string> = {}
  for (const it of items.value) m[it.id] = it.name_ru
  return m
})
const viewNameById = computed<Record<string, string>>(() => {
  const m: Record<string, string> = {}
  for (const v of views.value) m[v.id] = v.name
  return m
})
const parentOptions = computed(() =>
  items.value.filter((it) => it.id !== form.id).map((it) => ({ label: it.name_ru, value: it.id })),
)
const viewOptions = computed(() => views.value.map((v) => ({ label: `${v.name} (${v.slug})`, value: v.id })))

// --- диалог создания/редактирования ---
const dialog = ref(false)
const saving = ref(false)
const formError = ref('')
const linkType = ref<'section' | 'view'>('section')
const form = reactive<{
  id?: string
  name_ru: string
  name_kk: string
  parent_id: string
  data_view_id: string
  icon_key: string
  position: number
  public_access: boolean
  is_disabled: boolean
  role_codes: string[]
}>({
  id: undefined,
  name_ru: '',
  name_kk: '',
  parent_id: '',
  data_view_id: '',
  icon_key: 'pi pi-table',
  position: 0,
  public_access: false,
  is_disabled: false,
  role_codes: [],
})

function reset() {
  form.id = undefined
  form.name_ru = ''
  form.name_kk = ''
  form.parent_id = ''
  form.data_view_id = ''
  form.icon_key = 'pi pi-table'
  form.position = 0
  form.public_access = false
  form.is_disabled = false
  form.role_codes = []
  linkType.value = 'section'
  formError.value = ''
}

function openCreate() {
  reset()
  dialog.value = true
}
function openEdit(it: MenuItem) {
  reset()
  form.id = it.id
  form.name_ru = it.name_ru
  form.name_kk = it.name_kk
  form.parent_id = it.parent_id
  form.data_view_id = it.data_view_id
  form.icon_key = it.icon_key || 'pi pi-table'
  form.position = it.position
  form.public_access = it.public_access
  form.is_disabled = it.is_disabled
  form.role_codes = [...it.role_codes]
  linkType.value = it.data_view_id ? 'view' : 'section'
  dialog.value = true
}

function payloadFrom(): MenuItemPayload {
  return {
    name_ru: form.name_ru.trim(),
    name_kk: form.name_kk || undefined,
    parent_id: form.parent_id || undefined,
    data_view_id: linkType.value === 'view' ? form.data_view_id || undefined : undefined,
    icon_key: form.icon_key || undefined,
    position: form.position,
    public_access: form.public_access,
    is_disabled: form.is_disabled,
    role_codes: form.role_codes,
  }
}

async function save() {
  if (!form.name_ru.trim()) {
    formError.value = 'Наименование обязательно'
    return
  }
  if (linkType.value === 'view' && !form.data_view_id) {
    formError.value = 'Выберите витрину или переключите на «Раздел»'
    return
  }
  saving.value = true
  formError.value = ''
  try {
    if (form.id) await admin.menu.update(form.id, payloadFrom())
    else await admin.menu.create(payloadFrom())
    dialog.value = false
    await refresh()
  } catch (e) {
    formError.value = apiErrorMessage(e)
  } finally {
    saving.value = false
  }
}

const actionError = ref('')
async function remove(it: MenuItem) {
  if (!confirm(`Удалить пункт «${it.name_ru}»?`)) return
  actionError.value = ''
  try {
    await admin.menu.remove(it.id)
    await refresh()
  } catch (e) {
    actionError.value = apiErrorMessage(e)
  }
}
</script>

<template>
  <div>
    <PageHeader title="Навигация ЕХД БО" description="Дерево меню Reporter: разделы, витрины, доступ по ролям">
      <template #actions>
        <Button label="Создать пункт" icon="pi pi-plus" @click="openCreate" />
      </template>
    </PageHeader>

    <Message v-if="error" severity="error" :closable="false" class="mb">Не удалось загрузить меню</Message>
    <Message v-if="actionError" severity="error" :closable="true" class="mb">{{ actionError }}</Message>

    <DataTable :value="items" :loading="pending" data-key="id" size="small">
      <template #empty><div class="empty">Пунктов меню пока нет</div></template>
      <Column header="Наименование">
        <template #body="{ data: it }">
          <span class="name"><i :class="it.icon_key || 'pi pi-table'" /> {{ it.name_ru }}</span>
        </template>
      </Column>
      <Column header="Родитель">
        <template #body="{ data: it }">{{ it.parent_id ? nameById[it.parent_id] : '— корневой —' }}</template>
      </Column>
      <Column header="Витрина">
        <template #body="{ data: it }">
          <span v-if="it.data_view_id">{{ viewNameById[it.data_view_id] || '—' }}</span>
          <Tag v-else severity="secondary" value="раздел" />
        </template>
      </Column>
      <Column header="Доступ">
        <template #body="{ data: it }">
          <Tag v-if="it.public_access" severity="success" value="публичный" />
          <span v-else-if="it.role_codes.length" class="roles">{{ it.role_codes.join(', ') }}</span>
          <span v-else class="muted">—</span>
        </template>
      </Column>
      <Column header="Порядок"><template #body="{ data: it }">{{ it.position }}</template></Column>
      <Column header="Статус">
        <template #body="{ data: it }">
          <Tag :severity="it.is_disabled ? 'danger' : 'success'" :value="it.is_disabled ? 'выключен' : 'включён'" />
        </template>
      </Column>
      <Column header="" style="width: 6rem">
        <template #body="{ data: it }">
          <Button icon="pi pi-pencil" text rounded size="small" aria-label="Изменить" @click="openEdit(it)" />
          <Button icon="pi pi-trash" text rounded severity="danger" size="small" aria-label="Удалить" @click="remove(it)" />
        </template>
      </Column>
    </DataTable>

    <Dialog v-model:visible="dialog" modal :header="form.id ? 'Изменить пункт' : 'Новый пункт'" :style="{ width: '520px' }">
      <div class="form">
        <div class="field">
          <label>Наименование (RU)</label>
          <InputText v-model="form.name_ru" />
        </div>
        <div class="field">
          <label>Тип пункта</label>
          <Select
            v-model="linkType"
            :options="[
              { label: 'Раздел (без ссылки)', value: 'section' },
              { label: 'Ссылка на витрину', value: 'view' },
            ]"
            option-label="label"
            option-value="value"
          />
        </div>
        <div v-if="linkType === 'view'" class="field">
          <label>Витрина</label>
          <Select v-model="form.data_view_id" :options="viewOptions" option-label="label" option-value="value" filter placeholder="Выберите витрину" />
        </div>
        <div class="field">
          <label>Родительский пункт</label>
          <Select v-model="form.parent_id" :options="parentOptions" option-label="label" option-value="value" show-clear placeholder="— корневой —" />
        </div>
        <div class="grid2">
          <div class="field">
            <label>Иконка (PrimeIcons)</label>
            <InputText v-model="form.icon_key" placeholder="pi pi-table" />
          </div>
          <div class="field">
            <label>Порядок</label>
            <InputNumber v-model="form.position" :min="0" :useGrouping="false" />
          </div>
        </div>
        <div class="field">
          <label>Роли (кому виден пункт)</label>
          <MultiSelect v-model="form.role_codes" :options="roleOptions" option-label="name_ru" option-value="code" display="chip" placeholder="Роли (если не публичный)" />
        </div>
        <div class="switches">
          <div class="switch"><ToggleSwitch v-model="form.public_access" inputId="pub" /><label for="pub">Публичный (все аутентифицированные)</label></div>
          <div class="switch"><ToggleSwitch v-model="form.is_disabled" inputId="dis" /><label for="dis">Выключен</label></div>
        </div>
        <Message v-if="formError" severity="error" :closable="false">{{ formError }}</Message>
        <div class="actions">
          <Button label="Отмена" severity="secondary" text @click="dialog = false" />
          <Button :label="form.id ? 'Сохранить' : 'Создать'" icon="pi pi-check" :loading="saving" @click="save" />
        </div>
      </div>
    </Dialog>
  </div>
</template>

<style scoped>
.mb { margin-bottom: 1rem; }
.empty { padding: 2rem; text-align: center; color: var(--ehd-muted); }
.name { display: inline-flex; align-items: center; gap: 0.5rem; font-weight: 500; }
.roles { font-size: 0.85rem; color: var(--ehd-ink-2); }
.muted { color: var(--p-text-muted-color); }
.form { display: flex; flex-direction: column; gap: 0.85rem; }
.field { display: flex; flex-direction: column; gap: 0.35rem; }
.field label { font-size: 0.82rem; font-weight: 500; color: var(--ehd-ink-2); }
.field :deep(.p-inputtext), .field :deep(.p-select), .field :deep(.p-multiselect), .field :deep(.p-inputnumber) { width: 100%; }
.grid2 { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
.switches { display: flex; flex-direction: column; gap: 0.6rem; }
.switch { display: flex; align-items: center; gap: 0.6rem; }
.switch label { font-size: 0.88rem; }
.actions { display: flex; justify-content: flex-end; gap: 0.5rem; margin-top: 0.5rem; }
</style>
