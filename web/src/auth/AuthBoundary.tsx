import type { ReactNode } from 'react'
import { LoginPage } from './LoginPage'
import type { AuthStatus, AuthUser } from './types'

interface AuthBoundaryProps {
  status: AuthStatus
  configured: boolean
  user?: AuthUser
  error?: string
  onGoogleSignIn: () => void
  children: ReactNode
}

export function AuthBoundary({ status, configured, user, error, onGoogleSignIn, children }: AuthBoundaryProps) {
  if (status === 'loading') {
    return <div className="auth-loading" role="status" aria-live="polite"><span /><strong>Đang xác minh phiên đăng nhập</strong><small>P2B đang mở workspace bảo mật của bạn…</small></div>
  }
  if (status === 'authenticated' && user) return children
  return <LoginPage configured={configured} error={error} onGoogleSignIn={onGoogleSignIn} />
}
