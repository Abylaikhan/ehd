<script setup lang="ts">
// Карточка записи витрины 5-15а (EVGA-FR-018; история статусов появится в фазе 3).
definePageMeta({ middleware: 'auth' })

const route = useRoute()
const evga = useEvga()
const id = computed(() => Number(route.params.id))

const { data: rec, pending, error, refresh } = useLazyAsyncData(`evga-card-${id.value}`, () => evga.card(id.value), { server: false })

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
    <PageHeader :title="`Запись витрины № ${id}`" description="Карточка записи реестра рисков 5-15а. Режим чтения.">
      <template #actions>
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
          <Message severity="secondary" :closable="false" class="hist-note">
            История изменения статусов появится после внедрения журнала (фаза 3).
          </Message>
        </template>
      </Card>
    </div>
  </div>
</template>

<style scoped>
.center { display: flex; justify-content: center; padding: 2.5rem 0; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
.span2 { grid-column: 1 / -1; }
.props { display: grid; grid-template-columns: minmax(14rem, auto) 1fr; gap: 0.45rem 1rem; margin: 0; }
.props dt { color: var(--ehd-ink-2); font-size: 0.88rem; }
.props dd { margin: 0; color: var(--ehd-ink); font-size: 0.92rem; overflow-wrap: anywhere; }
.hist-note { margin-top: 1rem; }
@media (max-width: 1100px) { .grid { grid-template-columns: 1fr; } }
</style>
