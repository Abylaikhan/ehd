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
const fConfirmedNoDecision = ref(false) // быстрый фильтр «Подтверждено без решения» (EVGA-FR-082)
const fDepartment = ref<number | null>(null)

// --- колоночный поиск (спека 019): отдельная строка под шапкой таблицы ---
const cPpoPp = ref('')
const cFio = ref('')
const cLa1 = ref('')
const cStatusNote = ref('')
const cOutNum = ref('')
// применяется с задержкой, чтобы не дёргать сервер на каждую букву
let colTimer: ReturnType<typeof setTimeout> | undefined
function applyColumnSearch() {
  clearTimeout(colTimer)
  colTimer = setTimeout(applyFilters, 400)
}
// PrimeVue рисует строку фильтров по этому объекту; сам поиск серверный (lazy),
// поэтому значения держим в собственных ref-ах, а здесь только ключи колонок.
const colFilters = ref<Record<string, { value: null }>>({
  profile_title: { value: null }, gu: { value: null }, sendername: { value: null },
  ppo_pp: { value: null }, paymentdate: { value: null }, iin: { value: null },
  fio: { value: null }, la1: { value: null }, amount_part: { value: null },
  status_title: { value: null }, status_note: { value: null },
  notice_num: { value: null }, out_num: { value: null },
})

// Светофор контрольного срока: красный — истёк, жёлтый — истекает, зелёный — в сроке.
// Для статусов без срока (решение принято) индикатор не показываем.
function lightOf(r: EvgaRecord): { cls: string; title: string } | null {
  if (r.deadline_state === 'expired') return { cls: 'lamp-red', title: 'Срок истёк' }
  if (r.deadline_state === 'expiring') return { cls: 'lamp-amber', title: 'Срок истекает' }
  if (r.exec_due) return { cls: 'lamp-green', title: 'В пределах срока' }
  return null
}

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
    confirmed_no_decision: fConfirmedNoDecision.value ? true : null,
    department_id: fDepartment.value,
    // колоночный поиск (спека 019)
    ppo_pp: cPpoPp.value.trim(),
    fio: cFio.value.trim(),
    la1: cLa1.value.trim(),
    status_note: cStatusNote.value.trim(),
    out_num: cOutNum.value.trim(),
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
  fConfirmedNoDecision.value = false
  fDepartment.value = null
  cPpoPp.value = cFio.value = cLa1.value = cStatusNote.value = cOutNum.value = ''
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

// Подсветка строк по контрольному сроку (EVGA-FR-080); состояние приходит с сервера.
const rowClass = (r: EvgaRecord) =>
  r.deadline_state === 'expired' ? 'row-expired' : r.deadline_state === 'expiring' ? 'row-expiring' : ''
</script>

