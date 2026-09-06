<script setup lang="ts">
import type { EvgaReference, EvgaStatusChange } from '~~/shared/api/types'
import { allowedTargets } from '../../utils/statusFlow'

// Карточка записи витрины 5-15а: поля + смена статуса + история (EVGA-FR-015/018; спеки 005/006).
definePageMeta({ middleware: 'auth' })

const route = useRoute()
const evga = useEvga()
const session = useSessionStore()
const id = computed(() => Number(route.params.id))

const canWrite = computed(() => session.isAdmin || (session.user?.roles ?? []).includes('evga_auditor'))

const { data: rec, pending, error, refresh } = useLazyAsyncData(`evga-card-${id.value}`, () => evga.card(id.value), { server: false })

// история статусов (журнал спеки 008) и справочники для диалога
const { data: historyData, refresh: refreshHistory } = useLazyAsyncData(
  `evga-history-${id.value}`,
  () => evga.history(id.value),
  { server: false },
)
const history = computed(() => historyData.value?.items ?? [])
const { data: refs } = useLazyAsyncData('evga-references', () => evga.references(), { server: false })

// --- смена статуса (одиночная) ---
const statusDialog = ref(false)
const statusBusy = ref(false)
const statusError = ref('')

const targets = computed<EvgaReference[]>(() => {
  const allowed = new Set(allowedTargets(rec.value?.status_id ?? null))
  return (refs.value?.statuses ?? []).filter((s) => allowed.has(s.id))
})

async function submitStatus(change: EvgaStatusChange) {
  statusBusy.value = true
  statusError.value = ''
  try {
    await evga.changeStatus(id.value, change)
    statusDialog.value = false
    refresh()
    refreshHistory()
  } catch (e) {
    statusError.value = apiErrorMessage(e)
  } finally {
    statusBusy.value = false
  }
}

const fmtTS = (v: string | null) => (v ? new Date(v).toLocaleString('ru-RU') : '—')

const errCode = computed(() => apiErrorCode(error.value))
const screenState = computed(() => {
  switch (errCode.value) {
    case 'ACCESS_DENIED':
      return 'denied'
    case 'NOT_FOUND':
      return 'notfound'
    case 'EVGA_SOURCE_UNAVAILABLE':
      return 'source'
  }
  if (error.value) return 'error'
  if (!rec.value) return 'loading'
  return 'ready'
})

