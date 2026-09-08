<script setup lang="ts">
import type { EvgaParticipant, EvgaRouteStep } from '~~/shared/api/types'
import { routeComplete } from '../utils/routeForm'

// Блок «Маршрут» карточки уведомления (спека 008-evga-approval-ui; ТЗ §10).
const props = defineProps<{
  noticeId: number
  statusId: number | null
  canWrite: boolean
}>()

const emit = defineEmits<{ (e: 'changed'): void }>()

const evga = useEvga()

const { data: route, refresh: refreshRoute } = useLazyAsyncData(
  `evga-route-${props.noticeId}`,
  () => evga.routeGet(props.noticeId),
  { server: false },
)

// --- редактор шаблона ---
const approvers = ref<EvgaParticipant[]>([])
const outgoing = ref<EvgaParticipant | null>(null)
watch(
  route,
  (r) => {
    if (!r) return
    approvers.value = r.template
      .filter((s) => s.kind === 'approve')
      .map((s) => ({ id: s.assignee_id, name: s.assignee_name, login: '' }))
    const out = r.template.find((s) => s.kind === 'outgoing')
    outgoing.value = out ? { id: out.assignee_id, name: out.assignee_name, login: '' } : null
  },
  { immediate: true },
)

const suggestions = ref<EvgaParticipant[]>([])
async function search(e: { query: string }) {
  try {
    suggestions.value = (await evga.participants(props.noticeId, e.query)).items
  } catch {
    suggestions.value = []
  }
}

const newApprover = ref<EvgaParticipant | null>(null)
function addApprover() {
  const v = newApprover.value
  if (v && typeof v === 'object' && !approvers.value.some((a) => a.id === v.id)) {
    approvers.value.push(v)
  }
  newApprover.value = null
}
function removeApprover(i: number) {
  approvers.value.splice(i, 1)
}
function moveApprover(i: number, dir: -1 | 1) {
  const j = i + dir
  if (j < 0 || j >= approvers.value.length) return
  const arr = approvers.value
  ;[arr[i], arr[j]] = [arr[j]!, arr[i]!]
}

const editable = computed(() => (route.value?.editable ?? false) && props.canWrite)
const complete = computed(() => routeComplete(approvers.value.map((a) => a.id), outgoing.value?.id ?? null))

const busy = ref(false)
const error = ref('')
const info = ref('')

async function saveRoute() {
  busy.value = true
  error.value = ''
  try {
    await evga.routePut(props.noticeId, approvers.value.map((a) => a.id), outgoing.value!.id)
    info.value = 'Маршрут сохранён'
    refreshRoute()
  } catch (e) {
    error.value = apiErrorMessage(e)
  } finally {
    busy.value = false
  }
}

async function submit() {
  busy.value = true
  error.value = ''
  try {
    // маршрут сохраняется перед отправкой, чтобы кнопка работала в одно действие
    await evga.routePut(props.noticeId, approvers.value.map((a) => a.id), outgoing.value!.id)
    const res = await evga.noticeSubmit(props.noticeId)
    info.value = `Отправлено на согласование. Номер уведомления: ${res.docnum}`
    refreshRoute()
    emit('changed')
  } catch (e) {
    error.value = apiErrorMessage(e)
  } finally {
    busy.value = false
  }
}

// --- решения согласующего ---
const isApproving = computed(() => props.statusId === 2)
const rejectDialog = ref(false)
const rejectComment = ref('')

async function approve() {
  busy.value = true
  error.value = ''
  try {
    await evga.noticeApprove(props.noticeId)
    info.value = 'Уведомление согласовано и подписано'
    refreshRoute()
    emit('changed')
  } catch (e) {
    error.value = apiErrorMessage(e)
  } finally {
    busy.value = false
  }
}

async function reject() {
  busy.value = true
  error.value = ''
  try {
    await evga.noticeReject(props.noticeId, rejectComment.value.trim())
    rejectDialog.value = false
    rejectComment.value = ''
    info.value = 'Уведомление возвращено на доработку'
    refreshRoute()
    emit('changed')
  } catch (e) {
    error.value = apiErrorMessage(e)
  } finally {
    busy.value = false
  }
}

const kindLabel = (s: EvgaRouteStep) => (s.kind === 'approve' ? 'Согласование' : 'Создать исходящее')
const stepStatus = (s: EvgaRouteStep) => {
  if (s.status === 'open') return 'Открыта'
  if (s.status === 'done') return s.result === 'approved' ? 'Согласовано' : s.result === 'rejected' ? 'Возвращено' : 'Закрыта'
  return 'Ожидает'
}
const fmtTS = (v: string | null) => (v ? new Date(v).toLocaleString('ru-RU') : '—')
</script>

