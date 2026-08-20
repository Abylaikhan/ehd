// Загружает /auth/me при инициализации (SSR + гидрация), чтобы guard'ы и shell
// знали статус аутентификации до рендера страницы.
export default defineNuxtPlugin(async () => {
  const session = useSessionStore()
  if (!session.user) {
    await useAuth().fetchMe()
  }
})
