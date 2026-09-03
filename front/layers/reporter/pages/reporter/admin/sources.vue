<script setup lang="ts">
import type { DataSourceSummary } from '~~/shared/api/types'
import {
  PROTOCOLS,
  emptySourceForm,
  defaultPortFor,
  sourceStatusMeta,
  validateSourceForm,
  sourcePayload,
  type SourceForm,
} from '../../../utils/sourceForm'

definePageMeta({ middleware: ['auth', 'admin'] })

const admin = useReporterAdmin()

const { data, pending, error, refresh } = await useAsyncData('admin-sources', () => admin.sources.list())
const items = computed<DataSourceSummary[]>(() => data.value?.items ?? [])
// Система рассчитана на один источник (backend: второй → SOURCE_ALREADY_EXISTS).
const hasSource = computed(() => items.value.length > 0)

// Верхний баннер для результатов действий по строке (тест/активация).
const actionMsg = ref<{ severity: string; text: string } | null>(null)
const busyId = ref('')

// --- диалог создания/редактирования ---
const dialog = ref(false)
const saving = ref(false)
const testing = ref(false)
const editId = ref('')
const formError = ref('')
const testMsg = ref<{ severity: string; text: string } | null>(null)
const form = reactive<SourceForm>(emptySourceForm())
const isEdit = computed(() => !!editId.value)

function assign(f: SourceForm) {
  Object.assign(form, f)
}

function openCreate() {
  editId.value = ''
  assign(emptySourceForm())
  formError.value = ''
  testMsg.value = null
  dialog.value = true
}

function openEdit(s: DataSourceSummary) {
  editId.value = s.id
  assign({
    code: s.code,
    name: s.name,
    host: s.host,
    port: s.port,
    protocol: s.protocol,
    tls_enabled: s.tls_enabled,
    tls_skip_verify: s.tls_skip_verify,
    username: s.username,
    password: '',
  })
  formError.value = ''
  testMsg.value = null
  dialog.value = true
}

// Смена протокола подставляет стандартный порт, если он совпадает с дефолтом другого протокола.
function onProtocolChange() {
  const known = form.port === 8123 || form.port === 9000
  if (known) form.port = defaultPortFor(form.protocol)
}

async function testConnection() {
  const msg = validateSourceForm(form, isEdit.value)
  if (msg) {
    testMsg.value = { severity: 'warn', text: msg }
    return
  }
  testing.value = true
  testMsg.value = null
  try {
    await admin.sources.testParams(sourcePayload(form))
    testMsg.value = { severity: 'success', text: 'Подключение успешно' }
  } catch (e) {
    testMsg.value = { severity: 'error', text: apiErrorMessage(e) }
  } finally {
    testing.value = false
  }
}

async function save() {
  const msg = validateSourceForm(form, isEdit.value)
  if (msg) {
    formError.value = msg
    return
  }
  saving.value = true
  formError.value = ''
  try {
    if (editId.value) await admin.sources.update(editId.value, sourcePayload(form))
    else await admin.sources.create(sourcePayload(form))
    dialog.value = false
    actionMsg.value = { severity: 'success', text: editId.value ? 'Источник обновлён' : 'Источник создан' }
    await refresh()
  } catch (e) {
    formError.value = apiErrorMessage(e)
  } finally {
    saving.value = false
  }
}

async function testSaved(s: DataSourceSummary) {
  busyId.value = s.id
  actionMsg.value = null
  try {
    await admin.sources.test(s.id)
    actionMsg.value = { severity: 'success', text: `Связь с «${s.name}» успешна` }
  } catch (e) {
    actionMsg.value = { severity: 'error', text: apiErrorMessage(e) }
  } finally {
    busyId.value = ''
  }
}

async function toggleActive(s: DataSourceSummary) {
  busyId.value = s.id
  actionMsg.value = null
  try {
    if (s.status === 'active') await admin.sources.deactivate(s.id)
    else await admin.sources.activate(s.id)
    await refresh()
  } catch (e) {
    actionMsg.value = { severity: 'error', text: apiErrorMessage(e) }
  } finally {
    busyId.value = ''
  }
}
</script>

