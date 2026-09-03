import { describe, it, expect } from 'vitest'
import {
  defaultPortFor,
  sourceStatusMeta,
  validateSourceForm,
  sourcePayload,
  emptySourceForm,
  type SourceForm,
} from './sourceForm'

function form(over: Partial<SourceForm> = {}): SourceForm {
  return {
    code: 'ehd_bo',
    name: 'ЕХД БО',
    host: '10.58.56.50',
    port: 8123,
    protocol: 'http',
    tls_enabled: false,
    tls_skip_verify: false,
    username: 'default',
    password: 'secret',
    ...over,
  }
}

describe('defaultPortFor', () => {
  it('http → 8123, native → 9000', () => {
    expect(defaultPortFor('http')).toBe(8123)
    expect(defaultPortFor('native')).toBe(9000)
    expect(defaultPortFor('')).toBe(8123)
  })
})

describe('sourceStatusMeta', () => {
  it('маппит известные статусы', () => {
    expect(sourceStatusMeta('active')).toEqual({ label: 'Активен', severity: 'success' })
    expect(sourceStatusMeta('inactive').severity).toBe('secondary')
    expect(sourceStatusMeta('weird').label).toBe('weird')
  })
})

describe('validateSourceForm', () => {
  it('валидная форма → пусто', () => {
    expect(validateSourceForm(form(), false)).toBe('')
    expect(validateSourceForm(form(), true)).toBe('')
  })
  it('требует обязательные поля', () => {
    expect(validateSourceForm(form({ code: ' ' }), false)).not.toBe('')
    expect(validateSourceForm(form({ host: '' }), false)).not.toBe('')
    expect(validateSourceForm(form({ username: '' }), false)).not.toBe('')
  })
  it('проверяет диапазон порта', () => {
    expect(validateSourceForm(form({ port: 0 }), false)).not.toBe('')
    expect(validateSourceForm(form({ port: 70000 }), false)).not.toBe('')
  })
  it('отклоняет неизвестный протокол', () => {
    expect(validateSourceForm(form({ protocol: 'grpc' }), false)).not.toBe('')
  })
  it('пароль обязателен только при создании', () => {
    expect(validateSourceForm(form({ password: '' }), false)).not.toBe('')
    expect(validateSourceForm(form({ password: '' }), true)).toBe('')
  })
})

describe('sourcePayload', () => {
  it('тримит поля и включает пароль когда задан', () => {
    const p = sourcePayload(form({ code: ' ehd_bo ', password: 'pw' }))
    expect(p.code).toBe('ehd_bo')
    expect(p.password).toBe('pw')
    expect(p.port).toBe(8123)
  })
  it('пустой пароль не отправляется (undefined)', () => {
    const p = sourcePayload(form({ password: '' }))
    expect(p.password).toBeUndefined()
  })
})

describe('emptySourceForm', () => {
  it('дефолты: http/8123/default', () => {
    const f = emptySourceForm()
    expect(f.protocol).toBe('http')
    expect(f.port).toBe(8123)
    expect(f.username).toBe('default')
    expect(f.password).toBe('')
  })
})
