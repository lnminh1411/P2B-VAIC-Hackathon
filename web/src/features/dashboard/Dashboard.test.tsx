import { render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { Dashboard } from './Dashboard'
import type { Application, MatchResult, MatchRun, Passport } from '../../lib/types'

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

  it('summarizes the latest matching run, verified deadline, and selected application', () => {
    const passport: Passport = {
      id: 'passport-2', company_name: 'Công ty SSI', support_needs: [], version: 3, fields: {}, updated_at: '2026-07-19T00:00:00Z',
    }
    const selectedPolicy = {
      policy_id: 'policy-1', title: 'Nghị định 162/2024/NĐ-CP', deadline: '2026-08-31T00:00:00Z',
    } as MatchResult
    const run = {
      id: 'match-2', passport_version: 3, created_at: '2026-07-19T00:00:00Z',
      results: [selectedPolicy, { ...selectedPolicy, policy_id: 'policy-2', title: 'Chính sách khác', deadline: '2026-10-01T00:00:00Z' }],
    } as MatchRun
    const application = { id: 'application-1', status: 'DRAFT_READY' } as Application

    render(<Dashboard passport={passport} matchRun={run} selectedPolicy={selectedPolicy} application={application} onNavigate={vi.fn()} />)

    const opportunities = screen.getByText('Cơ hội đang theo dõi').parentElement!
    expect(within(opportunities).getByText('2')).toBeInTheDocument()
    expect(within(opportunities).getByText('Kết quả matching gần nhất')).toBeInTheDocument()

    const deadline = screen.getByText('Deadline gần nhất').parentElement!
    expect(within(deadline).getByText('31/08/2026')).toBeInTheDocument()

    const applicationMetric = screen.getByText('Hồ sơ đang chuẩn bị').parentElement!
    expect(within(applicationMetric).getByText('1')).toBeInTheDocument()
    expect(within(applicationMetric).getByText('Nghị định 162/2024/NĐ-CP')).toBeInTheDocument()
  })
})
