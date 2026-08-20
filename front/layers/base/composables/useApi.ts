// Типизированный клиент API ЕХД.
// На сервере (SSR) запросы идут напрямую в ehd-api, в браузере — через nitro-прокси /api.
// Позже сюда подключается клиент, сгенерированный из OpenAPI (shared/api).
export function useApi() {
  const config = useRuntimeConfig()

  return $fetch.create({
    baseURL: import.meta.server ? config.apiBase : '',
    headers: { Accept: 'application/json' },
  })
}
