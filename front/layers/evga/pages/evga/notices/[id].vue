<script setup lang="ts">
// Карточка уведомления (спека 007-evga-notices-ui; backend 009 FR-8/9).
definePageMeta({ middleware: 'auth' })

const route = useRoute()
const evga = useEvga()
const session = useSessionStore()
const id = computed(() => Number(route.params.id))

const canWrite = computed(() => session.isAdmin || (session.user?.roles ?? []).includes('evga_auditor'))

const { data: card, pending, error, refresh } = useLazyAsyncData(
  `evga-notice-${id.value}`,
  () => evga.noticeGet(id.value),
  { server: false },
)

// исходящее письмо (спека 011 FR-7)
const { data: outData, refresh: refreshOut } = useLazyAsyncData(
  `evga-notice-out-${id.value}`,
  () => evga.noticeOut(id.value),
  { server: false },
)
function refreshAll() {
  refresh()
  refreshOut()
}

const header = computed(() => card.value?.header)
const isDraft = computed(() => header.value?.status_id === 1)

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
  if (!card.value) return 'loading'
  return 'ready'
})

const pdfBusy = ref(false)
async function downloadPDF() {
  pdfBusy.value = true
  actionError.value = ''
  try {
    await evga.noticePDF(id.value)
  } catch (e) {
    actionError.value = apiErrorMessage(e)
  } finally {
    pdfBusy.value = false
  }
}

const deleting = ref(false)
const actionError = ref('')
const confirmDelete = ref(false)
async function doDelete() {
  deleting.value = true
  actionError.value = ''
  try {
    await evga.noticeDelete(id.value)
    await navigateTo('/evga/notices')
  } catch (e) {
    actionError.value = apiErrorMessage(e)
    confirmDelete.value = false
  } finally {
    deleting.value = false
  }
}

