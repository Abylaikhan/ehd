<script setup lang="ts">
import type { EvgaReference, EvgaStatusChange } from '~~/shared/api/types'
import { requiredFields, validateForm } from '../utils/statusFlow'

// Диалог смены статуса (спека 006-evga-statuses-ui): общий для одиночной (карточка)
// и массовой (реестр) операций. Поля показываются по условиям перехода §7.2.
const props = defineProps<{
  visible: boolean
  /** Целевые статусы, доступные для выбора (уже отфильтрованы по матрице). */
  targets: EvgaReference[]
  activities: EvgaReference[]
  /** Заголовок: «Сменить статус записи …» / «Сменить статус N записей». */
  title: string
  busy: boolean
  /** Ошибка backend (422 и пр.) — показывается внутри диалога. */
  error: string
}>()

const emit = defineEmits<{
  (e: 'update:visible', v: boolean): void
  (e: 'submit', change: EvgaStatusChange): void
}>()

const statusId = ref<number | null>(null)
const note = ref('')
const amount = ref('')
const refund = ref('')
const activityId = ref<number | null>(null)
const localError = ref('')

watch(
  () => props.visible,
  (v) => {
    if (v) {
      statusId.value = props.targets.length === 1 ? props.targets[0]!.id : null
      note.value = amount.value = refund.value = ''
      activityId.value = null
      localError.value = ''
    }
  },
)

const req = computed(() => requiredFields(statusId.value))
const statusOptions = computed(() => props.targets.map((s) => ({ label: s.title, value: s.id })))
const activityOptions = computed(() => props.activities.map((a) => ({ label: a.title, value: a.id })))

function submit() {
  localError.value =
    validateForm(statusId.value, {
      note: note.value,
      amount: amount.value,
      refund: refund.value,
      activityId: activityId.value,
    }) ?? ''
  if (localError.value) return
  const change: EvgaStatusChange = { status_id: statusId.value! }
  if (note.value.trim()) change.note = note.value.trim()
  if (req.value.amount) change.amount_for_vozvrat = amount.value
  if (req.value.refund) change.refund = refund.value
  if (req.value.showActivity && activityId.value) change.activity_id = activityId.value
  emit('submit', change)
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    :header="title"
    :style="{ width: '30rem' }"
    @update:visible="emit('update:visible', $event)"
  >
    <div class="form">
      <label class="field">
        <span>Целевой статус</span>
        <Select v-model="statusId" :options="statusOptions" option-label="label" option-value="value" placeholder="Выберите статус" />
      </label>

      <label class="field">
        <span>Комментарий <b v-if="req.note" class="req">*</b></span>
        <Textarea v-model="note" rows="3" auto-resize :placeholder="req.note ? 'Обязателен для «Не подтверждено»' : 'Необязательно'" />
      </label>

      <label v-if="req.amount" class="field">
        <span>Сумма к возмещению <b class="req">*</b></span>
        <InputText v-model="amount" placeholder="Например 100000.00" inputmode="decimal" />
      </label>

      <label v-if="req.refund" class="field">
        <span>Сумма возмещения <b class="req">*</b></span>
        <InputText v-model="refund" placeholder="Например 100000.00" inputmode="decimal" />
      </label>

      <label v-if="req.showActivity" class="field">
        <span>Аудиторское мероприятие (необязательно)</span>
        <Select v-model="activityId" :options="activityOptions" option-label="label" option-value="value" placeholder="Выберите мероприятие" filter show-clear />
      </label>

      <Message v-if="localError || error" severity="error" :closable="false">{{ localError || error }}</Message>
    </div>

    <template #footer>
      <Button label="Отмена" text severity="secondary" :disabled="busy" @click="emit('update:visible', false)" />
      <Button label="Применить" icon="pi pi-check" :loading="busy" :disabled="statusId == null" @click="submit" />
    </template>
  </Dialog>
</template>

<style scoped>
.form { display: flex; flex-direction: column; gap: 0.9rem; }
.field { display: flex; flex-direction: column; gap: 0.3rem; font-size: 0.88rem; color: var(--ehd-ink-2); }
.req { color: var(--p-red-500, #e24c4c); }
</style>
