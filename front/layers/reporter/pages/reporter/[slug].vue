<script setup lang="ts">
import type { QuerySpec, FilterSpec } from '~~/shared/api/types'
import { formatCell } from '../../utils/format'
import { defaultOperator } from '../../utils/filters'

// Пользовательская таблица витрины (REP-FR-050..055).
definePageMeta({ middleware: 'auth' })

const route = useRoute()
const router = useRouter()
const views = useReporterViews()

const slug = computed(() => String(route.params.slug))

// --- состояние таблицы (отражается в URL) ---
const searchInput = ref(typeof route.query.q === 'string' ? route.query.q : '')
const search = ref(searchInput.value)
const pageSize = ref(route.query.page_size ? Number(route.query.page_size) : 0)
const activeFilters = ref<FilterSpec[]>([])
const sort = ref<{ column: string; dir: 'asc' | 'desc' } | null>(null)

// --- метаданные ---
const {
  data: meta,
  pending: metaPending,
  error: metaError,
} = await useAsyncData(`view-meta-${slug.value}`, () => views.meta(slug.value))

const effectivePageSize = computed(() => pageSize.value || meta.value?.page_size_default || 50)

function currentSpec(cursor?: string): QuerySpec {
  return {
    search: search.value || undefined,
    page_size: effectivePageSize.value,
    cursor,
    filters: activeFilters.value.length ? activeFilters.value : undefined,
    sort: sort.value ? { column: sort.value.column, dir: sort.value.dir } : undefined,
  }
}

const filtersKey = computed(() => JSON.stringify(activeFilters.value))
const sortKey = computed(() => JSON.stringify(sort.value))

// Данные и total_count грузятся на КЛИЕНТЕ (server:false) и параллельно.
const {
  data: queryData,
  pending: dataPending,
  error: dataError,
  refresh: refreshData,
} = useLazyAsyncData(`view-data-${slug.value}`, () => views.query(slug.value, currentSpec()), {
  watch: [search, effectivePageSize, filtersKey, sortKey],
  server: false,
})

const {
  data: countData,
  pending: countPending,
  refresh: refreshCount,
} = useLazyAsyncData(`view-count-${slug.value}`, () => views.count(slug.value, currentSpec()), {
  watch: [search, effectivePageSize, filtersKey],
  server: false,
})

function reload() {
  refreshData()
  refreshCount()
}

// --- догруженные keyset-страницы ---
const extraRows = ref<Record<string, unknown>[]>([])
const nextCursor = ref('')
watch(
  queryData,
  (d) => {
    extraRows.value = []
    nextCursor.value = d?.page.next_cursor ?? ''
  },
  { immediate: true },
)

const columns = computed(() => queryData.value?.columns ?? meta.value?.columns ?? [])
const rows = computed(() => [...(queryData.value?.rows ?? []), ...extraRows.value])
const total = computed(() => countData.value?.total_count ?? 0)
const hasMore = computed(() => nextCursor.value !== '')

const loadingMore = ref(false)
async function loadMore() {
  if (!hasMore.value || loadingMore.value) return
  loadingMore.value = true
  try {
    const res = await views.query(slug.value, currentSpec(nextCursor.value))
    extraRows.value.push(...res.rows)
    nextCursor.value = res.page.next_cursor
  } catch (e) {
    actionError.value = apiErrorMessage(e)
  } finally {
    loadingMore.value = false
  }
}

function applySearch() {
  search.value = searchInput.value.trim()
}

// --- метаданные колонок: флаги sortable/filterable/type ---
const colMeta = computed(() => {
  const m: Record<string, { sortable: boolean; filterable: boolean; display_type: string }> = {}
  for (const c of meta.value?.columns ?? []) {
    m[c.source_name] = { sortable: c.sortable, filterable: c.filterable, display_type: c.display_type }
  }
  return m
})
const sortableFor = (name: string) => colMeta.value[name]?.sortable ?? false
const filterableFor = (name: string) => colMeta.value[name]?.filterable ?? false