const fmtAmount = (v: string) => {
  const n = Number(v)
  return Number.isFinite(n) ? n.toLocaleString('ru-RU', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : v
}
const fmtDate = (v: string | null) => {
  if (!v) return '—'
  const [y, m, d] = v.split('-')
  return `${d}.${m}.${y}`
}
const fio = (r: { fm: string; nm: string; ft: string }) => [r.fm, r.nm, r.ft].filter(Boolean).join(' ') || '—'
</script>

<template>
  <div>
    <PageHeader
      :title="header?.docnum ? `Уведомление № ${header.docnum}` : `Уведомление (проект #${id})`"
      :description="header ? `${header.status_title || '—'} · ГУ ${header.gu} · ${header.sendername}` : undefined"
    >
      <template #actions>
        <Button label="Скачать PDF" icon="pi pi-file-pdf" outlined :loading="pdfBusy" @click="downloadPDF" />
        <Button
          v-if="canWrite && isDraft"
          label="Удалить проект"
          icon="pi pi-trash"
          severity="danger"
          outlined
          :loading="deleting"
          @click="confirmDelete = true"
        />
        <NuxtLink to="/evga/notices"><Button label="К списку" icon="pi pi-arrow-left" text /></NuxtLink>
      </template>
    </PageHeader>

    <ErrorState v-if="screenState === 'denied'" title="Доступ запрещён" message="У вас нет прав на просмотр этого уведомления." />
    <ErrorState v-else-if="screenState === 'notfound'" title="Не найдено" message="Уведомление не существует или относится к другому департаменту." />
    <ErrorState v-else-if="screenState === 'source'" title="Источник недоступен" message="База данных ОБМ ЕВГА временно недоступна." retryable @retry="refresh" />
    <ErrorState v-else-if="screenState === 'error'" message="Не удалось загрузить уведомление." retryable @retry="refresh" />
    <div v-else-if="screenState === 'loading'" class="center"><ProgressSpinner style="width: 2.5rem; height: 2.5rem" /></div>

    <template v-else-if="card">
      <Message v-if="actionError" severity="error" :closable="true" class="mb">{{ actionError }}</Message>

      <Card class="mb">
        <template #title>Реквизиты</template>
        <template #content>
          <dl class="props">
            <dt>Статус</dt><dd>{{ header?.status_title || '—' }}</dd>
            <dt>Адресат (ЕСЭДО)</dt><dd>{{ header?.recipient || '—' }}</dd>
            <dt>Записей в приложении</dt><dd>{{ header?.rows_count }}</dd>
            <dt>Общая сумма</dt><dd>{{ fmtAmount(header?.total_sum || '0') }}</dd>
            <dt>Департамент</dt><dd>{{ header?.department || '—' }}</dd>
            <template v-if="outData?.exists">
              <dt>Исходящее письмо</dt><dd>{{ outData.doc_num || 'создано, ожидает регистрации' }}</dd>
              <dt v-if="outData.doc_date">Дата регистрации</dt><dd v-if="outData.doc_date">{{ fmtDate(outData.doc_date) }}</dd>
              <dt v-if="outData.exec_due">Срок исполнения</dt><dd v-if="outData.exec_due">{{ fmtDate(outData.exec_due) }}</dd>
            </template>
          </dl>
        </template>
      </Card>

      <Card class="mb">
        <template #title>Текст уведомления</template>
        <template #content>
          <p class="notice-text">{{ card.notice_txt }}</p>
        </template>
      </Card>

      <EvgaRouteCard
        :notice-id="id"
        :status-id="header?.status_id ?? null"
        :can-write="canWrite"
        class="mb"
        @changed="refreshAll"
      />

      <Card>
        <template #title>Приложение — записи витрины ({{ card.rows.length }})</template>
        <template #content>
          <DataTable :value="card.rows" size="small" striped-rows scrollable class="data-table">
            <Column header="№" :style="{ width: '3.5rem' }">
              <template #body="{ index }">{{ index + 1 }}</template>
            </Column>
            <Column field="gu" header="Код ГУ" />
            <Column field="ppo_pp" header="№ платежа" />
            <Column header="Дата платежа">
              <template #body="{ data: r }">{{ fmtDate(r.paymentdate) }}</template>
            </Column>
            <Column header="ФИО получателя" :style="{ minWidth: '12rem' }">
              <template #body="{ data: r }">{{ fio(r) }}</template>
            </Column>
            <Column field="iin" header="ИИН" />
            <Column field="la1" header="Карт-счёт" />
            <Column header="Сумма" class="num">
              <template #body="{ data: r }">{{ fmtAmount(r.amount_part) }}</template>
            </Column>
          </DataTable>
        </template>
      </Card>

      <Dialog v-model:visible="confirmDelete" modal header="Удалить проект уведомления?" :style="{ width: '26rem' }">
        <p class="confirm-text">Записи витрины снова станут доступны для формирования. Действие необратимо.</p>
        <template #footer>
          <Button label="Отмена" text severity="secondary" :disabled="deleting" @click="confirmDelete = false" />
          <Button label="Удалить" icon="pi pi-trash" severity="danger" :loading="deleting" @click="doDelete" />
        </template>
      </Dialog>
    </template>
  </div>
</template>

<style scoped>
.center { display: flex; justify-content: center; padding: 2.5rem 0; }
.mb { margin-bottom: 1rem; }
.props { display: grid; grid-template-columns: minmax(14rem, auto) 1fr; gap: 0.45rem 1rem; margin: 0; }
.props dt { color: var(--ehd-ink-2); font-size: 0.88rem; }
.props dd { margin: 0; color: var(--ehd-ink); font-size: 0.92rem; }
.notice-text { margin: 0; line-height: 1.6; color: var(--ehd-ink); }
.data-table { border: 1px solid var(--ehd-border); border-radius: var(--ehd-radius-sm); overflow: hidden; }
.num { text-align: right; }
.confirm-text { margin: 0; color: var(--ehd-ink-2); }
</style>
