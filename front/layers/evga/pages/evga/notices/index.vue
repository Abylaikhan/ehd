<script setup lang="ts">
// Список уведомлений (спека 007-evga-notices-ui; backend 009 FR-7).
definePageMeta({ middleware: 'auth' })

const evga = useEvga()

const page = ref(1)
const pageSize = ref(20)
const fDocnum = ref('')

// --- колоночный поиск (спека 019 FR-1/FR-2) ---
const cGu = ref('')
const cSender = ref('')
const cDept = ref('')
const cRecipient = ref('')
const cStatus = ref('')
const applied = ref<Record<string, string>>({})

function buildFilter() {
  return {
    docnum: fDocnum.value.trim(),
    gu: cGu.value.trim(),
    sendername: cSender.value.trim(),
    department: cDept.value.trim(),
    recipient: cRecipient.value.trim(),
    status_title: cStatus.value.trim(),
  }
}
function applyFilter() {
  page.value = 1
  applied.value = buildFilter()
}
// ввод применяется с задержкой, чтобы не дёргать сервер на каждую букву
let colTimer: ReturnType<typeof setTimeout> | undefined
function applyColumnSearch() {
  clearTimeout(colTimer)
  colTimer = setTimeout(applyFilter, 400)
}

const requestKey = computed(() => JSON.stringify({ page: page.value, size: pageSize.value, ...applied.value }))
const { data, pending, error, refresh } = useLazyAsyncData(
  'evga-notices',
  () => evga.noticeList({ page: page.value, page_size: pageSize.value, ...applied.value }),
  { watch: [requestKey], server: false },
)

// Светофор по статусу уведомления: красный — доработка, жёлтый — в работе
// (согласование/подписание), зелёный — исполнено, серый — проект/отозвано.
function lightOf(statusId: number | null): { cls: string; title: string } {
  switch (statusId) {
    case 12: return { cls: 'lamp-red', title: 'Отправлено на доработку' }
    case 2:
    case 5: return { cls: 'lamp-amber', title: 'В работе: согласование/подписание' }
    case 3: return { cls: 'lamp-green', title: 'Исполнено' }
    default: return { cls: 'lamp-off', title: 'Проект или отозвано' }
  }
}

const rows = computed(() => data.value?.items ?? [])
const total = computed(() => data.value?.total ?? 0)
const scope = computed(() => data.value?.scope)
const first = computed(() => (page.value - 1) * pageSize.value)

// PrimeVue рисует строку фильтров по этому объекту; поиск серверный (lazy),
// значения держим в собственных ref-ах.
const colFilters = ref<Record<string, { value: null }>>({
  docnum: { value: null }, status_title: { value: null }, gu: { value: null },
  sendername: { value: null }, department: { value: null }, recipient: { value: null },
})

function onPage(e: { page: number; rows: number }) {
  page.value = e.page + 1
  pageSize.value = e.rows
}

