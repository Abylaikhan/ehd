export default defineNuxtRouteMiddleware(() => {
  const session = useSessionStore()
  if (!session.isAuthenticated) {
    return navigateTo('/login')
  }
  // Скрытие в UI — не механизм безопасности: backend повторно авторизует каждый запрос.
  if (!session.isAdmin) {
    return navigateTo('/')
  }
})