<template>
  <div>
    <PageHeader title="ОБМ ЭВГА — Реестр рисков 5-15а" description="Витрина результатов онлайн-бюджетного мониторинга. Режим чтения.">
      <template #actions>
        <Tag v-if="scopeLabel" :value="scopeLabel" severity="info" />
      </template>
    </PageHeader>

    <Card>
      <template #content>
        <ErrorState v-if="screenState === 'denied'" title="Доступ запрещён" message="У вас нет роли модуля ОБМ ЭВГА." />
        <ErrorState v-else-if="screenState === 'source'" title="Источник недоступен" message="База данных ОБМ ЭВГА временно недоступна. Повторите позже." retryable @retry="refresh" />
        <ErrorState v-else-if="screenState === 'error'" message="Не удалось загрузить реестр." retryable @retry="refresh" />
        <div v-else-if="screenState === 'loading'" class="center">
          <ProgressSpinner style="width: 2.5rem; height: 2.5rem" />
        </div>

        <template v-else>
          <Message v-if="scope?.unmapped" severity="warn" :closable="false" class="scope-warn">
            Пользователь не сопоставлен с департаментом ДВГА: ИИН не найден в системе ОБМ ЭВГА либо не подтверждён.
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
            <ToggleButton
              v-model="fConfirmedNoDecision"
              on-label="Подтверждено без решения"
              off-label="Подтверждено без решения"
              on-icon="pi pi-check"
              off-icon="pi pi-filter"
              class="f-toggle"
              @update:model-value="applyFilters"
            />
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
            show-gridlines
            size="small"
            scrollable
            scroll-height="62vh"
            filter-display="row"
            :filters="colFilters"
            class="data-table compact"
            :row-class="rowClass"
            @sort="onSort"
          >
            <template #empty>
              <div class="empty-row">По текущему запросу строк не найдено.</div>
            </template>
            <Column v-if="canWrite" selection-mode="multiple" :style="{ width: '2.5rem' }" />
            <Column header="№" :style="{ width: '3.2rem' }">
              <template #body="{ index }">{{ first + index + 1 }}</template>
            </Column>
            <!-- светофор по контрольному сроку -->
            <Column header="" :style="{ width: '2.2rem' }">
              <template #body="{ data: r }">
                <span v-if="lightOf(r)" class="lamp" :class="lightOf(r)!.cls" :title="lightOf(r)!.title" />
                <span v-else class="lamp lamp-off" title="Срок не применяется" />
              </template>
            </Column>
            <Column field="profile_title" header="Профиль риска" :style="{ minWidth: '12rem' }">
              <template #filter>
                <Select v-model="fProfile" :options="refOptions(refs?.profiles)" option-label="label" option-value="value"
                  placeholder="все" show-clear filter size="small" class="col-f" @change="applyFilters" />
              </template>
            </Column>
            <Column field="gu" header="Код ГУ">
              <template #filter>
                <InputText v-model="fGu" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <Column field="sendername" header="Отправитель" :style="{ minWidth: '14rem' }">
              <template #filter>
                <InputText v-model="fSender" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <Column field="ppo_pp" header="№ платежа">
              <template #filter>
                <InputText v-model="cPpoPp" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <Column field="paymentdate" header="Дата платежа" sortable>
              <template #body="{ data: r }">{{ fmtDate(r.paymentdate) }}</template>
              <template #filter>
                <DatePicker v-model="fDateFrom" placeholder="с" date-format="dd.mm.yy" size="small" class="col-f" @update:model-value="applyFilters" />
              </template>
            </Column>
            <Column field="iin" header="ИИН получателя">
              <template #filter>
                <InputText v-model="fIin" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <Column field="fio" header="ФИО получателя" :style="{ minWidth: '11rem' }">
              <template #body="{ data: r }">{{ fio(r) }}</template>
              <template #filter>
                <InputText v-model="cFio" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <Column field="la1" header="Карт-счёт">
              <template #filter>
                <InputText v-model="cLa1" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <Column field="amount_part" header="Сумма" sortable class="num">
              <template #body="{ data: r }">{{ fmtAmount(r.amount_part) }}</template>
              <template #filter>
                <InputNumber v-model="fAmountFrom" placeholder="от" size="small" class="col-f" @update:model-value="applyColumnSearch" />
              </template>
            </Column>
            <Column field="status_title" header="Статус">
              <template #filter>
                <Select v-model="fStatus" :options="refOptions(refs?.statuses)" option-label="label" option-value="value"
                  placeholder="все" show-clear size="small" class="col-f" @change="applyFilters" />
              </template>
            </Column>
            <Column field="status_note" header="Комментарий" :style="{ maxWidth: '12rem' }">
              <template #filter>
                <InputText v-model="cStatusNote" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <Column field="notice_num" header="Уведомление">
              <template #body="{ data: r }">{{ r.notice_num || '—' }}</template>
              <template #filter>
                <InputText v-model="fNoticeNum" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <Column field="out_num" header="Исходящее">
              <template #body="{ data: r }">{{ r.out_num || '—' }}</template>
              <template #filter>
                <InputText v-model="cOutNum" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <Column header="Срок" :style="{ minWidth: '9.5rem' }">
              <template #body="{ data: r }">
                <span v-if="r.exec_due" class="due-cell">
                  {{ fmtDate(r.exec_due) }}
                  <Tag v-if="r.deadline_state === 'expired'" value="истёк" severity="danger" />
                  <Tag v-else-if="r.deadline_state === 'expiring'" value="истекает" severity="warn" />
                </span>
                <span v-else>—</span>
              </template>
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
/* Компактная панель фильтров: уменьшенные поля и кегль (запрос по UI). */
.filters { display: flex; flex-wrap: wrap; gap: 0.35rem; margin-bottom: 0.75rem; font-size: 0.82rem; }
.filters :deep(.p-inputtext),
.filters :deep(.p-select-label),
.filters :deep(.p-togglebutton) { font-size: 0.82rem; padding-top: 0.3rem; padding-bottom: 0.3rem; }
.f-narrow { width: 5.5rem; }
.f-year :deep(input) { width: 5.5rem; }
.f-mid { width: 10rem; }
.f-mid :deep(input) { width: 100%; }
.f-wide { width: 15rem; }
.f-actions { display: flex; gap: 0.4rem; align-items: center; }
.toolbar { display: flex; align-items: center; margin-bottom: 0.75rem; }
.count { font-size: 0.9rem; color: var(--ehd-ink-2); }
.count b { color: var(--ehd-ink); }
.toolbar-export { margin-left: auto; }
.action-error { margin-bottom: 1rem; }
.data-table { border: 1px solid var(--ehd-border); border-radius: var(--ehd-radius-sm); overflow: hidden; }
.empty-row { padding: 1.5rem; text-align: center; color: var(--p-text-muted-color); }
.num { text-align: right; }
.f-toggle { white-space: nowrap; }
.due-cell { display: inline-flex; align-items: center; gap: 0.4rem; white-space: nowrap; }

/* Компактная таблица: мельче кегль и плотнее ячейки (запрос по UI). */
.compact :deep(.p-datatable-tbody > tr > td),
.compact :deep(.p-datatable-thead > tr > th) { padding: 0.3rem 0.45rem; font-size: 0.8rem; }
/* Шапка и строка фильтров закреплены при прокрутке длинного списка. */
.compact :deep(.p-datatable-thead > tr > th) { position: sticky; z-index: 2; background: var(--p-datatable-header-cell-background); }
.col-f { width: 100%; min-width: 5rem; }
.col-f :deep(input) { width: 100%; font-size: 0.78rem; padding: 0.2rem 0.35rem; }
.col-f :deep(.p-select-label) { font-size: 0.78rem; padding: 0.2rem 0.35rem; }

/* Светофор контрольного срока */
.lamp { display: inline-block; width: 0.7rem; height: 0.7rem; border-radius: 50%; border: 1px solid rgb(0 0 0 / 12%); }
.lamp-red { background: var(--p-red-500); }
.lamp-amber { background: var(--p-amber-400); }
.lamp-green { background: var(--p-green-500); }
.lamp-off { background: var(--p-surface-300); }
/* Подсветка строк по контрольному сроку (EVGA-FR-080). td-фон перекрывает striped-rows. */
:deep(tr.row-expired > td) { background: var(--p-red-50); }
:deep(tr.row-expiring > td) { background: var(--p-orange-50); }
.report-line { margin: 0 0 0.75rem; font-size: 0.95rem; }
.report-table { border: 1px solid var(--ehd-border); border-radius: var(--ehd-radius-sm); overflow: hidden; }
</style>
