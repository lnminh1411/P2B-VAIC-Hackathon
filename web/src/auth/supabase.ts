import { createClient, type SupabaseClient } from '@supabase/supabase-js'
import { supabaseConfig, supabaseConfigured } from './config'

let client: SupabaseClient | undefined

export function getSupabase(): SupabaseClient | undefined {
  if (!supabaseConfigured) return undefined
  client ??= createClient(supabaseConfig.url, supabaseConfig.publishableKey, {
    auth: {
      autoRefreshToken: true,
      detectSessionInUrl: true,
      persistSession: true,
      // PKCE keeps access/refresh tokens out of the redirect URL (unlike the
      // implicit flow, which puts them in a visible #hash after Google sign-in).
      flowType: 'pkce',
    },
  })
  return client
}
