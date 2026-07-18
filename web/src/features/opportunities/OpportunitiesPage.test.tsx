import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { MatchRun } from '../../lib/types'
import { OpportunitiesPage } from './OpportunitiesPage'

const props = {
  onMatch: vi.fn(),
  matching: false,
  onSelect: vi.fn(),
  onPrepare: vi.fn(),
  onEnrich: vi.fn(),
  onAcceptEvidence: vi.fn(),
  busy: false,
}

describe('OpportunitiesPage', () => {
  it('renders an empty result instead of crashing when the API returns null results', () => {
    const run = {
      id: 'match-1',
      passport_version: 2,
      created_at: '2026-07-18T00:00:00Z',
      results: null,
    } as unknown as MatchRun

    render(<OpportunitiesPage {...props} run={run} />)

    expect(screen.getByText('Chưa tìm thấy kết quả phù hợp')).toBeInTheDocument()
  })

  it('renders a retrieved legal document with a safe source link and no fake deadline', () => {
	const run: MatchRun = {
	  id: 'match-2', passport_version: 3, created_at: '2026-07-19T00:00:00Z',
	  results: [{
		policy_id: 'document-1', policy_version: 1, title: 'Nghị định hỗ trợ doanh nghiệp', agency: 'Bộ KH&CN',
		benefit: 'Điều khoản hỗ trợ đổi mới công nghệ', benefit_amount: '', deadline: '0001-01-01T00:00:00Z', score: 82,
		eligibility: { status: 'MISSING_INFO', criteria: [] }, ranking_reasons: ['Độ tương đồng ngữ nghĩa 91%'],
		template_ready: false, retrieval_mode: 'HYBRID_RULE_VECTOR', source_url: 'https://vbpl.vn/van-ban/1',
	  }],
	}

	render(<OpportunitiesPage {...props} run={run} selected={run.results[0]} />)

	expect(screen.getByText('Không thời hạn')).toBeInTheDocument()
	expect(screen.getByRole('link', { name: /Mở văn bản nguồn/ })).toHaveAttribute('href', 'https://vbpl.vn/van-ban/1')
  })
})
