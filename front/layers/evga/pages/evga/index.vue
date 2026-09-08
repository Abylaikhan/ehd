<script setup lang="ts">
import type { EvgaBulkReport, EvgaRecord, EvgaReference, EvgaRegistryResponse, EvgaStatusChange } from '~~/shared/api/types'
import type { EvgaRegistryParams } from '../../composables/useEvga'
import { manualTargets } from '../../utils/statusFlow'

// Реестр рисков 5-15а (EVGA-FR-012…015/017; спеки 005/006-evga-*-ui).
definePageMeta({ middleware: 'auth' })

const evga = useEvga()
const session = useSessionStore()

// Пишущие контролы — аудитору и админу; куратор только читает. Backend авторизует повторно.
const canWrite = computed(() => session.isAdmin || (session.user?.roles ?? []).includes('evga_auditor'))

// --- фильтры (EVGA-FR-013) ---
const fProfile = ref<number | null>(null)
const fStatus = ref<number | null>(null)
const fGod = ref<number | null>(null)
const fMes = ref<number | null>(null)
const fDateFrom = ref<Date | null>(null)
const fDateTo = ref<Date | null>(null)
const fGu = ref('')
const fSender = ref('')
const fIin = ref('')
const fAmountFrom = ref<number | null>(null)
const fAmountTo = ref<number | null>(null)
const fInNotice = ref<boolean | null>(null)
const fNoticeNum = ref('')
const fDepartment = ref<number | null>(null)

const inNoticeOptions = [
  { label: 'Все записи', value: null },
  { label: 'Включено в уведомление', value: true },
  { label: 'Не включено', value: false },
]
const mesOptions = Array.from({ length: 12 }, (_, i) => ({ label: String(i + 1), value: i + 1 }))

// --- пагинация и сортировка (backend whitelist) ---
const page = ref(1)
const pageSize = ref(20)
const sort = ref<{ column: string; dir: 'asc' | 'desc' } | null>(null)

const dateStr = (d: Date | null) => (d ? `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}` : null)

// applied — фильтры применяются кнопкой/Enter, а не на каждый ввод
const applied = ref<EvgaRegistryParams>({})
function buildParams(): EvgaRegistryParams {
  return {
    profile_id: fProfile.value,
    status_id: fStatus.value,
    god: fGod.value,
    mes: fMes.value,
    paymentdate_from: dateStr(fDateFrom.value),
    paymentdate_to: dateStr(fDateTo.value),
    gu: fGu.value.trim(),
    sendername: fSender.value.trim(),
    iin: fIin.value.trim(),
    amount_from: fAmountFrom.value,
    amount_to: fAmountTo.value,
    in_notice: fInNotice.value,
    notice_num: fNoticeNum.value.trim(),
    department_id: fDepartment.value,
  }
}
function applyFilters() {
  page.value = 1
  applied.value = buildParams()
}
function resetFilters() {
  fProfile.value = fStatus.value = fGod.value = fMes.value = null
  fDateFrom.value = fDateTo.value = null
  fGu.value = fSender.value = fIin.value = fNoticeNum.value = ''
  fAmountFrom.value = fAmountTo.value = null
  fInNotice.value = null
  fDepartment.value = null
  applyFilters()
}

// --- данные ---
const requestKey = computed(() => JSON.stringify({ ...applied.value, page: page.value, page_size: pageSize.value, sort: sort.value }))

const { data, pending, error, refresh } = useLazyAsyncData<EvgaRegistryResponse>(
  'evga-registry',
  () =>
    evga.registry({
      ...applied.value,
      page: page.value,
      page_size: pageSize.value,
      sort: sort.value?.column,
      order: sort.value?.dir,
    }),
  { watch: [requestKey], server: false },
)

const { data: refs } = useLazyAsyncData('evga-references', () => evga.references(), { server: false })

const rows = computed(() => data.value?.items ?? [])
const total = computed(() => data.value?.total ?? 0)
const scope = computed(() => data.value?.scope)
const first = computed(() => (page.value - 1) * pageSize.value)

const refOptions = (list?: EvgaReference[]) => (list ?? []).map((r) => ({ label: r.title, value: r.id }))

