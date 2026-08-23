// Форматирование значений ячеек по display_type и извлечение имени файла экспорта.

export function formatCell(value: unknown, displayType: string): string {
  if (value === null || value === undefined || value === '') return '—'

  switch (displayType) {
    case 'boolean':
      return value ? 'Да' : 'Нет'
    case 'date': {
      const d = new Date(String(value))
      return isNaN(d.getTime()) ? String(value) : new Intl.DateTimeFormat('ru-RU').format(d)
    }
    case 'datetime': {
      const d = new Date(String(value))
      return isNaN(d.getTime())
        ? String(value)
        : new Intl.DateTimeFormat('ru-RU', { dateStyle: 'short', timeStyle: 'short' }).format(d)
    }
    default:
      return String(value)
  }
}

// Извлекает имя файла из заголовка Content-Disposition (filename* и filename).
export function filenameFromDisposition(header: string | null | undefined, fallback = 'export.xlsx'): string {
  if (!header) return fallback
  const star = /filename\*=(?:UTF-8'')?([^;]+)/i.exec(header)
  if (star) {
    try {
      return decodeURIComponent(star[1].replace(/"/g, '').trim())
    } catch {
      /* падение декодирования — пробуем обычный filename */
    }
  }
  const m = /filename="?([^";]+)"?/i.exec(header)
  return m ? m[1].trim() : fallback
}
