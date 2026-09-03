<script setup lang="ts">
import { validatePassword, validateIIN, validateEmail, required } from '../utils/validators'

definePageMeta({ layout: 'auth', middleware: 'guest' })

const auth = useAuth()
const form = reactive({ login: '', password: '', iin: '', full_name: '', email: '', phone: '' })
const errors = reactive<Record<string, string>>({})
const serverError = ref('')
const done = ref(false)
const loading = ref(false)

const validate = (): boolean => {
  for (const k of Object.keys(errors)) delete errors[k]
  const rules: Record<string, string | null> = {
    login: required(form.login, 'Логин'),
    full_name: required(form.full_name, 'ФИО'),
    email: validateEmail(form.email),
    iin: validateIIN(form.iin),
    password: validatePassword(form.password),
  }
  for (const [field, msg] of Object.entries(rules)) {
    if (msg) errors[field] = msg
  }
  return Object.keys(errors).length === 0
}

const submit = async () => {
  serverError.value = ''
  if (!validate()) return
  loading.value = true
  try {
    await auth.register({ ...form })
    done.value = true
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
      <span class="auth-brand-sub">Регистрация</span>
    </div>

    <div v-if="done" class="auth-body">
      <Message severity="success" :closable="false">
        Заявка принята. Учётная запись ожидает проверки и назначения роли администратором.
      </Message>
      <NuxtLink to="/login"><Button label="К входу" icon="pi pi-arrow-left" outlined fluid /></NuxtLink>
    </div>

    <form v-else class="auth-body" @submit.prevent="submit">
      <Message v-if="serverError" severity="error" :closable="false">{{ serverError }}</Message>

      <div class="field">
        <label for="full_name">ФИО</label>
        <InputText id="full_name" v-model="form.full_name" :invalid="!!errors.full_name" />
        <small v-if="errors.full_name" class="err">{{ errors.full_name }}</small>
      </div>
      <div class="field">
        <label for="iin">ИИН</label>
        <InputText id="iin" v-model="form.iin" maxlength="12" :invalid="!!errors.iin" />
        <small v-if="errors.iin" class="err">{{ errors.iin }}</small>
      </div>
      <div class="field">
        <label for="email">Email</label>
        <InputText id="email" v-model="form.email" :invalid="!!errors.email" />
        <small v-if="errors.email" class="err">{{ errors.email }}</small>
      </div>
      <div class="field">
        <label for="login">Логин</label>
        <InputText id="login" v-model="form.login" autocomplete="username" :invalid="!!errors.login" />
        <small v-if="errors.login" class="err">{{ errors.login }}</small>
      </div>
      <div class="field">
        <label for="password">Пароль</label>
        <Password input-id="password" v-model="form.password" toggle-mask :invalid="!!errors.password" fluid />
        <small v-if="errors.password" class="err">{{ errors.password }}</small>
      </div>

      <Button type="submit" label="Зарегистрироваться" icon="pi pi-user-plus" :loading="loading" fluid />
      <p class="auth-foot">Уже есть учётная запись? <NuxtLink to="/login">Войти</NuxtLink></p>
    </form>
  </div>
</template>

<style scoped>
.auth-card {
  width: 100%;
  max-width: 440px;
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
.field :deep(.p-inputtext),
.field :deep(.p-password) {
  width: 100%;
}
.err {
  color: var(--ehd-critical);
  font-size: 0.78rem;
}
.auth-foot {
  margin: 0;
  font-size: 0.85rem;
  color: var(--ehd-ink-2);
  text-align: center;
}
</style>
