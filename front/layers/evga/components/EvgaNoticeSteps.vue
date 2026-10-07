<script setup lang="ts">
// Прогресс-степпер уведомления (спека 009-evga-outgoing-ui): Создан → Подписание →
// Канцелярия → Выполнен. Логика шага — в utils/noticeSteps.ts (покрыта юнитами).
import { NOTICE_STEP_LABELS, currentNoticeStep, type NoticeStepOut } from '../utils/noticeSteps'

const props = defineProps<{
  statusId: number | null
  out?: NoticeStepOut | null
}>()

const steps = NOTICE_STEP_LABELS
const current = computed(() => currentNoticeStep(props.statusId, props.out))

function stepClass(i: number): 'done' | 'current' | 'pending' {
  if (i < current.value) return 'done'
  if (i === current.value) return 'current'
  return 'pending'
}
</script>

<template>
  <ol class="steps">
    <li v-for="(label, i) in steps" :key="i" class="step" :class="stepClass(i)">
      <span v-if="i > 0" class="line" />
      <span class="dot" />
      <span class="label">{{ label }}</span>
    </li>
  </ol>
</template>

<style scoped>
.steps {
  display: flex;
  align-items: flex-start;
  margin: 0;
  padding: 0.5rem 0 1rem;
  list-style: none;
}
.step {
  position: relative;
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.dot {
  width: 0.85rem;
  height: 0.85rem;
  border-radius: 50%;
  background: var(--ehd-border);
  z-index: 1;
}
.label {
  margin-top: 0.45rem;
  font-size: 0.82rem;
  color: var(--ehd-ink-2);
  text-align: center;
}
/* соединительная линия слева от точки (кроме первого шага) */
.line {
  position: absolute;
  top: calc(0.425rem - 1px);
  right: 50%;
  width: 100%;
  height: 2px;
  background: var(--ehd-border);
}
.step.done .dot,
.step.done .line { background: #22c55e; }
.step.done .label { color: var(--ehd-ink); }
.step.current .dot {
  background: #f97316;
  box-shadow: 0 0 0 3px color-mix(in srgb, #f97316 25%, transparent);
}
.step.current .label { color: var(--ehd-ink); font-weight: 600; }
</style>
