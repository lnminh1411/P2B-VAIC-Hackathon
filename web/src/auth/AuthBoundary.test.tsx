import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AuthBoundary } from './AuthBoundary'
import type { AuthUser } from './types'

const user: AuthUser = {
  id: 'user-1',
  email: 'founder@greentech.vn',
  name: 'Nguyễn Minh Anh',
  avatarUrl: 'https://example.com/avatar.png',
	isAdmin: false,
}

describe('AuthBoundary', () => {
  it('requires Google sign-in before mounting company features', () => {
    const onGoogleSignIn = vi.fn()
    render(
      <AuthBoundary status="anonymous" configured onGoogleSignIn={onGoogleSignIn}>
        <div>Company Passport workspace</div>
      </AuthBoundary>,
    )

    expect(screen.queryByText('Company Passport workspace')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Tiếp tục với Google' }))
    expect(onGoogleSignIn).toHaveBeenCalledOnce()
  })

  it('renders the workspace only after authentication', () => {
    render(
      <AuthBoundary status="authenticated" configured user={user} onGoogleSignIn={vi.fn()}>
        <div>Company Passport workspace</div>
      </AuthBoundary>,
    )

    expect(screen.getByText('Company Passport workspace')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Tiếp tục với Google' })).not.toBeInTheDocument()
  })

  it('fails closed when Supabase public configuration is missing', () => {
    render(
      <AuthBoundary status="anonymous" configured={false} onGoogleSignIn={vi.fn()}>
        <div>Company Passport workspace</div>
      </AuthBoundary>,
    )

    expect(screen.getByRole('button', { name: 'Tiếp tục với Google' })).toBeDisabled()
    expect(screen.getByText('Google Login chưa được cấu hình')).toBeInTheDocument()
  })
})
