<script setup lang="ts">
import { validateIIN } from '../utils/validators'

definePageMeta({ layout: 'auth', middleware: 'guest' })

const auth = useAuth()
const edsMode = useRuntimeConfig().public.edsMode as string

const login = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const submit = async () => {
  error.value = ''
  loading.value = true
  try {
    const res = await auth.login(login.value, password.value)
    await navigateTo(res.password_change_required ? '/change-password' : '/')
  } catch (e) {
    error.value = apiErrorMessage(e)
  } finally {
    loading.value = false
  }
}

// --- ЭЦП ---
const edsOpen = ref(false)
const edsError = ref('')
const edsLoading = ref(false)
// поля для sandbox-режима (без реального NCALayer)
const edsIin = ref('')
const edsBin = ref('')
const edsName = ref('')

const openEds = () => {
  edsError.value = ''
  edsOpen.value = true
}

const finishEds = async (res: { password_change_required: boolean }) => {
  edsOpen.value = false
  await navigateTo(res.password_change_required ? '/change-password' : '/')
}

// Реальный NCALayer: challenge → CMS-подпись → verify.
const edsNcaSubmit = async () => {
  edsError.value = ''
  edsLoading.value = true
  try {
    const ch = await auth.edsChallenge()
    const cms = await useNcaLayer().signChallenge(btoa(ch.challenge))
    const res = await auth.edsVerify(ch.challenge, cms)
    await finishEds(res)
  } catch (e) {
    // Ошибка NCALayer (не запущен/отмена) или отказ проверки на бэкенде.
    edsError.value = (e as { data?: unknown })?.data ? apiErrorMessage(e) : ncaErrorMessage(e)
  } finally {
    edsLoading.value = false
  }
}

// Sandbox: подпись вводится вручную (ИИН обязателен).
const edsStubSubmit = async () => {
  edsError.value = ''
  const iinErr = validateIIN(edsIin.value.trim())
  if (iinErr) {
    edsError.value = iinErr
    return
  }
  edsLoading.value = true
  try {
    const signedData = `${edsIin.value.trim()}|${edsBin.value.trim()}|${edsName.value.trim()}`
    const ch = await auth.edsChallenge()
    const res = await auth.edsVerify(ch.challenge, signedData)
    await finishEds(res)
  } catch (e) {
    edsError.value = 'Не удалось выполнить вход по ЭЦП. Проверьте корректность данных подписи.'
    void apiErrorMessage(e)
  } finally {
    edsLoading.value = false
  }
}
</script>

<template>
  <div class="auth-card">
    <div class="auth-brand">
      <span class="auth-mark">ЕХД</span>
      <span class="auth-brand-sub">Единое хранилище данных</span>
    </div>

    <form class="auth-body" @submit.prevent="submit">
      <h1 class="auth-title">Вход в систему</h1>
      <p class="auth-subtitle">Авторизуйтесь по логину или ЭЦП</p>

      <Message v-if="error" severity="error" :closable="false">{{ error }}</Message>

      <div class="field">
        <label for="login">Логин</label>
        <InputText id="login" v-model="login" autocomplete="username" placeholder="Введите логин" />
      </div>

      <div class="field">
        <label for="password">Пароль</label>
        <Password input-id="password" v-model="password" :feedback="false" toggle-mask placeholder="Введите пароль" fluid />
      </div>

      <Button type="submit" label="Войти" icon="pi pi-sign-in" :loading="loading" fluid />

      <Divider align="center"><span class="auth-or">или</span></Divider>

      <Button
        label="Войти по ЭЦП (NCALayer)"
        icon="pi pi-id-card"
        severity="secondary"
        outlined
        fluid
        type="button"
        @click="openEds"
      />

      <p class="auth-foot">
        Нет учётной записи?
        <NuxtLink to="/register">Зарегистрироваться</NuxtLink>
      </p>
    </form>

    <Dialog v-model:visible="edsOpen" modal header="Вход по ЭЦП" :style="{ width: '440px' }">
      <!-- Реальный NCALayer (режимы ncalayer/ncanode) -->
      <div v-if="edsMode !== 'stub'" class="eds-body">
        <Message severity="info" :closable="false">
          Убедитесь, что приложение <b>NCALayer</b> установлено и запущено, затем нажмите «Подписать» и
          выберите сертификат в NCALayer.
        </Message>
        <Message v-if="edsError" severity="error" :closable="false">
          {{ edsError }}
          <template v-if="/NCALayer/i.test(edsError)">
            <br /><a href="https://pki.gov.kz/ncalayer/" target="_blank" rel="noopener">Скачать NCALayer</a>
          </template>
        </Message>
        <Button label="Подписать через NCALayer" icon="pi pi-id-card" :loading="edsLoading" fluid @click="edsNcaSubmit" />
      </div>

      <!-- Sandbox (без NCALayer) -->
      <div v-else class="eds-body">
        <Message severity="info" :closable="false">
          В sandbox нет реального NCALayer — введите данные подписи вручную. Обязателен только ИИН.
        </Message>
        <div class="field">
          <label for="eds-iin">ИИН</label>
          <InputText id="eds-iin" v-model="edsIin" maxlength="12" placeholder="12 цифр" />
        </div>
        <div class="field">
          <label for="eds-bin">БИН (для юр. лица, опционально)</label>
          <InputText id="eds-bin" v-model="edsBin" maxlength="12" placeholder="123456789012" />
        </div>
        <div class="field">
          <label for="eds-name">ФИО (опционально)</label>
          <InputText id="eds-name" v-model="edsName" placeholder="Иван Иванов" />
        </div>
        <Message v-if="edsError" severity="error" :closable="false">{{ edsError }}</Message>
        <Button label="Подтвердить" icon="pi pi-check" :loading="edsLoading" fluid @click="edsStubSubmit" />
      </div>
    </Dialog>
  </div>
</template>

<style scoped>
.auth-card {
  width: 100%;
  max-width: 420px;
  background: var(--ehd-surface);
  border: 1px solid var(--ehd-border);
  border-radius: 16px;
  box-shadow: var(--ehd-shadow-md);
  overflow: hidden;
}
.auth-brand {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 1.5rem 1.75rem;
  color: #fff;
  background: linear-gradient(135deg, var(--ehd-brand), #163a63);
}
.auth-mark {
  font-weight: 800;
  letter-spacing: 0.08em;
  font-size: 1.35rem;
}
.auth-brand-sub {
  font-size: 0.78rem;
  color: rgba(255, 255, 255, 0.6);
}
.auth-body {
  padding: 1.75rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.auth-title {
  margin: 0;
  font-size: 1.35rem;
  font-weight: 700;
}
.auth-subtitle {
  margin: -0.5rem 0 0.25rem;
  font-size: 0.88rem;
  color: var(--ehd-ink-2);
}
.field {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}
.field label {
  font-size: 0.82rem;
  font-weight: 500;
  color: var(--ehd-ink-2);
}
.field :deep(.p-inputtext),
.field :deep(.p-password) {
  width: 100%;
}
.auth-or {
  font-size: 0.78rem;
  color: var(--ehd-muted);
}
.auth-foot {
  margin: 0;
  font-size: 0.85rem;
  color: var(--ehd-ink-2);
  text-align: center;
}
.eds-body {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}
</style>
