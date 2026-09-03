import type { DataSourcePayload } from '~~/shared/api/types'

// Протоколы подключения к ClickHouse (совпадают с backend domain.Protocol*).
export const PROTOCOLS = [
  { label: 'HTTP (8123)', value: 'http' },
  { label: 'Native (9000)', value: 'native' },
] as const

// Поля формы источника (пароль отдельно: на вход, в ответах не приходит).
export interface SourceForm {
  code: string
  name: string
  host: string
  port: number
  protocol: string
  tls_enabled: boolean
  tls_skip_verify: boolean
  username: string
  password: string
}

export function emptySourceForm(): SourceForm {
  return {
    code: '',
    name: '',
    host: '',
    port: 8123,
    protocol: 'http',
    tls_enabled: false,
    tls_skip_verify: false,
    username: 'default',
    password: '',
  }
}

// Порт по умолчанию для протокола (подставляется при смене протокола).
export function defaultPortFor(protocol: string): number {
  return protocol === 'native' ? 9000 : 8123
}

// Метка и severity статуса источника для Tag.
export function sourceStatusMeta(status: string): { label: string; severity: string } {
  switch (status) {
    case 'active':
      return { label: 'Активен', severity: 'success' }
    case 'inactive':
      return { label: 'Неактивен', severity: 'secondary' }
    default:
      return { label: status, severity: 'secondary' }
  }
}

// Валидация формы. При редактировании пароль необязателен (пустой = не менять).
// Возвращает текст ошибки или '' если форма корректна.
export function validateSourceForm(form: SourceForm, isEdit: boolean): string {
  if (!form.code.trim() || !form.name.trim() || !form.host.trim() || !form.username.trim()) {
    return 'Заполните код, название, хост и пользователя'
  }
  if (!Number.isInteger(form.port) || form.port < 1 || form.port > 65535) {
    return 'Порт должен быть в диапазоне 1–65535'
  }
  if (form.protocol !== 'http' && form.protocol !== 'native') {
    return 'Недопустимый протокол'
  }
  if (!isEdit && !form.password) {
    return 'Пароль обязателен при создании источника'
  }
  return ''
}

// Сборка payload. Пустой пароль не отправляем (на обновлении секрет сохраняется).
export function sourcePayload(form: SourceForm): DataSourcePayload {
  return {
    code: form.code.trim(),
    name: form.name.trim(),
    host: form.host.trim(),
    port: form.port,
    protocol: form.protocol,
    tls_enabled: form.tls_enabled,
    tls_skip_verify: form.tls_skip_verify,
    username: form.username.trim(),
    password: form.password || undefined,
  }
}
