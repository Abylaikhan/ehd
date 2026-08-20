// Pinia — только для кросс-страничного состояния (требование ТЗ): сессия здесь.
export const useSessionStore = defineStore('session', {
  state: () => ({
    user: null as null | { id: string; login: string; roles: string[] },
  }),
  getters: {
    isAuthenticated: (state) => state.user !== null,
    isAdmin: (state) => state.user?.roles.includes('admin') ?? false,
  },
})
