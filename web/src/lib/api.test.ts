import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, setApiUnauthorizedHandler } from './api'

describe('api authentication failures', () => {
  afterEach(() => {
    setApiUnauthorizedHandler(undefined)
    vi.unstubAllGlobals()
  })

  it('notifies auth when the backend rejects a stale bearer token', async () => {
    const onUnauthorized = vi.fn()
    setApiUnauthorizedHandler(onUnauthorized)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'UNAUTHENTICATED', message: 'Valid Supabase bearer token required' },
    }), { status: 401, headers: { 'Content-Type': 'application/json' } })))

    await expect(api.workspaces()).rejects.toThrow('Valid Supabase bearer token required')
    expect(onUnauthorized).toHaveBeenCalledOnce()
  })
})