function onPage(e: { page: number; rows: number }) {
  page.value = e.page + 1
  pageSize.value = e.rows
}
function onSort(e: { sortField?: string | null; sortOrder?: number | null }) {
  if (e.sortField && e.sortOrder) {
    sort.value = { column: String(e.sortField), dir: e.sortOrder === 1 ? 'asc' : 'desc' }
  } else {
    sort.value = null
  }
  page.value = 1
}

// --- экспорт (EVGA-FR-017) ---
const exporting = ref(false)
const actionError = ref('')
async function doExport() {
  exporting.value = true
  actionError.value = ''
  try {
    await evga.exportRegistry({ ...applied.value, sort: sort.value?.column, order: sort.value?.dir })
  } catch (e) {
    actionError.value = apiErrorMessage(e)
  } finally {
    exporting.value = false
  }
}

// --- массовая простановка статуса (EVGA-FR-014/015/021…023; спека 006) ---
const selection = ref<EvgaRecord[]>([])
const statusDialog = ref(false)
const statusBusy = ref(false)
const statusError = ref('')
const bulkReport = ref<EvgaBulkReport | null>(null)
const reportDialog = ref(false)

// bulk: целевые — все вручную назначаемые статусы; недопустимые для конкретных записей отклонит сервер с причиной
const bulkTargets = computed<EvgaReference[]>(() => {
  const manual = new Set(manualTargets())
  return (refs.value?.statuses ?? []).filter((s) => manual.has(s.id))
})

function openStatusDialog() {
  statusError.value = ''
  statusDialog.value = true
}

// --- формирование уведомления (EVGA-FR-016; спека 007-evga-notices-ui) ---
const noticeDraft = useEvgaNoticeDraft()
function goCreateNotice() {
  noticeDraft.value = selection.value.map((r) => r.id)
  navigateTo('/evga/notices/create')
}

async function submitBulkStatus(change: EvgaStatusChange) {
  statusBusy.value = true
  statusError.value = ''
  try {
    bulkReport.value = await evga.bulkStatus(selection.value.map((r) => r.id), change)
    statusDialog.value = false
    reportDialog.value = true
    selection.value = []
    refresh()
  } catch (e) {
    statusError.value = apiErrorMessage(e)
  } finally {
    statusBusy.value = false
  }
}

// --- состояния экрана ---
const errCode = computed(() => apiErrorCode(error.value))
const screenState = computed(() => {
  switch (errCode.value) {
    case 'ACCESS_DENIED':
      return 'denied'
    case 'EVGA_SOURCE_UNAVAILABLE':
      return 'source'
  }
  if (error.value) return 'error'
  if (!data.value && pending.value) return 'loading'
  return 'ready'
})

const scopeLabel = computed(() => {
  const s = scope.value
  if (!s) return ''
  if (s.unmapped) return ''
  if (s.all_departments) return 'Все департаменты'
  return s.department_title || ''
})

