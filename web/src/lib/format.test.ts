import { describe, expect, it } from 'vitest'
import { formatMoney, statusLabel } from './format'

describe('statusLabel', () => {
  it('keeps missing information distinct from not met', () => {
    expect(statusLabel('MISSING_INFO')).toBe('Thiếu thông tin')
    expect(statusLabel('NOT_MET')).toBe('Chưa đáp ứng')
  })
})

describe('formatMoney', () => {
  it('formats numeric company facts in Vietnamese currency', () => {
    expect(formatMoney(5_000_000_000)).toContain('5')
    expect(formatMoney(5_000_000_000)).toContain('₫')
  })
})

