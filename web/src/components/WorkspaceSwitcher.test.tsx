import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { WorkspaceSwitcher } from './WorkspaceSwitcher'

describe('WorkspaceSwitcher', () => {
  it('lets the user switch business and start creating another one', () => {
    const onChange = vi.fn()
    const onCreate = vi.fn()
    render(<WorkspaceSwitcher workspaces={[{ id: 'one', display_name: 'Công ty Một', role: 'OWNER', created_at: '' }, { id: 'two', display_name: 'Công ty Hai', role: 'MEMBER', created_at: '' }]} activeWorkspaceId="one" onChange={onChange} onCreate={onCreate} />)

    fireEvent.change(screen.getByLabelText('Doanh nghiệp đang chọn'), { target: { value: 'two' } })
    fireEvent.click(screen.getByRole('button', { name: 'Thêm doanh nghiệp' }))

    expect(onChange).toHaveBeenCalledWith('two')
    expect(onCreate).toHaveBeenCalledOnce()
  })
})