<template>
  <div>
    <PageHeader title="Источники данных" description="Подключения Reporter к ClickHouse (read-only). Поддерживается один источник.">
      <template #actions>
        <span v-tooltip.left="hasSource ? 'Уже настроен источник — поддерживается один' : undefined">
          <Button label="Добавить источник" icon="pi pi-plus" :disabled="hasSource" @click="openCreate" />
        </span>
      </template>
    </PageHeader>

    <Message v-if="error" severity="error" :closable="false" class="mb">Не удалось загрузить источники</Message>
    <Message v-if="actionMsg" :severity="actionMsg.severity" :closable="true" class="mb" @close="actionMsg = null">
      {{ actionMsg.text }}
    </Message>

    <div v-if="!pending && !hasSource" class="empty-cta">
      <i class="pi pi-database" />
      <p>Источник данных ещё не настроен.</p>
      <Button label="Добавить источник" icon="pi pi-plus" @click="openCreate" />
    </div>

    <DataTable v-else :value="items" :loading="pending" data-key="id" size="small">
      <Column header="Источник">
        <template #body="{ data: s }">
          <div class="src-name"><i class="pi pi-database" /> {{ s.name }}</div>
          <div class="src-code">{{ s.code }}</div>
        </template>
      </Column>
      <Column header="Подключение">
        <template #body="{ data: s }">
          <code>{{ s.protocol }}://{{ s.host }}:{{ s.port }}</code>
          <Tag v-if="s.tls_enabled" severity="info" value="TLS" class="tls-tag" />
        </template>
      </Column>
      <Column header="Пользователь"><template #body="{ data: s }">{{ s.username }}</template></Column>
      <Column header="Статус">
        <template #body="{ data: s }">
          <Tag :severity="sourceStatusMeta(s.status).severity" :value="sourceStatusMeta(s.status).label" />
        </template>
      </Column>
      <Column header="" style="width: 15rem">
        <template #body="{ data: s }">
          <div class="row-actions">
            <Button
              icon="pi pi-bolt"
              text
              rounded
              size="small"
              severity="secondary"
              :loading="busyId === s.id"
              v-tooltip.top="'Проверить связь'"
              aria-label="Проверить связь"
              @click="testSaved(s)"
            />
            <Button
              :icon="s.status === 'active' ? 'pi pi-pause' : 'pi pi-play'"
              text
              rounded
              size="small"
              :severity="s.status === 'active' ? 'warn' : 'success'"
              :loading="busyId === s.id"
              v-tooltip.top="s.status === 'active' ? 'Деактивировать' : 'Активировать'"
              :aria-label="s.status === 'active' ? 'Деактивировать' : 'Активировать'"
              @click="toggleActive(s)"
            />
            <Button icon="pi pi-pencil" text rounded size="small" v-tooltip.top="'Изменить'" aria-label="Изменить" @click="openEdit(s)" />
          </div>
        </template>
      </Column>
    </DataTable>

    <Dialog v-model:visible="dialog" modal :header="isEdit ? 'Изменить источник' : 'Новый источник'" :style="{ width: '560px' }">
      <div class="form">
        <div class="grid2">
          <div class="field">
            <label>Код</label>
            <InputText v-model="form.code" placeholder="ehd_bo" />
          </div>
          <div class="field">
            <label>Название</label>
            <InputText v-model="form.name" placeholder="ЕХД БО" />
          </div>
        </div>
        <div class="grid-conn">
          <div class="field">
            <label>Протокол</label>
            <Select v-model="form.protocol" :options="PROTOCOLS" option-label="label" option-value="value" @change="onProtocolChange" />
          </div>
          <div class="field">
            <label>Хост</label>
            <InputText v-model="form.host" placeholder="10.58.56.50" />
          </div>
          <div class="field">
            <label>Порт</label>
            <InputNumber v-model="form.port" :min="1" :max="65535" :useGrouping="false" />
          </div>
        </div>
        <div class="grid2">
          <div class="field">
            <label>Пользователь</label>
            <InputText v-model="form.username" placeholder="default" />
          </div>
          <div class="field">
            <label>Пароль</label>
            <Password v-model="form.password" :feedback="false" toggle-mask :placeholder="isEdit ? 'Не менять' : '••••••••'" />
            <small v-if="isEdit" class="hint">Оставьте пустым, чтобы не менять пароль.</small>
          </div>
        </div>
        <div class="switches">
          <div class="switch"><ToggleSwitch v-model="form.tls_enabled" inputId="tls" /><label for="tls">TLS</label></div>
          <div class="switch">
            <ToggleSwitch v-model="form.tls_skip_verify" inputId="tlsskip" :disabled="!form.tls_enabled" />
            <label for="tlsskip">Не проверять сертификат</label>
          </div>
        </div>

        <Message v-if="testMsg" :severity="testMsg.severity" :closable="false">{{ testMsg.text }}</Message>
        <Message v-if="formError" severity="error" :closable="false">{{ formError }}</Message>

        <div class="actions">
          <Button label="Проверить связь" icon="pi pi-bolt" severity="secondary" outlined :loading="testing" @click="testConnection" />
          <span class="spacer" />
          <Button label="Отмена" severity="secondary" text @click="dialog = false" />
          <Button :label="isEdit ? 'Сохранить' : 'Создать'" icon="pi pi-check" :loading="saving" @click="save" />
        </div>
      </div>
    </Dialog>
  </div>
</template>

<style scoped>
.mb { margin-bottom: 1rem; }
.empty-cta {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  padding: 3rem 1rem;
  border: 1px dashed var(--ehd-border);
  border-radius: var(--ehd-radius);
  color: var(--ehd-ink-2);
}
.empty-cta i { font-size: 2rem; color: var(--ehd-muted); }
.empty-cta p { margin: 0; }
.src-name { display: inline-flex; align-items: center; gap: 0.5rem; font-weight: 500; }
.src-code { font-size: 0.8rem; color: var(--ehd-muted); }
.tls-tag { margin-left: 0.5rem; }
.row-actions { display: flex; gap: 0.15rem; }
.form { display: flex; flex-direction: column; gap: 0.85rem; }
.field { display: flex; flex-direction: column; gap: 0.35rem; }
.field label { font-size: 0.82rem; font-weight: 500; color: var(--ehd-ink-2); }
.field :deep(.p-inputtext),
.field :deep(.p-select),
.field :deep(.p-inputnumber),
.field :deep(.p-password),
.field :deep(.p-password-input) { width: 100%; }
.hint { font-size: 0.76rem; color: var(--ehd-muted); }
.grid2 { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
.grid-conn { display: grid; grid-template-columns: 1fr 2fr 1fr; gap: 1rem; }
.switches { display: flex; gap: 2rem; }
.switch { display: flex; align-items: center; gap: 0.6rem; }
.switch label { font-size: 0.88rem; }
.actions { display: flex; align-items: center; gap: 0.5rem; margin-top: 0.5rem; }
.spacer { flex: 1; }
</style>
