<script setup lang="ts">
// Карточка уведомления (спека 007-evga-notices-ui; backend 009 FR-8/9).
// Проход 1 сближения с экраном платформы: степпер + табы + блок отправителя (спека 009-evga-outgoing-ui).
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

// комментарий к уведомлению (backend-спека 018 FR-2)
const commentDraft = ref('')
const commentBusy = ref(false)
watch(card, (c) => { commentDraft.value = c?.comment ?? '' }, { immediate: true })

async function saveComment() {
  commentBusy.value = true
  actionError.value = ''
  try {
    await evga.noticeSetComment(id.value, commentDraft.value)
    refresh()
  } catch (e) {
    actionError.value = apiErrorMessage(e)
  } finally {
    commentBusy.value = false
  }
}

// предпросмотр документа в модальном окне (PDF во встроенном просмотрщике браузера)
const previewOpen = ref(false)
const previewUrl = ref('')
const previewBusy = ref(false)

async function openPreview() {
  previewBusy.value = true
  actionError.value = ''
  try {
    previewUrl.value = await evga.noticePDFUrl(id.value)
    previewOpen.value = true
  } catch (e) {
    actionError.value = apiErrorMessage(e)
  } finally {
    previewBusy.value = false
  }
}

// освобождаем blob-URL, иначе документ висит в памяти вкладки
function closePreview() {
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
  }
}
onBeforeUnmount(closePreview)

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

