import type { LoginResponse, Me, RegisterRequest } from '~~/shared/api/types'

// Действия аутентификации: вызывают Auth Module и синхронизируют session-стор.
export function useAuth() {
  const api = useApi()
  const session = useSessionStore()

  const fetchMe = async () => {
    try {
      session.set(await api<Me>('/api/v1/auth/me'))
    } catch {
      session.set(null)
    }
  }

  const login = async (login: string, password: string) => {
    const res = await api<LoginResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: { login, password },
    })
    await fetchMe()
    return res
  }

  const register = (body: RegisterRequest) =>
    api<{ user_id: string; status: string }>('/api/v1/auth/register', { method: 'POST', body })

  const changePassword = (old_password: string, new_password: string) =>
    api('/api/v1/auth/change-password', { method: 'POST', body: { old_password, new_password } })

  const edsChallenge = () =>
    api<{ challenge: string }>('/api/v1/auth/eds/challenge', { method: 'POST' })

  const edsVerify = async (challenge: string, signed_data: string) => {
    const res = await api<LoginResponse>('/api/v1/auth/eds/verify', {
      method: 'POST',
      body: { challenge, signed_data },
    })
    await fetchMe()
    return res
  }

  const logout = async () => {
    try {
      await api('/api/v1/auth/logout', { method: 'POST' })
    } catch {
      /* ignore — всё равно чистим локально */
    }
    session.set(null)
  }

  return { fetchMe, login, register, changePassword, edsChallenge, edsVerify, logout }
}
