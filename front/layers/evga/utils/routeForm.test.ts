import { describe, expect, it } from 'vitest'
import { routeComplete } from './routeForm'

describe('routeForm.routeComplete', () => {
  it('AT-09: неполный маршрут не проходит', () => {
    expect(routeComplete([], 5)).toBe(false) // нет согласующих
    expect(routeComplete([3], null)).toBe(false) // нет ответственного
    expect(routeComplete([3, 3], 5)).toBe(false) // дубль согласующего
    expect(routeComplete([0], 5)).toBe(false) // некорректный id
  })
  it('полный маршрут проходит', () => {
    expect(routeComplete([3], 5)).toBe(true)
    expect(routeComplete([3, 7], 5)).toBe(true)
  })
})
