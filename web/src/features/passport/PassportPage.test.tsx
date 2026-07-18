import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { Candidate, Passport } from '../../lib/types'
import { CandidateCard, PassportPage } from './PassportPage'

const passport: Passport = {
  id: 'passport-1',
  company_name: 'Công ty Dữ Liệu Thật',
  website: 'https://dulieuthat.vn',
  support_needs: ['Chuyển đổi số'],
  version: 2,
  fields: {
    legal_name: {
      key: 'legal_name',
      label: 'Tên pháp lý',
      value: 'Công ty Dữ Liệu Thật',
      data_type: 'string',
      status: 'CONFIRMED',
      confidence: 1,
      evidence: [],
    },
    employee_count: { key: 'employee_count', label: 'Số lao động', data_type: 'integer', status: 'MISSING', confidence: 0, evidence: [] },
    province: { key: 'province', label: 'Tỉnh/thành', data_type: 'string', status: 'MISSING', confidence: 0, evidence: [] },
    technologies: { key: 'technologies', label: 'Công nghệ', data_type: 'string_array', status: 'MISSING', confidence: 0, evidence: [] },
  },
  updated_at: '2026-07-18T00:00:00Z',
}

describe('PassportPage', () => {
  it('shows missing scale, geography, and technology fields and lets the user enter a value', async () => {
    const onSaveField = vi.fn().mockResolvedValue(undefined)

    render(<PassportPage passport={passport} candidates={[]} onConfirm={vi.fn()} onSaveField={onSaveField} busy={false} />)

    expect(screen.getByText('Số lao động')).toBeInTheDocument()
    expect(screen.getByText('Tỉnh/thành')).toBeInTheDocument()
    expect(screen.getByText('Công nghệ')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Chỉnh sửa Số lao động' }))
    fireEvent.change(screen.getByLabelText('Giá trị Số lao động'), { target: { value: '25' } })
    fireEvent.click(screen.getByRole('button', { name: 'Lưu thay đổi' }))

    await waitFor(() => expect(onSaveField).toHaveBeenCalledWith('employee_count', 25))
  })

  it('rejects a fractional employee count instead of silently rounding it', async () => {
    const onSaveField = vi.fn().mockResolvedValue(undefined)
    render(<PassportPage passport={passport} candidates={[]} onConfirm={vi.fn()} onSaveField={onSaveField} busy={false} />)

    fireEvent.click(screen.getByRole('button', { name: 'Chỉnh sửa Số lao động' }))
    fireEvent.change(screen.getByLabelText('Giá trị Số lao động'), { target: { value: '25.5' } })
    fireEvent.click(screen.getByRole('button', { name: 'Lưu thay đổi' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Giá trị phải là số nguyên')
    expect(onSaveField).not.toHaveBeenCalled()
  })

  it('does not display a page number in AI suggestions', () => {
    const candidate: Candidate = {
      id: 'candidate-1',
      field_key: 'tax_code',
      value: '0312345678',
      data_type: 'string',
      confidence: 0.98,
      evidence: {
        source_id: 'source-1',
        source_name: 'dang-ky-doanh-nghiep.pdf',
        page: 7,
        quote: 'Mã số doanh nghiệp: 0312345678',
        content_hash: 'sha256:test',
        observed_at: '2026-07-18T00:00:00Z',
      },
      status: 'NEEDS_REVIEW',
    }

    render(<CandidateCard candidate={candidate} onConfirm={vi.fn()} busy={false} />)

    expect(screen.getByText('dang-ky-doanh-nghiep.pdf')).toBeInTheDocument()
    expect(screen.queryByText(/Trang 7/)).not.toBeInTheDocument()
  })
})
