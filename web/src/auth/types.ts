export type AuthStatus = 'loading' | 'anonymous' | 'authenticated'

export interface AuthUser {
  id: string
  email: string
  name: string
  avatarUrl?: string
  isAdmin: boolean
}
