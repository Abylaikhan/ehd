import { describe, it, expect } from 'vitest'
import { validatePassword, validateIIN, validateEmail } from './validators'

describe('validatePassword', () => {
  it('accepts a compliant password', () => {
    expect(validatePassword('Passw0rd')).toBeNull()
  })
  it('rejects too short', () => {
    expect(validatePassword('Pa0ss')).not.toBeNull()
  })
  it('rejects missing upper/lower/digit', () => {
    expect(validatePassword('password')).not.toBeNull()
    expect(validatePassword('PASSWORD0')).not.toBeNull()
    expect(validatePassword('Password')).not.toBeNull()
  })
})

describe('validateIIN', () => {
  it('accepts 12 digits', () => {
    expect(validateIIN('990101300123')).toBeNull()
  })
  it('rejects non-12-digit', () => {
    expect(validateIIN('12345')).not.toBeNull()
    expect(validateIIN('99010130012a')).not.toBeNull()
  })
})

describe('validateEmail', () => {
  it('accepts a valid email', () => {
    expect(validateEmail('user@example.kz')).toBeNull()
  })
  it('rejects invalid', () => {
    expect(validateEmail('nope')).not.toBeNull()
  })
})
