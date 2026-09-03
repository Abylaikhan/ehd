<script setup lang="ts">
definePageMeta({ middleware: ['auth', 'admin'] })

const admin = useReporterAdmin()
const { data: sourcesData } = await useAsyncData('admin-sources', () => admin.sources.list())
const sources = computed(() => sourcesData.value?.items ?? [])

const sourceId = ref('')
const database = ref('')
const table = ref('')
const databases = ref<string[]>([])
const tables = ref<{ name: string; kind: string }[]>([])
const loadingDb = ref(false)
const loadingTbl = ref(false)
const loadError = ref('')

// Источник по умолчанию — первый активный.
watchEffect(() => {
  if (!sourceId.value && sources.value.length) {
    sourceId.value = (sources.value.find((s) => s.status === 'active') || sources.value[0]).id
  }
})

watch(
  sourceId,
  async (id) => {
    database.value = ''
    table.value = ''
    databases.value = []
    tables.value = []
    if (!id || !import.meta.client) return
    loadingDb.value = true
    loadError.value = ''
    try {
      databases.value = (await admin.sources.databases(id)).items.map((d) => d.name)
    } catch (e) {
      loadError.value = apiErrorMessage(e)
    } finally {
      loadingDb.value = false
    }
  },
  { immediate: true },
)

watch(database, async (db) => {
  table.value = ''
  tables.value = []
  if (!db || !import.meta.client) return
  loadingTbl.value = true
  loadError.value = ''
  try {
    tables.value = (await admin.sources.tables(sourceId.value, db)).items.map((t) => ({ name: t.name, kind: t.kind }))
  } catch (e) {
    loadError.value = apiErrorMessage(e)
  } finally {
    loadingTbl.value = false
  }
})

const form = reactive({ name: '', slug: '', description: '' })
const creating = ref(false)
const createError = ref('')

async function create() {
  createError.value = ''
  if (!form.name.trim() || !/^[a-z0-9-]+$/.test(form.slug) || !sourceId.value || !database.value || !table.value) {
    createError.value = 'Заполните название, корректный slug (латиница/цифры/дефис), источник, базу и таблицу.'
    return
  }
  creating.value = true
  try {
    const v = await admin.views.create({
      name: form.name,
      slug: form.slug,
      description: form.description || undefined,
      data_source_id: sourceId.value,
      database: database.value,
      table: table.value,
    })
    await navigateTo(`/reporter/admin/views/${v.id}`)
  } catch (e) {
    createError.value = apiErrorMessage(e)
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <div>
    <PageHeader title="Новая витрина" description="Выберите источник, базу и таблицу, затем настройте колонки" />
    <Card>
      <template #content>
        <div class="form">
          <div class="grid3">
            <div class="field">
              <label>Источник</label>
              <Select
                v-model="sourceId"
                :options="sources"
                option-label="name"
                option-value="id"
                placeholder="Источник данных"
              />
            </div>
            <div class="field">
              <label>База данных</label>
              <Select
                v-model="database"
                :options="databases"
                :loading="loadingDb"
                :disabled="!sourceId"
                placeholder="Выберите базу"
              />
            </div>
            <div class="field">
              <label>Таблица</label>
              <Select
                v-model="table"
                :options="tables"
                option-label="name"
                option-value="name"
                :loading="loadingTbl"
                :disabled="!database"
                filter
                filter-placeholder="Поиск таблицы..."
                :reset-filter-on-hide="true"
                placeholder="Выберите таблицу/VIEW"
                :virtual-scroller-options="{ itemSize: 38 }"
              />
            </div>
          </div>

          <Message v-if="loadError" severity="error" :closable="false">{{ loadError }}</Message>

          <div class="grid2">
            <div class="field">
              <label>Название</label>
              <InputText v-model="form.name" placeholder="Например, Транзакции (проводки)" />
            </div>
            <div class="field">
              <label>Slug (латиница/цифры/дефис)</label>
              <InputText v-model="form.slug" placeholder="например, ehd-transactions" />
            </div>
          </div>
          <div class="field">
            <label>Описание</label>
            <InputText v-model="form.description" placeholder="Краткое назначение набора данных" />
          </div>

          <Message v-if="createError" severity="error" :closable="true">{{ createError }}</Message>

          <div class="actions">
            <Button label="Отмена" severity="secondary" text @click="navigateTo('/reporter/admin/views')" />
            <Button
              label="Создать черновик"
              icon="pi pi-check"
              :loading="creating"
              :disabled="!table"
              @click="create"
            />
          </div>
        </div>
      </template>
    </Card>
  </div>
</template>

<style scoped>
.form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  max-width: 900px;
}
.grid3 {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}
.grid2 {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}
.field label {
  font-size: 0.82rem;
  font-weight: 500;
  color: var(--ehd-ink-2);
}
.field :deep(.p-select),
.field :deep(.p-inputtext) {
  width: 100%;
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}
</style>
