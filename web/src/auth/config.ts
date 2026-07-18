export const devAuthEnabled = import.meta.env.VITE_DEV_AUTH === 'true'

export const supabaseConfig = {
  url: import.meta.env.VITE_SUPABASE_URL?.trim() ?? '',
  publishableKey: import.meta.env.VITE_SUPABASE_PUBLISHABLE_KEY?.trim() ?? '',
}

export const supabaseConfigured = Boolean(supabaseConfig.url && supabaseConfig.publishableKey)