const fmtAmount = (v: string) => {
  const n = Number(v)
  return Number.isFinite(n) ? n.toLocaleString('ru-RU', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : v
}
const fmtDate = (v: string | null) => {
  if (!v) return '—'
  const [y, m, d] = v.split('-')
  return `${d}.${m}.${y}`
}
const fio = (r: { fm: string; nm: string; ft: string }) => [r.fm, r.nm, r.ft].filter(Boolean).join(' ')
</script>

<template>
  <div>
    <PageHeader title="ОБМ ЕВГА — Реестр рисков 5-15а" description="Витрина результатов онлайн-бюджетного мониторинга. Режим чтения.">
      <template #actions>
        <Tag v-if="scopeLabel" :value="scopeLabel" severity="info" />
      </template>
    </PageHeader>

    <Card>
      <template #content>
        <ErrorState v-if="screenState === 'denied'" title="Доступ запрещён" message="У вас нет роли модуля ОБМ ЕВГА." />
        <ErrorState v-else-if="screenState === 'source'" title="Источник недоступен" message="База данных ОБМ ЕВГА временно недоступна. Повторите позже." retryable @retry="refresh" />
        <ErrorState v-else-if="screenState === 'error'" message="Не удалось загрузить реестр." retryable @retry="refresh" />
        <div v-else-if="screenState === 'loading'" class="center">
          <ProgressSpinner style="width: 2.5rem; height: 2.5rem" />
        </div>

        <template v-else>
          <Message v-if="scope?.unmapped" severity="warn" :closable="false" class="scope-warn">
            Пользователь не сопоставлен с департаментом ДВГА: ИИН не найден в системе ОБМ ЕВГА либо не подтверждён.
            Обратитесь к администратору.
          </Message>

          <div class="filters">
            <Select v-model="fProfile" :options="refOptions(refs?.profiles)" option-label="label" option-value="value" placeholder="Профиль риска" show-clear filter class="f-wide" />
            <Select v-model="fStatus" :options="refOptions(refs?.statuses)" option-label="label" option-value="value" placeholder="Статус" show-clear class="f-mid" />
            <Select v-if="scope?.all_departments" v-model="fDepartment" :options="refOptions(refs?.departments)" option-label="label" option-value="value" placeholder="Департамент" show-clear filter class="f-wide" />
            <InputNumber v-model="fGod" placeholder="Год" :use-grouping="false" :min="2020" :max="2100" class="f-year" />
            <Select v-model="fMes" :options="mesOptions" option-label="label" option-value="value" placeholder="Месяц" show-clear class="f-narrow" />
            <DatePicker v-model="fDateFrom" placeholder="Дата платежа с" date-format="dd.mm.yy" show-icon class="f-mid" />
            <DatePicker v-model="fDateTo" placeholder="по" date-format="dd.mm.yy" show-icon class="f-mid" />
            <InputText v-model="fGu" placeholder="Код ГУ" class="f-mid" @keyup.enter="applyFilters" />
            <InputText v-model="fSender" placeholder="Наименование отправителя" class="f-wide" @keyup.enter="applyFilters" />
            <InputText v-model="fIin" placeholder="ИИН получателя" class="f-mid" @keyup.enter="applyFilters" />
            <InputNumber v-model="fAmountFrom" placeholder="Сумма от" :min-fraction-digits="0" :max-fraction-digits="2" class="f-mid" />
            <InputNumber v-model="fAmountTo" placeholder="Сумма до" :min-fraction-digits="0" :max-fraction-digits="2" class="f-mid" />
            <Select v-model="fInNotice" :options="inNoticeOptions" option-label="label" option-value="value" placeholder="Уведомление" class="f-mid" />
            <InputText v-model="fNoticeNum" placeholder="№ уведомления" class="f-mid" @keyup.enter="applyFilters" />
            <div class="f-actions">
              <Button label="Применить" icon="pi pi-filter" @click="applyFilters" />
              <Button label="Сбросить" icon="pi pi-filter-slash" text severity="secondary" @click="resetFilters" />
            </div>
          </div>

          <div class="toolbar">
            <span class="count">Записей: <b>{{ total.toLocaleString('ru-RU') }}</b></span>
            <Button
              v-if="canWrite"
              :label="selection.length ? `Сменить статус (${selection.length})` : 'Сменить статус'"
              icon="pi pi-pencil"
              severity="warn"
              :disabled="selection.length === 0"
              @click="openStatusDialog"
            />
            <Button
              v-if="canWrite"
              :label="selection.length ? `Сформировать уведомление (${selection.length})` : 'Сформировать уведомление'"
              icon="pi pi-envelope"
              :disabled="selection.length === 0"
              @click="goCreateNotice"
            />
            <Button class="toolbar-export" label="Экспорт в Excel" icon="pi pi-download" :loading="exporting" :disabled="rows.length === 0" @click="doExport" />
          </div>
          <Message v-if="actionError" severity="error" :closable="true" class="action-error">{{ actionError }}</Message>

          <DataTable
            v-model:selection="selection"
            :value="rows"
            :loading="pending"
            lazy
            data-key="id"
            :sort-field="sort?.column"
            :sort-order="sort ? (sort.dir === 'asc' ? 1 : -1) : 0"
            removable-sort
            striped-rows
            size="small"
            scrollable
            class="data-table"
            @sort="onSort"
          >
            <template #empty>
              <div class="empty-row">По текущему запросу строк не найдено.</div>
            </template>
            <Column v-if="canWrite" selection-mode="multiple" :style="{ width: '2.5rem' }" />
            <Column header="№" :style="{ width: '3.5rem' }">
              <template #body="{ index }">{{ first + index + 1 }}</template>
            </Column>
            <Column field="profile_title" header="Профиль риска" :style="{ minWidth: '14rem' }" />
            <Column field="gu" header="Код ГУ" />
            <Column field="sendername" header="Отправитель" :style="{ minWidth: '16rem' }" />
            <Column field="ppo_pp" header="№ платежа" />
            <Column field="paymentdate" header="Дата платежа" sortable>
              <template #body="{ data: r }">{{ fmtDate(r.paymentdate) }}</template>
            </Column>
            <Column field="iin" header="ИИН получателя" />
            <Column header="ФИО получателя" :style="{ minWidth: '12rem' }">
              <template #body="{ data: r }">{{ fio(r) }}</template>
            </Column>
            <Column field="la1" header="Карт-счёт" />
            <Column field="amount_part" header="Сумма" sortable class="num">
              <template #body="{ data: r }">{{ fmtAmount(r.amount_part) }}</template>
            </Column>
            <Column field="status_title" header="Статус" />
            <Column field="status_note" header="Комментарий" :style="{ maxWidth: '14rem' }" />
            <Column header="Уведомление">
              <template #body="{ data: r }">{{ r.notice_num || '—' }}</template>
            </Column>
            <Column header="Исходящее">
              <template #body="{ data: r }">{{ r.out_num || '—' }}</template>
            </Column>
            <Column :style="{ width: '3rem' }">
              <template #body="{ data: r }">
                <NuxtLink :to="`/evga/${r.id}`" aria-label="Карточка записи">
                  <Button icon="pi pi-eye" text rounded size="small" />
                </NuxtLink>
              </template>
            </Column>
          </DataTable>

          <Paginator
            :rows="pageSize"
            :total-records="total"
            :first="first"
            :rows-per-page-options="[20, 50, 100]"
            @page="onPage"
          />
        </template>
      </template>
    </Card>

    <EvgaStatusDialog
      v-model:visible="statusDialog"
      :title="`Сменить статус: выбрано ${selection.length}`"
      :targets="bulkTargets"
      :activities="refs?.activities ?? []"
      :busy="statusBusy"
      :error="statusError"
      @submit="submitBulkStatus"
    />

    <Dialog v-model:visible="reportDialog" modal header="Результат массовой простановки" :style="{ width: '34rem' }">
      <template v-if="bulkReport">
        <p class="report-line">
          Обработано: <b>{{ bulkReport.processed }}</b> · Отклонено: <b>{{ bulkReport.rejected }}</b>
          <span v-if="bulkReport.cascaded"> · Обновлено по тому же платежу (другие профили): <b>{{ bulkReport.cascaded }}</b></span>
        </p>
        <DataTable v-if="bulkReport.rejections.length" :value="bulkReport.rejections" size="small" striped-rows class="report-table">
          <Column field="id" header="Запись" :style="{ width: '8rem' }" />
          <Column field="reason" header="Причина отклонения" />
        </DataTable>
      </template>
      <template #footer>
        <Button label="Закрыть" @click="reportDialog = false" />
      </template>
    </Dialog>
  </div>
</template>

<style scoped>
.center { display: flex; justify-content: center; padding: 2.5rem 0; }
.scope-warn { margin-bottom: 1rem; }
.filters { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-bottom: 1rem; }
.f-narrow { width: 6.5rem; }
.f-year :deep(input) { width: 6.5rem; }
.f-mid { width: 12rem; }
.f-mid :deep(input) { width: 100%; }
.f-wide { width: 18rem; }
.f-actions { display: flex; gap: 0.4rem; align-items: center; }
.toolbar { display: flex; align-items: center; margin-bottom: 0.75rem; }
.count { font-size: 0.9rem; color: var(--ehd-ink-2); }
.count b { color: var(--ehd-ink); }
.toolbar-export { margin-left: auto; }
.action-error { margin-bottom: 1rem; }
.data-table { border: 1px solid var(--ehd-border); border-radius: var(--ehd-radius-sm); overflow: hidden; }
.empty-row { padding: 1.5rem; text-align: center; color: var(--p-text-muted-color); }
.num { text-align: right; }
.report-line { margin: 0 0 0.75rem; font-size: 0.95rem; }
.report-table { border: 1px solid var(--ehd-border); border-radius: var(--ehd-radius-sm); overflow: hidden; }
</style>