// --- клик-сортировка по колонке ---
function onSort(e: { sortField?: string | null; sortOrder?: number | null }) {
  if (e.sortField && e.sortOrder) {
    sort.value = { column: String(e.sortField), dir: e.sortOrder === 1 ? 'asc' : 'desc' }
  } else {
    sort.value = null
  }
}

// --- inline-фильтры под заголовками (PrimeVue filterDisplay="row") ---
type FilterCell = { value: string | null; matchMode: string }
const pvFilters = ref<Record<string, FilterCell>>({})
watch(
  meta,
  (m) => {
    const next: Record<string, FilterCell> = {}
    for (const c of m?.columns ?? []) {
      if (c.filterable) next[c.source_name] = pvFilters.value[c.source_name] ?? { value: null, matchMode: 'contains' }
    }
    pvFilters.value = next
  },
  { immediate: true },
)

// применяет значения фильтр-ряда → activeFilters (перезагрузка через watch)
function applyColumnFilters() {
  const out: FilterSpec[] = []
  for (const c of meta.value?.columns ?? []) {
    if (!c.filterable) continue
    const raw = pvFilters.value[c.source_name]?.value
    const v = (raw ?? '').toString().trim()
    if (v === '') continue
    out.push({ column: c.source_name, operator: defaultOperator(c.display_type), value: v })
  }
  activeFilters.value = out
}

// --- экспорт ---
const exporting = ref(false)
const actionError = ref('')
async function doExport() {
  exporting.value = true
  actionError.value = ''
  try {
    await views.exportView(slug.value, {
      search: search.value || undefined,
      filters: activeFilters.value.length ? activeFilters.value : undefined,
    })
  } catch (e) {
    actionError.value = apiErrorMessage(e)
  } finally {
    exporting.value = false
  }
}

// --- синхронизация состояния в URL (только на клиенте) ---
if (import.meta.client) {
  watch([search, pageSize], () => {
    const q: Record<string, string> = {}
    if (search.value) q.q = search.value
    if (pageSize.value) q.page_size = String(pageSize.value)
    router.replace({ query: q })
  })
}

// --- классификация состояния экрана (ТЗ, Принцип 6) ---
const errCode = computed(() => apiErrorCode(metaError.value) || apiErrorCode(dataError.value))
const screenState = computed(() => {
  if (metaPending.value) return 'loading'
  switch (errCode.value) {
    case 'ACCESS_DENIED':
      return 'denied'
    case 'VIEW_NOT_FOUND':
      return 'notfound'
    case 'SOURCE_UNAVAILABLE':
      return 'source'
  }
  if (metaError.value || dataError.value) return 'error'
  if (!queryData.value) return 'loading'
  return 'ready'
})

const isEmpty = computed(() => !!queryData.value && rows.value.length === 0)
const tableLoading = computed(() => dataPending.value || loadingMore.value)
const countReady = computed(() => !countPending.value && countData.value != null)

const pageSizeOptions = computed(() => {
  const m = meta.value
  if (!m) return [50]
  return [m.page_size_min, m.page_size_default, m.page_size_max].filter((v, i, a) => a.indexOf(v) === i)
})
</script>

