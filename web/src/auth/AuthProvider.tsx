import type { Session } from '@supabase/supabase-js'
import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { setApiAccessToken } from '../lib/api'
import { AuthBoundary } from './AuthBoundary'
import { devAuthEnabled, supabaseConfigured } from './config'
import { AuthContext } from './context'
import { getSupabase } from './supabase'
import type { AuthStatus, AuthUser } from './types'
import { mapAuthUser } from './user'

const devUser: AuthUser = {
  id: 'dev-user',
  email: 'founder@p2b.local',
  name: 'P2B Founder',
	isAdmin: true,
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>(devAuthEnabled ? 'authenticated' : supabaseConfigured ? 'loading' : 'anonymous')
  const [user, setUser] = useState<AuthUser | undefined>(devAuthEnabled ? devUser : undefined)
  const [error, setError] = useState<string>()

  const applySession = useCallback((session: Session | null) => {
    setApiAccessToken(session?.access_token)
    setUser(session?.user ? mapAuthUser(session.user) : undefined)
    setStatus(session?.user ? 'authenticated' : 'anonymous')
  }, [])

  useEffect(() => {
    if (devAuthEnabled) return
    const supabase = getSupabase()
    if (!supabase) return

    let active = true
    void supabase.auth.getSession().then(({ data, error: sessionError }) => {
      if (!active) return
      if (sessionError) setError('Không thể khôi phục phiên đăng nhập. Vui lòng thử lại.')
      applySession(data.session)
    })
    const { data: listener } = supabase.auth.onAuthStateChange((_event, session) => {
      if (active) applySession(session)
    })
    return () => {
      active = false
      listener.subscription.unsubscribe()
    }
  }, [applySession])

  const googleSignIn = useCallback(async () => {
    const supabase = getSupabase()
    if (!supabase) return
    setError(undefined)
    const { error: signInError } = await supabase.auth.signInWithOAuth({
      provider: 'google',
      options: {
        redirectTo: `${window.location.origin}/auth/callback`,
        queryParams: { prompt: 'select_account' },
      },
    })
    if (signInError) setError('Google Login chưa thể khởi động. Vui lòng thử lại.')
  }, [])

  const signOut = useCallback(async () => {
    if (devAuthEnabled) return
    const supabase = getSupabase()
    if (!supabase) return
    await supabase.auth.signOut()
    setApiAccessToken(undefined)
  }, [])

  const value = useMemo(() => ({ status, user, signOut }), [signOut, status, user])
  return (
    <AuthContext.Provider value={value}>
      <AuthBoundary status={status} user={user} configured={devAuthEnabled || supabaseConfigured} error={error} onGoogleSignIn={() => void googleSignIn()}>
        {children}
      </AuthBoundary>
    </AuthContext.Provider>
  )
}