// Отправка в ЕСЭДО (спека 014, вариант B). На заглушке реально ничего не уходит.
const esedoBusy = ref(false)
const esedoMsg = ref('')
async function sendESEDO() {
  esedoBusy.value = true
  actionError.value = ''
  esedoMsg.value = ''
  try {
    const r = await evga.noticeSendESEDO(id.value)
    esedoMsg.value = r.accepted ? `Отправлено в ЕСЭДО (${r.note || 'ок'})` : 'ЕСЭДО не приняла документ'
    refreshAll()
  } catch (e) {
    actionError.value = apiErrorMessage(e)
  } finally {
    esedoBusy.value = false
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
        <Button label="Предпросмотр" icon="pi pi-eye" outlined :loading="previewBusy" @click="openPreview" />
        <Button label="Скачать PDF" icon="pi pi-file-pdf" outlined :loading="pdfBusy" @click="downloadPDF" />
        <Button
          v-if="canWrite && outData?.exists"
          label="Отправить в ЕСЭДО"
          icon="pi pi-send"
          :loading="esedoBusy"
          @click="sendESEDO"
        />
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
      <Card class="mb">
        <template #content>
          <EvgaNoticeSteps :status-id="header?.status_id ?? null" :out="outData" />
        </template>
      </Card>

      <Message v-if="actionError" severity="error" :closable="true" class="mb">{{ actionError }}</Message>
      <Message v-if="esedoMsg" severity="success" :closable="true" class="mb">{{ esedoMsg }}</Message>

      <Tabs value="doc">
        <TabList>
          <Tab value="doc">Документ</Tab>
          <Tab value="preview">Предпросмотр</Tab>
          <Tab value="route">Ход исполнения</Tab>
          <Tab value="app">Приложение ({{ card.rows.length }})</Tab>
        </TabList>
        <TabPanels>
          <TabPanel value="doc">
            <Card class="mb">
              <template #title>Регистрационные реквизиты</template>
              <template #content>
                <dl class="props">
                  <dt>Номер документа</dt><dd>{{ header?.docnum || '—' }}</dd>
                  <dt>Статус</dt><dd>{{ header?.status_title || '—' }}</dd>
                  <dt>Дата создания</dt><dd>{{ header?.created_at ? fmtDate(header.created_at) : '—' }}</dd>
                  <dt>Инициатор</dt><dd>{{ card.initiator || '—' }}</dd>
                  <!-- адресаты с порядком (backend-спека 018) -->
                  <dt>Адресаты (ЕСЭДО)</dt>
                  <dd>
                    <template v-if="card.recipients?.length">
                      <div v-for="r in card.recipients" :key="r.cli_id" class="recipient">
                        <span class="recipient-nn">{{ r.nn }}.</span> {{ r.title }}
                      </div>
                    </template>
                    <template v-else>{{ header?.recipient || '—' }}</template>
                  </dd>
                  <dt>Записей в приложении</dt><dd>{{ header?.rows_count }}</dd>
                  <dt>Общая сумма</dt><dd>{{ fmtAmount(header?.total_sum || '0') }}</dd>
                  <template v-if="outData?.exists">
                    <dt>Исходящее письмо</dt><dd>{{ outData.doc_num || 'создано, ожидает регистрации' }}</dd>
                    <dt v-if="outData.doc_date">Дата регистрации</dt><dd v-if="outData.doc_date">{{ fmtDate(outData.doc_date) }}</dd>
                    <dt v-if="outData.exec_due">Срок исполнения</dt><dd v-if="outData.exec_due">{{ fmtDate(outData.exec_due) }}</dd>
                  </template>
                  <template v-else>
                    <dt>Исходящее письмо</dt><dd>—</dd>
                  </template>
                </dl>

                <!-- комментарий к уведомлению (backend-спека 018 FR-2) -->
                <div class="comment-block">
                  <label class="comment-label">Комментарий</label>
                  <Textarea v-model="commentDraft" rows="2" auto-resize fluid :disabled="!canWrite" placeholder="Произвольный комментарий к уведомлению" />
                  <div v-if="canWrite" class="comment-actions">
                    <Button label="Сохранить комментарий" icon="pi pi-save" size="small" text
                      :loading="commentBusy" :disabled="commentDraft === (card.comment || '')" @click="saveComment" />
                  </div>
                </div>
              </template>
            </Card>

            <Card class="mb">
              <template #title>Реквизиты отправителя</template>
              <template #content>
                <dl class="props">
                  <dt>Отправитель</dt><dd>{{ header?.department || '—' }}</dd>
                  <dt>ГУ</dt><dd>{{ header?.gu }} · {{ header?.sendername }}</dd>
                  <dt>Вложения</dt>
                  <dd>
                    <Button
                      label="Уведомление об устранении нарушения.pdf"
                      icon="pi pi-file-pdf"
                      link
                      class="attach"
                      :loading="pdfBusy"
                      @click="downloadPDF"
                    />
                  </dd>
                </dl>
              </template>
            </Card>

            <Card>
              <template #title>Текст уведомления</template>
              <template #content>
                <p class="notice-text">{{ card.notice_txt }}</p>
              </template>
            </Card>
          </TabPanel>

          <TabPanel value="preview">
            <Card>
              <template #title>Предпросмотр</template>
              <template #content>
                <p class="muted">
                  PDF формируется по утверждённому образцу (казахская и русская части). ЭЦП подписанта
                  проставляется при регистрации канцелярией.
                </p>
                <div class="preview-actions">
                  <Button label="Открыть предпросмотр" icon="pi pi-eye" :loading="previewBusy" @click="openPreview" />
                  <Button label="Скачать одним файлом" icon="pi pi-download" outlined :loading="pdfBusy" @click="downloadPDF" />
                </div>
              </template>
            </Card>
          </TabPanel>

          <TabPanel value="route">
            <EvgaRouteCard
              :notice-id="id"
              :status-id="header?.status_id ?? null"
              :can-write="canWrite"
              @changed="refreshAll"
            />
          </TabPanel>

          <TabPanel value="app">
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
          </TabPanel>
        </TabPanels>
      </Tabs>

      <Dialog v-model:visible="confirmDelete" modal header="Удалить проект уведомления?" :style="{ width: '26rem' }">
        <p class="confirm-text">Записи витрины снова станут доступны для формирования. Действие необратимо.</p>
        <template #footer>
          <Button label="Отмена" text severity="secondary" :disabled="deleting" @click="confirmDelete = false" />
          <Button label="Удалить" icon="pi pi-trash" severity="danger" :loading="deleting" @click="doDelete" />
        </template>
      </Dialog>

      <!-- предпросмотр документа: PDF во встроенном просмотрщике браузера -->
      <Dialog
        v-model:visible="previewOpen"
        modal
        maximizable
        :header="`Предпросмотр — уведомление ${header?.docnum || '(проект)'}`"
        :style="{ width: '92vw', maxWidth: '1400px' }"
        :content-style="{ padding: '0' }"
        @hide="closePreview"
      >
        <iframe v-if="previewUrl" :src="previewUrl" class="pdf-frame" title="Предпросмотр уведомления" />
        <template #footer>
          <Button label="Скачать PDF" icon="pi pi-download" outlined :loading="pdfBusy" @click="downloadPDF" />
          <Button label="Закрыть" text severity="secondary" @click="previewOpen = false" />
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
.recipient { line-height: 1.5; }
.recipient-nn { color: var(--ehd-ink-2); margin-right: 0.25rem; }
.comment-block { margin-top: 1rem; display: flex; flex-direction: column; gap: 0.4rem; }
.comment-label { font-size: 0.88rem; color: var(--ehd-ink-2); }
.comment-actions { display: flex; justify-content: flex-end; }
.muted { margin: 0 0 0.75rem; color: var(--ehd-ink-2); }
.attach :deep(.p-button-label) { text-decoration: underline; }
.data-table { border: 1px solid var(--ehd-border); border-radius: var(--ehd-radius-sm); overflow: hidden; }
.num { text-align: right; }
.confirm-text { margin: 0; color: var(--ehd-ink-2); }
.preview-actions { display: flex; gap: 0.5rem; flex-wrap: wrap; }
.pdf-frame { width: 100%; height: 78vh; border: 0; display: block; }
</style>
