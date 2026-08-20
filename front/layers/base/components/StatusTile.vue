<script setup lang="ts">
// Stat-tile здоровья компонента. Статус всегда несёт иконку + подпись,
// а не только цвет (правило доступности). Цвета — из зарезервированной статус-палитры.
type Status = 'good' | 'warning' | 'critical' | 'unknown'

const props = defineProps<{
  category: string
  name: string
  status: Status
  meta?: string
}>()

const MAP: Record<Status, { label: string; icon: string }> = {
  good: { label: 'Доступен', icon: 'pi pi-check-circle' },
  warning: { label: 'Внимание', icon: 'pi pi-exclamation-triangle' },
  critical: { label: 'Недоступен', icon: 'pi pi-times-circle' },
  unknown: { label: 'Неизвестно', icon: 'pi pi-question-circle' },
}

const view = computed(() => MAP[props.status])
</script>

<template>
  <article class="tile" :data-status="status">
    <div class="tile-head">
      <span class="tile-category">{{ category }}</span>
      <span class="tile-dot" aria-hidden="true" />
    </div>

    <p class="tile-name">{{ name }}</p>

    <div class="tile-status">
      <i :class="view.icon" aria-hidden="true" />
      <span>{{ view.label }}</span>
    </div>

    <p v-if="meta" class="tile-meta">{{ meta }}</p>
  </article>
</template>

<style scoped>
.tile {
  --c: var(--ehd-unknown);
  --c-soft: var(--ehd-unknown-soft);
  position: relative;
  background: var(--ehd-surface);
  border: 1px solid var(--ehd-border);
  border-radius: var(--ehd-radius);
  padding: 1.15rem 1.25rem;
  box-shadow: var(--ehd-shadow-sm);
  overflow: hidden;
  transition: box-shadow 0.15s ease, transform 0.15s ease;
}
.tile:hover {
  box-shadow: var(--ehd-shadow-md);
  transform: translateY(-1px);
}
.tile::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 4px;
  background: var(--c);
}
.tile[data-status='good'] {
  --c: var(--ehd-good);
  --c-soft: var(--ehd-good-soft);
}
.tile[data-status='warning'] {
  --c: var(--ehd-warning);
  --c-soft: var(--ehd-warning-soft);
}
.tile[data-status='critical'] {
  --c: var(--ehd-critical);
  --c-soft: var(--ehd-critical-soft);
}

.tile-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.tile-category {
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--ehd-muted);
}
.tile-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--c);
  box-shadow: 0 0 0 4px var(--c-soft);
}
.tile-name {
  margin: 0.55rem 0 0.9rem;
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--ehd-ink);
}
.tile-status {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.28rem 0.6rem;
  border-radius: 999px;
  background: var(--c-soft);
  color: var(--c);
  font-size: 0.82rem;
  font-weight: 600;
}
.tile-status i {
  font-size: 0.9rem;
}
.tile-meta {
  margin: 0.75rem 0 0;
  font-size: 0.78rem;
  color: var(--ehd-muted);
  font-variant-numeric: tabular-nums;
}
</style>
