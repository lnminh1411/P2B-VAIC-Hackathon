import { render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { Dashboard } from './Dashboard'
import type { Passport } from '../../lib/types'

describe('Dashboard', () => {
  it('does not render invented opportunity, deadline, or application data', () => {
    const passport: Passport = {
      id: 'passport-1',
      company_name: 'Công ty Dữ Liệu Thật',
      website: 'https://dulieuthat.vn',
      support_needs: ['Vốn ưu đãi'],
      version: 2,
      fields: {},
      updated_at: '2026-07-18T00:00:00Z',
    }

    render(<Dashboard passport={passport} onNavigate={vi.fn()} />)

    expect(screen.queryByText(/GreenTech/i)).not.toBeInTheDocument()
    expect(screen.queryByText('68 ngày')).not.toBeInTheDocument()
    expect(within(screen.getByText('Cơ hội đang theo dõi').parentElement!).queryByText('03')).not.toBeInTheDocument()
    expect(screen.getByText('Chưa chạy matching')).toBeInTheDocument()
    expect(screen.getByText('Chưa có deadline')).toBeInTheDocument()
    expect(screen.getByText('Chưa có hồ sơ')).toBeInTheDocument()
  })
})