const fmtAmount = (v: string) => {
  if (!v) return '—'
  const n = Number(v)
  return Number.isFinite(n) ? n.toLocaleString('ru-RU', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : v
}
const fmtDate = (v: string | null) => {
  if (!v) return '—'
  const [y, m, d] = v.split('-')
  return `${d}.${m}.${y}`
}
const fio = computed(() => (rec.value ? [rec.value.fm, rec.value.nm, rec.value.ft].filter(Boolean).join(' ') || '—' : '—'))
const flFio = computed(() => (rec.value ? [rec.value.fl_fm, rec.value.fl_nm, rec.value.fl_ft].filter(Boolean).join(' ') || '—' : '—'))

type Row = { label: string; value: string }
const paymentRows = computed<Row[]>(() => {
  const r = rec.value
  if (!r) return []
  return [
    { label: 'Номер платежа', value: r.ppo_pp || '—' },
    { label: 'Дата платежа', value: fmtDate(r.paymentdate) },
    { label: 'Сумма (часть)', value: fmtAmount(r.amount_part) },
    { label: 'Период (год/месяц)', value: r.god ? `${r.god} / ${r.mes ?? '—'}` : '—' },
    { label: 'Код ГУ отправителя', value: r.gu || '—' },
    { label: 'БИН ГУ', value: r.gu_bin || '—' },
    { label: 'Наименование отправителя', value: r.sendername || '—' },
  ]
})
const receiverRows = computed<Row[]>(() => {
  const r = rec.value
  if (!r) return []
  return [
    { label: 'ИИН получателя', value: r.iin || '—' },
    { label: 'ФИО получателя', value: fio.value },
    { label: 'Карт-счёт', value: r.la1 || '—' },
    { label: 'ФИО по ГБД ФЛ', value: flFio.value },
    { label: 'Найден в ГБД ФЛ', value: r.is_gbdfl == null ? '—' : r.is_gbdfl ? 'Да' : 'Нет' },
  ]
})
const workRows = computed<Row[]>(() => {
  const r = rec.value
  if (!r) return []
  return [
    { label: 'Профиль риска', value: r.profile_title || '—' },
    { label: 'Департамент', value: r.department_title || '—' },
    { label: 'Статус', value: r.status_title || '—' },
    { label: 'Комментарий', value: r.status_note || '—' },
    { label: 'Сумма к возмещению', value: fmtAmount(r.amount_for_vozvrat) },
    { label: 'Возмещено', value: fmtAmount(r.refund) },
    { label: 'Уведомление', value: r.notice_num || '—' },
    { label: 'Исходящее письмо', value: r.out_num || '—' },
  ]
})
</script>

<template>
  <div>
    <PageHeader :title="`Запись витрины № ${id}`" description="Карточка записи реестра рисков 5-15а.">
      <template #actions>
        <Button
          v-if="canWrite && targets.length"
          label="Сменить статус"
          icon="pi pi-pencil"
          severity="warn"
          @click="statusError = ''; statusDialog = true"
        />
        <NuxtLink to="/evga"><Button label="К реестру" icon="pi pi-arrow-left" text /></NuxtLink>
      </template>
    </PageHeader>

    <ErrorState v-if="screenState === 'denied'" title="Доступ запрещён" message="У вас нет прав на просмотр этой записи." />
    <ErrorState v-else-if="screenState === 'notfound'" title="Запись не найдена" message="Запись не существует или относится к другому департаменту." />
    <ErrorState v-else-if="screenState === 'source'" title="Источник недоступен" message="База данных ОБМ ЕВГА временно недоступна." retryable @retry="refresh" />
    <ErrorState v-else-if="screenState === 'error'" message="Не удалось загрузить карточку." retryable @retry="refresh" />
    <div v-else-if="screenState === 'loading'" class="center"><ProgressSpinner style="width: 2.5rem; height: 2.5rem" /></div>

    <div v-else class="grid">
      <Card>
        <template #title>Платёж</template>
        <template #content>
          <dl class="props">
            <template v-for="row in paymentRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </template>
          </dl>
        </template>
      </Card>
      <Card>
        <template #title>Получатель</template>
        <template #content>
          <dl class="props">
            <template v-for="row in receiverRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </template>
          </dl>
        </template>
      </Card>
      <Card class="span2">
        <template #title>Отработка</template>
        <template #content>
          <dl class="props">
            <template v-for="row in workRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </template>
          </dl>
        </template>
      </Card>

      <Card class="span2">
        <template #title>История статусов</template>
        <template #content>
          <p v-if="history.length === 0" class="hist-empty">Изменений статуса пока не зафиксировано.</p>
          <DataTable v-else :value="history" size="small" striped-rows class="hist-table">
            <Column header="Дата">
              <template #body="{ data: h }">{{ fmtTS(h.changed_at) }}</template>
            </Column>
            <Column header="Переход">
              <template #body="{ data: h }">{{ h.status_from_title || '∅' }} → {{ h.status_to_title }}</template>
            </Column>
            <Column field="note" header="Комментарий">
              <template #body="{ data: h }">{{ h.note || '—' }}</template>
            </Column>
            <Column header="Суммы">
              <template #body="{ data: h }">
                {{ h.amount_for_vozvrat ? `к возм.: ${h.amount_for_vozvrat}` : '' }}
                {{ h.refund ? `возм.: ${h.refund}` : '' }}
                {{ !h.amount_for_vozvrat && !h.refund ? '—' : '' }}
              </template>
            </Column>
            <Column header="Кто">
              <template #body="{ data: h }">{{ h.changed_by_name || '—' }}</template>
            </Column>
            <Column field="change_source" header="Источник" :style="{ width: '7rem' }" />
          </DataTable>
        </template>
      </Card>
    </div>

    <EvgaStatusDialog
      v-model:visible="statusDialog"
      :title="`Сменить статус записи № ${id}`"
      :targets="targets"
      :activities="refs?.activities ?? []"
      :busy="statusBusy"
      :error="statusError"
      @submit="submitStatus"
    />
  </div>
</template>

<style scoped>
.center { display: flex; justify-content: center; padding: 2.5rem 0; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
.span2 { grid-column: 1 / -1; }
.props { display: grid; grid-template-columns: minmax(14rem, auto) 1fr; gap: 0.45rem 1rem; margin: 0; }
.props dt { color: var(--ehd-ink-2); font-size: 0.88rem; }
.props dd { margin: 0; color: var(--ehd-ink); font-size: 0.92rem; overflow-wrap: anywhere; }
.hist-empty { margin: 0; color: var(--p-text-muted-color); font-size: 0.9rem; }
.hist-table { border: 1px solid var(--ehd-border); border-radius: var(--ehd-radius-sm); overflow: hidden; }
@media (max-width: 1100px) { .grid { grid-template-columns: 1fr; } }
</style>
