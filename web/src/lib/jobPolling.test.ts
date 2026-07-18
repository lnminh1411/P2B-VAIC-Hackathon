import { describe, expect, it, vi } from 'vitest'
import { waitForJob, type JobStatus } from './jobPolling'

const queuedJob: JobStatus = { id: 'job-1', status: 'QUEUED', progress: 0 }

describe('waitForJob', () => {
  it('returns an already completed job without polling', async () => {
    const load = vi.fn()
    const completed = { ...queuedJob, status: 'SUCCEEDED', progress: 100 }

    await expect(waitForJob(completed, { load, failureMessage: 'failed', timeoutMessage: 'timeout' })).resolves.toEqual(completed)
    expect(load).not.toHaveBeenCalled()
  })

  it('polls until the job succeeds', async () => {
    const completed = { ...queuedJob, status: 'SUCCEEDED', progress: 100 }
    const load = vi.fn().mockResolvedValueOnce({ ...queuedJob, progress: 50 }).mockResolvedValueOnce(completed)
    const wait = vi.fn().mockResolvedValue(undefined)

    await expect(waitForJob(queuedJob, { load, wait, failureMessage: 'failed', timeoutMessage: 'timeout' })).resolves.toEqual(completed)
    expect(load).toHaveBeenCalledTimes(2)
    expect(wait).toHaveBeenCalledTimes(2)
  })

  it('uses the server error when the job fails', async () => {
    const load = vi.fn().mockResolvedValue({ ...queuedJob, status: 'FAILED', last_error: 'extraction failed' })

    await expect(waitForJob(queuedJob, { load, wait: async () => undefined, failureMessage: 'fallback', timeoutMessage: 'timeout' })).rejects.toThrow('extraction failed')
  })

  it('throws after the configured polling limit', async () => {
    const load = vi.fn().mockResolvedValue(queuedJob)

    await expect(waitForJob(queuedJob, { load, wait: async () => undefined, maxAttempts: 2, failureMessage: 'failed', timeoutMessage: 'timeout' })).rejects.toThrow('timeout')
    expect(load).toHaveBeenCalledTimes(2)
  })
})
