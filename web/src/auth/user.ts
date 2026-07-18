import type { User } from '@supabase/supabase-js'
import type { AuthUser } from './types'

export function mapAuthUser(user: User): AuthUser {
  const metadata = user.user_metadata ?? {}
	const trustedRoles = Array.isArray(user.app_metadata?.roles) ? user.app_metadata.roles : []
  return {
    id: user.id,
    email: user.email ?? '',
    name: metadata.full_name ?? metadata.name ?? user.email?.split('@')[0] ?? 'P2B User',
    avatarUrl: metadata.avatar_url ?? metadata.picture,
		isAdmin: trustedRoles.some(role => typeof role === 'string' && role.toLowerCase() === 'admin'),
  }
}
