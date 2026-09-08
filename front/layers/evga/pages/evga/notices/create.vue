<script setup lang="ts">
import type { EvgaCliOrg, EvgaNoticeCreateResult } from '~~/shared/api/types'
import { canCreate, type GroupState } from '../../../utils/noticeForm'

// Экран группировки формирования уведомлений (EVGA-FR-031…036; спека 007-evga-notices-ui).
definePageMeta({ middleware: 'auth' })

const evga = useEvga()
const draft = useEvgaNoticeDraft()
const ids = [...draft.value]

const { data: preview, pending, error, refresh } = useLazyAsyncData(
  'evga-notice-preview',
  () => evga.noticePreview(ids),
  { server: false, immediate: ids.length > 0 },
)

// состояние групп: исключение + выбранный адресат (ключ — gu)
const groupState = ref<Record<string, { excluded: boolean; cli: EvgaCliOrg | null }>>({})
watch(
  preview,
  (p) => {
    const next: typeof groupState.value = {}
    for (const g of p?.groups ?? []) next[g.gu] = groupState.value[g.gu] ?? { excluded: false, cli: null }
    groupState.value = next
  },
  { immediate: true },
)

const groups = computed(() => preview.value?.groups ?? [])
const rejected = computed(() => preview.value?.rejected ?? [])

const createEnabled = computed(() => {
  const states: GroupState[] = groups.value.map((g) => ({
    gu: g.gu,
    excluded: groupState.value[g.gu]?.excluded ?? false,
    cliId: groupState.value[g.gu]?.cli?.id ?? null,
  }))
  return canCreate(states)
})

// автодополнение адресата (EVGA-FR-033)
const cliSuggestions = ref<EvgaCliOrg[]>([])
async function searchCli(e: { query: string }) {
  try {
    cliSuggestions.value = (await evga.cliSearch(e.query)).items
  } catch {
    cliSuggestions.value = []
  }
}

const creating = ref(false)
const actionError = ref('')
const result = ref<EvgaNoticeCreateResult | null>(null)

async function submit() {
  const payload = groups.value
    .filter((g) => !groupState.value[g.gu]?.excluded)
    .map((g) => ({
      record_ids: g.records.map((r) => r.id),
      its_cli_id: groupState.value[g.gu]!.cli!.id,
    }))
  creating.value = true
  actionError.value = ''
  try {
    result.value = await evga.noticeCreate(payload)
    draft.value = []
  } catch (e) {
    actionError.value = apiErrorMessage(e)
  } finally {
    creating.value = false
  }
}

