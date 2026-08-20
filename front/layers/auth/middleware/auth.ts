export default defineNuxtRouteMiddleware(() => {
  const session = useSessionStore()
  if (!session.isAuthenticated) {
    return navigateTo('/login')
  }
})