<template>
  <div>
    <PageHeader :title="meta?.name || 'Витрина'" :description="meta?.description || undefined" />
    <Card>
      <template #content>
        <div v-if="screenState === 'loading'" class="center">
          <ProgressSpinner style="width: 2.5rem; height: 2.5rem" />
          <p class="loading-text">Загрузка данных…</p>
        </div>
        <ErrorState v-else-if="screenState === 'denied'" title="Доступ запрещён" message="У вас нет прав на просмотр этой витрины." />
        <ErrorState v-else-if="screenState === 'notfound'" title="Витрина не найдена" message="Представление не существует или снято с публикации." />
        <ErrorState v-else-if="screenState === 'source'" title="Источник недоступен" message="Источник данных временно недоступен. Повторите позже." retryable @retry="reload" />
        <ErrorState v-else-if="screenState === 'error'" message="Не удалось загрузить данные витрины." retryable @retry="reload" />

        <template v-else>
          <div class="toolbar">
            <IconField iconPosition="left" class="search">
              <InputIcon class="pi pi-search" />
              <InputText v-model="searchInput" placeholder="Поиск..." @keyup.enter="applySearch" @blur="applySearch" />
            </IconField>
            <span class="count">
              Записей:
              <b v-if="countReady">{{ total.toLocaleString('ru-RU') }}</b>
              <span v-else class="counting"><i class="pi pi-spin pi-spinner" /></span>
            </span>
            <div class="toolbar-right">
              <Select v-model="pageSize" :options="pageSizeOptions" placeholder="Размер страницы" aria-label="Размер страницы" />
              <Button label="Экспорт" icon="pi pi-download" :loading="exporting" :disabled="rows.length === 0" @click="doExport" />
            </div>
          </div>

          <Message v-if="actionError" severity="error" :closable="true" class="action-error">{{ actionError }}</Message>

          <DataTable
            :value="rows"
            :loading="tableLoading"
            lazy
            v-model:filters="pvFilters"
            filter-display="row"
            :sort-field="sort?.column"
            :sort-order="sort ? (sort.dir === 'asc' ? 1 : -1) : 0"
            removable-sort
            striped-rows
            size="small"
            scrollable
            class="data-table"
            @sort="onSort"
          >
            <template #loading><ProgressSpinner style="width: 2rem; height: 2rem" /></template>
            <template #empty>
              <div class="empty-row">По текущему запросу строк не найдено.</div>
            </template>
            <Column
              v-for="col in columns"
              :key="col.source_name"
              :field="col.source_name"
              :header="col.label"
              :sortable="sortableFor(col.source_name)"
              :show-filter-menu="false"
            >
              <template v-if="filterableFor(col.source_name)" #filter="{ filterModel, filterCallback }">
                <InputText
                  v-model="filterModel.value"
                  placeholder="Все"
                  class="col-filter"
                  @keyup.enter="() => { filterCallback(); applyColumnFilters() }"
                  @blur="() => { filterCallback(); applyColumnFilters() }"
                />
              </template>
              <template #body="{ data }">{{ formatCell(data[col.source_name], col.display_type) }}</template>
            </Column>
          </DataTable>

          <div v-if="!isEmpty" class="pager">
            <span class="loaded">Загружено {{ rows.length }}</span>
            <Button v-if="hasMore" label="Показать ещё" icon="pi pi-chevron-down" outlined size="small" :loading="loadingMore" @click="loadMore" />
          </div>
        </template>
      </template>
    </Card>
  </div>
</template>

<style scoped>
.center { display: flex; flex-direction: column; align-items: center; gap: 0.75rem; padding: 2.5rem 0; }
.loading-text { margin: 0; font-size: 0.9rem; color: var(--p-text-muted-color); }
.toolbar { display: flex; align-items: center; gap: 1rem; margin-bottom: 1rem; }
.count { font-size: 0.9rem; color: var(--ehd-ink-2); }
.count b { color: var(--ehd-ink); }
.counting { display: inline-flex; align-items: center; }
.toolbar-right { margin-left: auto; display: flex; align-items: center; gap: 0.6rem; }
.search :deep(input) { min-width: 16rem; }
.action-error { margin-bottom: 1rem; }
.data-table { border: 1px solid var(--ehd-border); border-radius: var(--ehd-radius-sm); overflow: hidden; }
.col-filter { width: 100%; min-width: 7rem; }
.empty-row { padding: 1.5rem; text-align: center; color: var(--p-text-muted-color); }
.pager { display: flex; align-items: center; justify-content: space-between; margin-top: 1rem; }
.loaded { font-size: 0.85rem; color: var(--p-text-muted-color); }
</style>
