import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ApplicationPage } from './ApplicationPage'
import type { Application, MatchResult } from '../../lib/types'

const policy = {
  title: 'Chương trình chuyển đổi xanh',
  agency: 'Sở Khoa học và Công nghệ',
} as MatchResult

const application = {
  id: 'application-1',
  status: 'DRAFT_READY',
  version: 1,
  passport_version: 8,
  sections: { company_overview: 'Tổng quan doanh nghiệp' },
  blocking_reasons: null,
} as unknown as Application

describe('ApplicationPage', () => {
  it('renders applications whose API payload has null blocking reasons', () => {
    render(<ApplicationPage
      policy={policy}
      checklist={{ id: 'checklist-1', policy_id: 'policy-1', policy_version: 1, version: 1, items: [], updated_at: new Date().toISOString() }}
      application={application}
      onCreateChecklist={vi.fn()}
      onMarkAvailable={vi.fn()}
      onCreateApplication={vi.fn()}
      onSave={vi.fn()}
      onAction={vi.fn()}
      onDownload={vi.fn()}
      busy={false}
    />)

    expect(screen.getByRole('heading', { name: 'Nội dung hồ sơ' })).toBeInTheDocument()
    expect(screen.getByText('Passport').parentElement).toHaveTextContent('v8')
  })
})
