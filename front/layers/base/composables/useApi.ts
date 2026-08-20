// Типизированный клиент API ЕХД.
// Браузер: запросы к /api/... идут через nitro-прокси на ehd-api (cookie сессии — HttpOnly, шлёт браузер).
// SSR: запросы идут напрямую в ehd-api (config.apiBase), поэтому входящий Cookie форвардится вручную.
export function useApi() {
  const config = useRuntimeConfig()
  const headers = import.meta.server ? useRequestHeaders(['cookie']) : undefined

  return $fetch.create({
    baseURL: import.meta.server ? config.apiBase : '',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      ...(headers || {}),
    },
  })
}
