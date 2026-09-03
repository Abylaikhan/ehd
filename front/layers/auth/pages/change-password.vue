<script setup lang="ts">
import { validatePassword } from '../utils/validators'

definePageMeta({ layout: 'auth', middleware: 'auth' })

const auth = useAuth()
const session = useSessionStore()
// У входа по ЭЦП пароля может не быть — тогда это установка первого пароля (без старого).
const hasPassword = computed(() => session.user?.has_password ?? true)
const title = computed(() => (hasPassword.value ? 'Смена пароля' : 'Установка пароля'))

const oldPassword = ref('')
const newPassword = ref('')
const fieldError = ref('')
const serverError = ref('')
const done = ref(false)
const loading = ref(false)

const submit = async () => {
  serverError.value = ''
  fieldError.value = validatePassword(newPassword.value) || ''
  if (fieldError.value) return
  loading.value = true
  try {
    await auth.changePassword(oldPassword.value, newPassword.value)
    done.value = true
    setTimeout(() => navigateTo('/'), 900)
  } catch (e) {
    serverError.value = apiErrorMessage(e)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-card">
    <div class="auth-brand">
      <span class="auth-mark">ЕХД БО</span>
      <span class="auth-brand-sub">{{ title }}</span>
    </div>

    <form class="auth-body" @submit.prevent="submit">
      <Message severity="info" :closable="false">
        <template v-if="hasPassword">Задайте новый пароль: минимум 8 символов, заглавная, строчная и цифра.</template>
        <template v-else>Вы вошли по ЭЦП — задайте пароль для входа по логину. Минимум 8 символов, заглавная, строчная и цифра.</template>
      </Message>
      <Message v-if="serverError" severity="error" :closable="false">{{ serverError }}</Message>
      <Message v-if="done" severity="success" :closable="false">Готово. Переходим…</Message>

      <div v-if="hasPassword" class="field">
        <label for="old">Текущий пароль</label>
        <Password input-id="old" v-model="oldPassword" :feedback="false" toggle-mask fluid />
      </div>
      <div class="field">
        <label for="new">Новый пароль</label>
        <Password input-id="new" v-model="newPassword" toggle-mask :invalid="!!fieldError" fluid />
        <small v-if="fieldError" class="err">{{ fieldError }}</small>
      </div>

      <Button type="submit" :label="hasPassword ? 'Сменить пароль' : 'Установить пароль'" icon="pi pi-key" :loading="loading" fluid />
    </form>
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
  padding: 1.35rem 1.75rem;
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
  padding: 1.5rem 1.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
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
.field :deep(.p-password) {
  width: 100%;
}
.err {
  color: var(--ehd-critical);
  font-size: 0.78rem;
}
</style>
