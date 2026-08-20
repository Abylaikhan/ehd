<script setup lang="ts">
definePageMeta({ layout: 'auth' })

// Заглушка sandbox: форма без реальной аутентификации.
// Далее: POST /api/v1/auth/login, вход по ЭЦП через NCALayer, смена временного пароля.
const login = ref('')
const password = ref('')
</script>

<template>
  <div class="auth-card">
    <div class="auth-brand">
      <span class="auth-mark">ЕХД</span>
      <span class="auth-brand-sub">Единое хранилище данных</span>
    </div>

    <div class="auth-body">
      <h1 class="auth-title">Вход в систему</h1>
      <p class="auth-subtitle">Авторизуйтесь по логину или ЭЦП</p>

      <div class="field">
        <label for="login">Логин</label>
        <InputText id="login" v-model="login" autocomplete="username" placeholder="Введите логин" />
      </div>

      <div class="field">
        <label for="password">Пароль</label>
        <Password
          id="password"
          v-model="password"
          :feedback="false"
          toggle-mask
          placeholder="Введите пароль"
          fluid
        />
      </div>

      <Button label="Войти" icon="pi pi-sign-in" fluid />

      <Divider align="center"><span class="auth-or">или</span></Divider>

      <Button label="Войти по ЭЦП (NCALayer)" icon="pi pi-id-card" severity="secondary" outlined fluid />

      <Message severity="info" :closable="false" class="auth-note">
        Аутентификация будет подключена на этапе Auth Module
      </Message>
    </div>
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
.auth-note {
  margin-top: 0.25rem;
}
</style>
