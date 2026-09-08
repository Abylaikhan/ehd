<script setup lang="ts">
// Список уведомлений (спека 007-evga-notices-ui; backend 009 FR-7).
definePageMeta({ middleware: 'auth' })

const evga = useEvga()

const page = ref(1)
const pageSize = ref(20)
const fDocnum = ref('')
const appliedDocnum = ref('')

const requestKey = computed(() => JSON.stringify({ page: page.value, size: pageSize.value, docnum: appliedDocnum.value }))
const { data, pending, error, refresh } = useLazyAsyncData(
  'evga-notices',
  () => evga.noticeList({ page: page.value, page_size: pageSize.value, docnum: appliedDocnum.value || undefined }),
  { watch: [requestKey], server: false },
)

const rows = computed(() => data.value?.items ?? [])
const total = computed(() => data.value?.total ?? 0)
const scope = computed(() => data.value?.scope)
const first = computed(() => (page.value - 1) * pageSize.value)

function applyFilter() {
  page.value = 1
  appliedDocnum.value = fDocnum.value.trim()
}
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
    <PageHeader title="ОБМ ЕВГА — Уведомления" description="Уведомления об устранении нарушений по витрине рисков 5-15а.">
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
            <IconField icon-position="left">
              <InputIcon class="pi pi-search" />
              <InputText v-model="fDocnum" placeholder="№ уведомления…" @keyup.enter="applyFilter" @blur="applyFilter" />
            </IconField>
            <span class="count">Всего: <b>{{ total.toLocaleString('ru-RU') }}</b></span>
          </div>

          <DataTable :value="rows" :loading="pending" lazy striped-rows size="small" scrollable class="data-table">
            <template #empty>
              <div class="empty-row">Уведомлений не найдено.</div>
            </template>
            <Column header="№ п/п" :style="{ width: '4rem' }">
              <template #body="{ index }">{{ first + index + 1 }}</template>
            </Column>
            <Column header="Номер">
              <template #body="{ data: n }">{{ n.docnum || '—' }}</template>
            </Column>
            <Column field="status_title" header="Статус" />
            <Column field="gu" header="Код ГУ" />
            <Column field="sendername" header="Отправитель" :style="{ minWidth: '14rem' }" />
            <Column field="recipient" header="Адресат" :style="{ minWidth: '16rem' }" />
            <Column field="rows_count" header="Записей" :style="{ width: '6rem' }" />
            <Column header="Сумма" class="num">
              <template #body="{ data: n }">{{ fmtAmount(n.total_sum) }}</template>
            </Column>
            <Column header="Создано">
              <template #body="{ data: n }">{{ fmtTS(n.created_at) }}</template>
            </Column>
            <Column v-if="scope?.all_departments" field="department" header="Департамент" :style="{ minWidth: '12rem' }" />
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
.toolbar { display: flex; align-items: center; gap: 1rem; margin-bottom: 1rem; }
.count { font-size: 0.9rem; color: var(--ehd-ink-2); }
.count b { color: var(--ehd-ink); }
.data-table { border: 1px solid var(--ehd-border); border-radius: var(--ehd-radius-sm); overflow: hidden; }
.empty-row { padding: 1.5rem; text-align: center; color: var(--p-text-muted-color); }
.num { text-align: right; }
</style>
