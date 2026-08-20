// Клиентские валидаторы (зеркалят серверную политику Auth Module).
// Возвращают текст ошибки или null, если значение валидно.

export function validatePassword(pw: string): string | null {
  if (pw.length < 8) return 'Минимум 8 символов'
  if (!/\p{Lu}/u.test(pw)) return 'Нужна хотя бы одна заглавная буква'
  if (!/\p{Ll}/u.test(pw)) return 'Нужна хотя бы одна строчная буква'
  if (!/\d/.test(pw)) return 'Нужна хотя бы одна цифра'
  return null
}

export function validateIIN(iin: string): string | null {
  if (!/^\d{12}$/.test(iin)) return 'ИИН должен содержать 12 цифр'
  return null
}

export function validateEmail(email: string): string | null {
  if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email)) return 'Некорректный email'
  return null
}

export function required(value: string, label = 'Поле'): string | null {
  if (!value || value.trim() === '') return `${label} обязательно`
  return null
}