const fmtAmount = (v: string) => {
  const n = Number(v)
  return Number.isFinite(n) ? n.toLocaleString('ru-RU', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : v
}
</script>

<template>
  <div>
    <PageHeader title="Формирование уведомлений" description="Записи сгруппированы по коду ГУ. Каждая группа — отдельное уведомление; выберите адресата из справочника ЕСЭДО.">
      <template #actions>
        <NuxtLink to="/evga"><Button label="К реестру" icon="pi pi-arrow-left" text /></NuxtLink>
      </template>
    </PageHeader>

    <Card v-if="ids.length === 0 && !result">
      <template #content>
        <EmptyState title="Записи не выбраны" message="Вернитесь в реестр и отметьте записи для формирования уведомления." />
      </template>
    </Card>

    <template v-else-if="result">
      <Card>
        <template #title>Результат</template>
        <template #content>
          <p class="report-line">
            Создано уведомлений: <b>{{ result.created.length }}</b>
            <span v-if="result.rejected.length"> · Отклонено записей: <b>{{ result.rejected.length }}</b></span>
          </p>
          <ul class="created-list">
            <li v-for="c in result.created" :key="c.notice_id">
              <NuxtLink :to="`/evga/notices/${c.notice_id}`">Уведомление #{{ c.notice_id }}</NuxtLink>
              — ГУ {{ c.gu }}, записей: {{ c.records }}
            </li>
          </ul>
          <DataTable v-if="result.rejected.length" :value="result.rejected" size="small" striped-rows class="mt">
            <Column field="id" header="Запись" :style="{ width: '8rem' }" />
            <Column field="reason" header="Причина" />
          </DataTable>
          <div class="mt">
            <NuxtLink to="/evga/notices"><Button label="К списку уведомлений" icon="pi pi-list" /></NuxtLink>
          </div>
        </template>
      </Card>
    </template>

    <template v-else>
      <Card>
        <template #content>
          <div v-if="pending" class="center"><ProgressSpinner style="width: 2.5rem; height: 2.5rem" /></div>
          <ErrorState v-else-if="error" message="Не удалось загрузить группировку." retryable @retry="refresh" />
          <template v-else>
            <DataTable :value="groups" size="small" striped-rows class="data-table">
              <template #empty>
                <div class="empty-row">Ни одна из выбранных записей не может войти в уведомление.</div>
              </template>
              <Column field="gu" header="Код ГУ" :style="{ width: '8rem' }" />
              <Column field="sendername" header="Наименование отправителя" :style="{ minWidth: '18rem' }" />
              <Column field="count" header="Кол-во записей" :style="{ width: '8rem' }" />
              <Column header="Общая сумма" class="num" :style="{ width: '10rem' }">
                <template #body="{ data: g }">{{ fmtAmount(g.total_sum) }}</template>
              </Column>
              <Column header="Адресат (ЕСЭДО)" :style="{ minWidth: '22rem' }">
                <template #body="{ data: g }">
                  <AutoComplete
                    v-model="groupState[g.gu]!.cli"
                    :suggestions="cliSuggestions"
                    option-label="title"
                    placeholder="Поиск по наименованию или коду…"
                    :disabled="groupState[g.gu]?.excluded"
                    :min-length="2"
                    fluid
                    @complete="searchCli"
                  />
                </template>
              </Column>
              <Column :style="{ width: '9rem' }">
                <template #body="{ data: g }">
                  <Button
                    :label="groupState[g.gu]?.excluded ? 'Вернуть' : 'Исключить'"
                    :severity="groupState[g.gu]?.excluded ? 'secondary' : 'danger'"
                    text
                    size="small"
                    @click="groupState[g.gu]!.excluded = !groupState[g.gu]!.excluded"
                  />
                </template>
              </Column>
            </DataTable>

            <Message v-if="rejected.length" severity="warn" :closable="false" class="mt">
              Не попали в формирование ({{ rejected.length }}):
            </Message>
            <DataTable v-if="rejected.length" :value="rejected" size="small" striped-rows class="mt-sm">
              <Column field="id" header="Запись" :style="{ width: '8rem' }" />
              <Column field="reason" header="Причина" />
            </DataTable>

            <Message v-if="actionError" severity="error" :closable="true" class="mt">{{ actionError }}</Message>

            <div class="actions">
              <Button label="Создать уведомления" icon="pi pi-check" :disabled="!createEnabled" :loading="creating" @click="submit" />
              <span v-if="!createEnabled && groups.length" class="hint">
                Кнопка активируется, когда у каждой включённой группы выбран адресат.
              </span>
            </div>
          </template>
        </template>
      </Card>
    </template>
  </div>
</template>

<style scoped>
.center { display: flex; justify-content: center; padding: 2.5rem 0; }
.data-table { border: 1px solid var(--ehd-border); border-radius: var(--ehd-radius-sm); overflow: hidden; }
.empty-row { padding: 1.5rem; text-align: center; color: var(--p-text-muted-color); }
.num { text-align: right; }
.mt { margin-top: 1rem; }
.mt-sm { margin-top: 0.5rem; }
.actions { display: flex; align-items: center; gap: 1rem; margin-top: 1.25rem; }
.hint { font-size: 0.85rem; color: var(--p-text-muted-color); }
.report-line { margin: 0 0 0.75rem; font-size: 0.95rem; }
.created-list { margin: 0; padding-left: 1.2rem; }
</style>