<template>
  <Card class="route-card">
    <template #title>Маршрут</template>
    <template #content>
      <Message v-if="info" severity="success" :closable="true" class="mb" @close="info = ''">{{ info }}</Message>
      <Message v-if="error" severity="error" :closable="true" class="mb" @close="error = ''">{{ error }}</Message>

      <template v-if="editable">
        <p class="section-label">Согласующие (по порядку; последний подписывает уведомление)</p>
        <ol v-if="approvers.length" class="approvers">
          <li v-for="(a, i) in approvers" :key="a.id">
            <span class="approver-name">{{ a.name }}</span>
            <span class="approver-actions">
              <Button icon="pi pi-angle-up" text rounded size="small" :disabled="i === 0" aria-label="Выше" @click="moveApprover(i, -1)" />
              <Button icon="pi pi-angle-down" text rounded size="small" :disabled="i === approvers.length - 1" aria-label="Ниже" @click="moveApprover(i, 1)" />
              <Button icon="pi pi-times" text rounded size="small" severity="danger" aria-label="Убрать" @click="removeApprover(i)" />
            </span>
          </li>
        </ol>
        <p v-else class="muted">Согласующие не назначены.</p>
        <div class="add-row">
          <AutoComplete
            v-model="newApprover"
            :suggestions="suggestions"
            option-label="name"
            placeholder="Добавить согласующего…"
            :min-length="0"
            @complete="search"
            @option-select="addApprover"
          />
        </div>

        <p class="section-label">Ответственный за создание исходящего</p>
        <AutoComplete
          v-model="outgoing"
          :suggestions="suggestions"
          option-label="name"
          placeholder="Сотрудник департамента…"
          :min-length="0"
          @complete="search"
        />

        <div class="actions">
          <Button label="Сохранить маршрут" icon="pi pi-save" outlined :loading="busy" :disabled="!complete" @click="saveRoute" />
          <Button label="Отправить на согласование" icon="pi pi-send" :loading="busy" :disabled="!complete" @click="submit" />
          <span v-if="!complete" class="muted">Нужны согласующий и ответственный за исходящее.</span>
        </div>
      </template>

      <template v-else-if="route">
        <p class="section-label">Этапы маршрута</p>
        <ul class="tpl-list">
          <li v-for="s in route.template" :key="s.id || s.step_nn">
            {{ s.step_nn }}. {{ kindLabel(s) }} — {{ s.assignee_name || s.assignee_id }}
          </li>
        </ul>
        <div v-if="isApproving && canWrite" class="actions">
          <Button label="Согласовать и подписать" icon="pi pi-check-circle" severity="success" :loading="busy" @click="approve" />
          <Button label="Вернуть на доработку" icon="pi pi-undo" severity="danger" outlined :loading="busy" @click="rejectDialog = true" />
        </div>
      </template>

      <template v-if="route && route.history.length">
        <p class="section-label">Ход исполнения</p>
        <DataTable :value="route.history" size="small" striped-rows class="hist-table">
          <Column header="Раунд" field="round" :style="{ width: '5rem' }" />
          <Column header="Этап">
            <template #body="{ data: s }">{{ kindLabel(s) }}</template>
          </Column>
          <Column header="Исполнитель" field="assignee_name" />
          <Column header="Статус">
            <template #body="{ data: s }">{{ stepStatus(s) }}</template>
          </Column>
          <Column header="Комментарий">
            <template #body="{ data: s }">{{ s.comment || '—' }}</template>
          </Column>
          <Column header="Открыт">
            <template #body="{ data: s }">{{ fmtTS(s.opened_at) }}</template>
          </Column>
          <Column header="Закрыт">
            <template #body="{ data: s }">{{ fmtTS(s.closed_at) }}</template>
          </Column>
        </DataTable>
      </template>

      <Dialog v-model:visible="rejectDialog" modal header="Вернуть на доработку" :style="{ width: '28rem' }">
        <label class="field">
          <span>Комментарий (обязателен)</span>
          <Textarea v-model="rejectComment" rows="3" auto-resize placeholder="Что нужно исправить…" />
        </label>
        <template #footer>
          <Button label="Отмена" text severity="secondary" :disabled="busy" @click="rejectDialog = false" />
          <Button label="Вернуть" icon="pi pi-undo" severity="danger" :loading="busy" :disabled="!rejectComment.trim()" @click="reject" />
        </template>
      </Dialog>
    </template>
  </Card>
</template>

<style scoped>
.mb { margin-bottom: 1rem; }
.section-label { margin: 1rem 0 0.4rem; font-size: 0.85rem; font-weight: 600; color: var(--ehd-ink-2); }
.section-label:first-of-type { margin-top: 0; }
.approvers { margin: 0; padding-left: 1.3rem; }
.approvers li { display: flex; align-items: center; gap: 0.4rem; padding: 0.15rem 0; }
.approver-name { flex: 0 1 auto; }
.approver-actions { display: inline-flex; }
.add-row { margin-top: 0.4rem; }
.muted { font-size: 0.85rem; color: var(--p-text-muted-color); }
.actions { display: flex; align-items: center; gap: 0.75rem; margin-top: 1.25rem; flex-wrap: wrap; }
.tpl-list { margin: 0; padding-left: 1.2rem; }
.hist-table { border: 1px solid var(--ehd-border); border-radius: var(--ehd-radius-sm); overflow: hidden; }
.field { display: flex; flex-direction: column; gap: 0.3rem; font-size: 0.88rem; color: var(--ehd-ink-2); width: 100%; }
</style>
