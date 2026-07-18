import { createContext, useContext } from 'react'
import type { AuthStatus, AuthUser } from './types'

export interface AuthContextValue {
  status: AuthStatus
  user?: AuthUser
  signOut: () => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | undefined>(undefined)

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext)
  if (!value) throw new Error('useAuth must be used inside AuthProvider')
  return value
}