const fmtAmount = (v: string) => {
  const n = Number(v)
  return Number.isFinite(n) ? n.toLocaleString('ru-RU', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : v
}
const fmtTS = (v: string | null) => (v ? new Date(v).toLocaleString('ru-RU') : '—')
</script>

<template>
  <div>
    <PageHeader title="ОБМ ЭВГА — Уведомления" description="Уведомления об устранении нарушений по витрине рисков 5-15а.">
      <template #actions>
        <Tag v-if="scope?.all_departments" value="Все департаменты" severity="info" />
        <NuxtLink to="/evga"><Button label="Реестр рисков" icon="pi pi-table" text /></NuxtLink>
      </template>
    </PageHeader>

    <Card>
      <template #content>
        <ErrorState v-if="error" message="Не удалось загрузить уведомления." retryable @retry="refresh" />
        <template v-else>
          <div class="toolbar">
            <span class="count">Всего: <b>{{ total.toLocaleString('ru-RU') }}</b></span>
            <span class="hint">Поиск — в строке под заголовками столбцов</span>
          </div>

          <DataTable
            show-gridlines
            :value="rows"
            :loading="pending"
            lazy
            striped-rows
            size="small"
            scrollable
            scroll-height="62vh"
            filter-display="row"
            :filters="colFilters"
            class="data-table compact"
          >
            <template #empty>
              <div class="empty-row">Уведомлений не найдено.</div>
            </template>
            <Column header="№ п/п" :style="{ width: '3.6rem' }">
              <template #body="{ index }">{{ first + index + 1 }}</template>
            </Column>
            <!-- светофор по статусу уведомления -->
            <Column header="" :style="{ width: '2.2rem' }">
              <template #body="{ data: n }">
                <span class="lamp" :class="lightOf(n.status_id).cls" :title="lightOf(n.status_id).title" />
              </template>
            </Column>
            <Column field="docnum" header="Номер">
              <template #body="{ data: n }">{{ n.docnum || '—' }}</template>
              <template #filter>
                <InputText v-model="fDocnum" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <Column field="status_title" header="Статус">
              <template #filter>
                <InputText v-model="cStatus" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <Column field="gu" header="Код ГУ">
              <template #filter>
                <InputText v-model="cGu" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <!-- ГУ-плательщик по платежу: это НЕ отправитель уведомления (спека 022 FR-3) -->
            <Column field="sendername" header="Наименование ГУ" :style="{ minWidth: '13rem' }">
              <template #filter>
                <InputText v-model="cSender" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <!-- Отправитель уведомления — департамент ДВГА (спека 022 FR-3) -->
            <Column field="department" header="Отправитель" :style="{ minWidth: '13rem' }">
              <template #body="{ data: n }">{{ n.department || '—' }}</template>
              <template #filter>
                <InputText v-model="cDept" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <Column field="recipient" header="Адресат" :style="{ minWidth: '14rem' }">
              <template #filter>
                <InputText v-model="cRecipient" placeholder="поиск" size="small" class="col-f" @input="applyColumnSearch" />
              </template>
            </Column>
            <Column field="rows_count" header="Записей" :style="{ width: '5.5rem' }" />
            <Column header="Сумма" class="num">
              <template #body="{ data: n }">{{ fmtAmount(n.total_sum) }}</template>
            </Column>
            <Column header="Создано">
              <template #body="{ data: n }">{{ fmtTS(n.created_at) }}</template>
            </Column>
            <Column :style="{ width: '3rem' }">
              <template #body="{ data: n }">
                <NuxtLink :to="`/evga/notices/${n.id}`" aria-label="Карточка уведомления">
                  <Button icon="pi pi-eye" text rounded size="small" />
                </NuxtLink>
              </template>
            </Column>
          </DataTable>

          <Paginator :rows="pageSize" :total-records="total" :first="first" :rows-per-page-options="[20, 50, 100]" @page="onPage" />
        </template>
      </template>
    </Card>
  </div>
</template>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 1rem; margin-bottom: 0.75rem; }
.count { font-size: 0.85rem; color: var(--ehd-ink-2); }
.count b { color: var(--ehd-ink); }
.hint { font-size: 0.78rem; color: var(--p-text-muted-color); }
.data-table { border: 1px solid var(--ehd-border); border-radius: var(--ehd-radius-sm); overflow: hidden; }
.empty-row { padding: 1.5rem; text-align: center; color: var(--p-text-muted-color); }
.num { text-align: right; }

/* Компактная таблица + закреплённая шапка со строкой поиска (спека 019 FR-3/FR-4). */
.compact :deep(.p-datatable-tbody > tr > td),
.compact :deep(.p-datatable-thead > tr > th) { padding: 0.3rem 0.45rem; font-size: 0.8rem; }
.compact :deep(.p-datatable-thead > tr > th) { position: sticky; z-index: 2; background: var(--p-datatable-header-cell-background); }
.col-f { width: 100%; min-width: 5rem; }
.col-f :deep(input) { width: 100%; font-size: 0.78rem; padding: 0.2rem 0.35rem; }

/* Светофор по статусу уведомления (спека 019 FR-5) */
.lamp { display: inline-block; width: 0.7rem; height: 0.7rem; border-radius: 50%; border: 1px solid rgb(0 0 0 / 12%); }
.lamp-red { background: var(--p-red-500); }
.lamp-amber { background: var(--p-amber-400); }
.lamp-green { background: var(--p-green-500); }
.lamp-off { background: var(--p-surface-300); }
</style>
