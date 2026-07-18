import { describe, expect, it } from 'vitest'
import { buildPassportPayload } from './buildPassportPayload'

describe('buildPassportPayload', () => {
  it('does not send browser File objects as unknown API fields', () => {
    const files = [new File(['pdf'], 'dang-ky-doanh-nghiep.pdf', { type: 'application/pdf' })]

    const payload = buildPassportPayload({
      company_name: 'P2B',
      website: 'https://p2b.vn',
      support_needs: ['Vốn ưu đãi'],
      files,
    })

    expect(payload).toEqual({
      company_name: 'P2B',
      website: 'https://p2b.vn',
      support_needs: ['Vốn ưu đãi'],
      source_names: ['dang-ky-doanh-nghiep.pdf'],
    })
    expect(payload).not.toHaveProperty('files')
  })
})
