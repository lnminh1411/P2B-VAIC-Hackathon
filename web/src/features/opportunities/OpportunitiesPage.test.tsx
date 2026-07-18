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

    expect(screen.getByText('Chưa có policy đã publish')).toBeInTheDocument()
    expect(screen.getByText('NO PUBLISHED CORPUS')).toBeInTheDocument()
  })
})
