import type { User } from '@supabase/supabase-js'
import { describe, expect, it } from 'vitest'
import { mapAuthUser } from './user'

describe('mapAuthUser', () => {
  it('uses only app_metadata for admin UI capability', () => {
    const base = { id: 'user-1', email: 'founder@p2b.vn', user_metadata: { roles: ['admin'] } } as unknown as User
    expect(mapAuthUser(base).isAdmin).toBe(false)

    const trusted = { ...base, app_metadata: { roles: ['admin'] } } as unknown as User
    expect(mapAuthUser(trusted).isAdmin).toBe(true)
  })
})
