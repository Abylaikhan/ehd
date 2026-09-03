<script setup lang="ts">
// Дашборд состояния sandbox: hero-сводка + KPI-плитки компонентов.
type Status = 'good' | 'warning' | 'critical' | 'unknown'
const api = useApi()

const { data: ready, error: readyError, pending, refresh } = await useAsyncData(
  'readyz',
  () => api<Record<string, string>>('/readyz'),
)
const { data: auth, refresh: refreshAuth } = await useAsyncData('auth-ping', () =>
  api<{ status: string }>('/api/v1/auth/ping').catch(() => null),
)
const { data: reporter, refresh: refreshReporter } = await useAsyncData('reporter-ping', () =>
  api<{ status: string }>('/api/v1/reporter/ping').catch(() => null),
)

const toStatus = (ok: boolean | undefined): Status =>
  readyError.value ? 'critical' : ok === undefined ? 'unknown' : ok ? 'good' : 'critical'

const components = computed(() => [
  { category: 'База данных', name: 'PostgreSQL', status: toStatus(ready.value?.postgres === 'ok'), meta: 'Основная БД · localhost:5433' },
  { category: 'Источник', name: 'ClickHouse', status: toStatus(ready.value?.clickhouse === 'ok'), meta: 'Read-only · localhost:8123' },
  { category: 'Модуль', name: 'Auth Module', status: toStatus(auth.value?.status === 'ok'), meta: '/api/v1/auth' },
  { category: 'Модуль', name: 'Reporter Module', status: toStatus(reporter.value?.status === 'ok'), meta: '/api/v1/reporter' },
] as const)

const healthy = computed(() => components.value.filter((c) => c.status === 'good').length)
const total = computed(() => components.value.length)
const allOk = computed(() => healthy.value === total.value)

const refreshAll = () => Promise.all([refresh(), refreshAuth(), refreshReporter()])
</script>

<template>
  <div>
    <PageHeader
      title="Панель мониторинга"
      description="Состояние компонентов локального окружения ЕХД БО"
    >
      <template #actions>
        <Button
          label="Обновить"
          icon="pi pi-refresh"
          :loading="pending"
          severity="secondary"
          outlined
          @click="refreshAll"
        />
      </template>
    </PageHeader>

    <section class="hero" :data-ok="allOk">
      <div class="hero-figure">
        <span class="hero-number">{{ healthy }}<span class="hero-total">/{{ total }}</span></span>
        <span class="hero-caption">компонентов доступно</span>
      </div>
      <div class="hero-status">
        <i :class="allOk ? 'pi pi-verified' : 'pi pi-exclamation-circle'" aria-hidden="true" />
        <div>
          <p class="hero-status-title">{{ allOk ? 'Система работает штатно' : 'Есть недоступные компоненты' }}</p>
          <p class="hero-status-sub">Sandbox · Docker Compose</p>
        </div>
      </div>
    </section>

    <div class="grid">
      <StatusTile
        v-for="c in components"
        :key="c.name"
        :category="c.category"
        :name="c.name"
        :status="c.status"
        :meta="c.meta"
      />
    </div>
  </div>
</template>

<style scoped>
.hero {
  display: flex;
  align-items: center;
  gap: 2rem;
  flex-wrap: wrap;
  padding: 1.5rem 1.75rem;
  margin-bottom: 1.25rem;
  border-radius: var(--ehd-radius);
  color: #eaf0f8;
  background:
    radial-gradient(600px 200px at 100% 0%, rgba(255, 255, 255, 0.08), transparent 70%),
    linear-gradient(135deg, var(--ehd-brand), #163a63);
  box-shadow: var(--ehd-shadow-md);
}
.hero-figure {
  display: flex;
  flex-direction: column;
  padding-right: 2rem;
  border-right: 1px solid rgba(255, 255, 255, 0.15);
}
.hero-number {
  font-size: 3rem;
  font-weight: 700;
  line-height: 1;
  color: #fff;
}
.hero-total {
  font-size: 1.5rem;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.55);
}
.hero-caption {
  margin-top: 0.35rem;
  font-size: 0.85rem;
  color: rgba(255, 255, 255, 0.7);
}
.hero-status {
  display: flex;
  align-items: center;
  gap: 0.9rem;
}
.hero-status i {
  font-size: 1.9rem;
  color: #fff;
}
.hero-status-title {
  margin: 0;
  font-size: 1.05rem;
  font-weight: 600;
  color: #fff;
}
.hero-status-sub {
  margin: 0.15rem 0 0;
  font-size: 0.8rem;
  color: rgba(255, 255, 255, 0.6);
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 1rem;
}
</style>
