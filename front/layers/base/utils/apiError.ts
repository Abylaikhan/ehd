import type { ApiError } from '~~/shared/api/types'

// Извлекает сообщение из единого error-контракта ($fetch кладёт тело в err.data).
export function apiErrorMessage(e: unknown): string {
  const data = (e as { data?: ApiError })?.data
  return data?.error?.message || (e as { message?: string })?.message || 'Произошла ошибка'
}

export function apiErrorCode(e: unknown): string | undefined {
  return (e as { data?: ApiError })?.data?.error?.code
}
