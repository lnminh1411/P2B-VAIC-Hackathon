import { act, fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ApplicationPage } from './ApplicationPage'
import type { Application, Checklist, MatchResult } from '../../lib/types'

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
	it('allows selecting one cached template before creating a draft', () => {
		const onCreateApplication = vi.fn()
		render(<ApplicationPage
			policy={policy}
			checklist={{ id: 'checklist-1', policy_id: 'policy-1', policy_version: 1, version: 1, items: [], updated_at: new Date().toISOString() }}
			templates={[{ id: 'template-1', name: 'Hồ sơ vay vốn 2025', filename: 'mau.docx', content_type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', placeholders: ['company_name'], created_at: '', updated_at: '' }]}
			onCreateChecklist={vi.fn()}
			onMarkAvailable={vi.fn()}
			onCreateApplication={onCreateApplication}
			onUploadTemplate={vi.fn()}
			onSave={vi.fn()}
			onAction={vi.fn()}
			onDownload={vi.fn()}
			busy={false}
		/>)

		fireEvent.click(screen.getByRole('radio', { name: /Hồ sơ vay vốn 2025/ }))
		fireEvent.click(screen.getByRole('button', { name: 'Tạo bản nháp' }))
		expect(onCreateApplication).toHaveBeenCalledWith('template-1')
	})

	it('automatically saves a changed draft after the debounce', async () => {
		vi.useFakeTimers()
		const onSave = vi.fn().mockResolvedValue({ ...application, version: 2 })
		render(<ApplicationPage
			policy={policy}
			application={application}
			onCreateChecklist={vi.fn()}
			onMarkAvailable={vi.fn()}
			onCreateApplication={vi.fn()}
			onUploadTemplate={vi.fn()}
			onSave={onSave}
			onAction={vi.fn()}
			onDownload={vi.fn()}
			busy={false}
		/>)

		fireEvent.change(screen.getByRole('textbox', { name: 'Tổng quan doanh nghiệp' }), { target: { value: 'Nội dung mới' } })
		await act(async () => { await vi.advanceTimersByTimeAsync(900) })
		expect(onSave).toHaveBeenCalledWith({ company_overview: 'Nội dung mới' })
		expect(screen.getByRole('status')).toHaveTextContent('Đã tự động lưu')
		vi.useRealTimers()
	})

	it('offers a transparent P2B working template for a retrieved legal document', () => {
	  render(<ApplicationPage
		policy={{ ...policy, retrieval_mode: 'HYBRID_RULE_VECTOR', eligibility: { status: 'MISSING_INFO', criteria: [{ rule_id: 'document-review-1' }] } } as MatchResult}
		onCreateChecklist={vi.fn()}
		onMarkAvailable={vi.fn()}
		onCreateApplication={vi.fn()}
		onSave={vi.fn()}
		onAction={vi.fn()}
		onDownload={vi.fn()}
		busy={false}
	  />)

	  expect(screen.getByRole('heading', { name: 'Tạo checklist từ văn bản' })).toBeInTheDocument()
	  expect(screen.getByText(/mẫu làm việc P2B/)).toBeInTheDocument()
	  expect(screen.getByRole('button', { name: /Tạo checklist/ })).toBeEnabled()
	})

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

  it('renders checklist items from legacy payloads whose field keys are null', () => {
    const checklist = {
      id: 'checklist-legacy',
      policy_id: 'policy-1',
      policy_version: 1,
      version: 1,
      items: [{ id: 'item-1', title: 'Đối chiếu văn bản', description: '', required: true, status: 'MISSING', field_keys: null }],
      updated_at: new Date().toISOString(),
    } as unknown as Checklist

    render(<ApplicationPage
      policy={policy}
      checklist={checklist}
      onCreateChecklist={vi.fn()}
      onMarkAvailable={vi.fn()}
      onCreateApplication={vi.fn()}
      onSave={vi.fn()}
      onAction={vi.fn()}
      onDownload={vi.fn()}
      busy={false}
    />)

    expect(screen.getByText('Đối chiếu văn bản')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Xác nhận đã có' })).toBeEnabled()
  })
})
