import type { Me } from '~~/shared/api/types'

// Pinia — только кросс-страничное состояние (ТЗ): текущая сессия/личность.
export const useSessionStore = defineStore('session', {
  state: () => ({
    user: null as Me | null,
  }),
  getters: {
    isAuthenticated: (s): boolean => s.user !== null,
    isAdmin: (s): boolean => s.user?.is_admin ?? false,
  },
  actions: {
    set(user: Me | null) {
      this.user = user
    },
  },
})
