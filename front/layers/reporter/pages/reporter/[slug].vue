<script setup lang="ts">
import type { QuerySpec } from '~~/shared/api/types'
import { formatCell } from '../../utils/format'

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

// --- метаданные ---
const {
  data: meta,
  pending: metaPending,
  error: metaError,
} = await useAsyncData(`view-meta-${slug.value}`, () => views.meta(slug.value))

const effectivePageSize = computed(() => pageSize.value || meta.value?.page_size_default || 50)

function currentSpec(cursor?: string): QuerySpec {
  return { search: search.value || undefined, page_size: effectivePageSize.value, cursor }
}

// Данные и total_count грузятся на КЛИЕНТЕ (server:false) и параллельно: на больших таблицах
// запрос/COUNT долгие — не блокируем SSR, показываем лоадер, каркас (meta) отрендерен на сервере.
const {
  data: queryData,
  pending: dataPending,
  error: dataError,
  refresh: refreshData,
} = useLazyAsyncData(`view-data-${slug.value}`, () => views.query(slug.value, currentSpec()), {
  watch: [search, effectivePageSize],
  server: false,
})

const {
  data: countData,
  pending: countPending,
  refresh: refreshCount,
} = useLazyAsyncData(`view-count-${slug.value}`, () => views.count(slug.value, currentSpec()), {
  watch: [search, effectivePageSize],
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

// --- экспорт ---
const exporting = ref(false)
const actionError = ref('')
async function doExport() {
  exporting.value = true
  actionError.value = ''
  try {
    await views.exportView(slug.value, { search: search.value || undefined })
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
  if (!queryData.value) return 'loading' // первая порция ещё грузится (клиентский fetch)
  if (rows.value.length === 0) return 'empty'
  return 'ready'
})

// таблица показывает оверлей-лоадер при любой подгрузке (первичной, поиске, смене размера)
const tableLoading = computed(() => dataPending.value || loadingMore.value)
// count готов только когда реально загружен (иначе показываем «…», не «0»)
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
        <ErrorState
          v-else-if="screenState === 'denied'"
          title="Доступ запрещён"
          message="У вас нет прав на просмотр этой витрины."
        />
        <ErrorState
          v-else-if="screenState === 'notfound'"
          title="Витрина не найдена"
          message="Представление не существует или снято с публикации."
        />
        <ErrorState
          v-else-if="screenState === 'source'"
          title="Источник недоступен"
          message="Источник данных временно недоступен. Повторите позже."
          retryable
          @retry="reload"
        />
        <ErrorState
          v-else-if="screenState === 'error'"
          message="Не удалось загрузить данные витрины."
          retryable
          @retry="reload"
        />

        <template v-else>
          <div class="toolbar">
            <IconField iconPosition="left" class="search">
              <InputIcon class="pi pi-search" />
              <InputText
                v-model="searchInput"
                placeholder="Поиск..."
                @keyup.enter="applySearch"
                @blur="applySearch"
              />
            </IconField>
            <div class="toolbar-right">
              <Select
                v-model="pageSize"
                :options="pageSizeOptions"
                placeholder="Размер страницы"
                aria-label="Размер страницы"
              />
              <Button
                label="Экспорт"
                icon="pi pi-download"
                :loading="exporting"
                :disabled="rows.length === 0"
                @click="doExport"
              />
            </div>
          </div>

          <Message v-if="actionError" severity="error" :closable="true" class="action-error">
            {{ actionError }}
          </Message>

          <EmptyState
            v-if="screenState === 'empty'"
            icon="pi pi-inbox"
            title="Нет данных"
            hint="По текущему запросу строк не найдено."
          />
          <template v-else>
            <DataTable :value="rows" :loading="tableLoading" stripedRows size="small" scrollable class="data-table">
              <template #loading>
                <ProgressSpinner style="width: 2rem; height: 2rem" />
              </template>
              <Column v-for="col in columns" :key="col.source_name" :header="col.label">
                <template #body="{ data }">
                  {{ formatCell(data[col.source_name], col.display_type) }}
                </template>
              </Column>
            </DataTable>

            <div class="pager">
              <span class="loaded">
                Загружено {{ rows.length }} из
                <span v-if="countReady">{{ total.toLocaleString('ru-RU') }}</span>
                <span v-else class="counting"><i class="pi pi-spin pi-spinner" /> …</span>
              </span>
              <Button
                v-if="hasMore"
                label="Показать ещё"
                icon="pi pi-chevron-down"
                outlined
                size="small"
                :loading="loadingMore"
                @click="loadMore"
              />
            </div>
          </template>
        </template>
      </template>
    </Card>
  </div>
</template>

<style scoped>
.center {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  padding: 2.5rem 0;
}
.loading-text {
  margin: 0;
  font-size: 0.9rem;
  color: var(--p-text-muted-color);
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1rem;
}
.toolbar-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 0.6rem;
}
.search :deep(input) {
  min-width: 16rem;
}
.action-error {
  margin-bottom: 1rem;
}
.data-table {
  border: 1px solid var(--ehd-border);
  border-radius: var(--ehd-radius-sm);
  overflow: hidden;
}
.pager {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 1rem;
}
.loaded {
  font-size: 0.85rem;
  color: var(--p-text-muted-color);
}
.counting {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}
</style>
