export type JobStatus = {
  id: string
  status: string
  progress: number
  last_error?: string
}

type WaitForJobOptions = {
  load: (jobId: string) => Promise<JobStatus>
  failureMessage: string
  timeoutMessage: string
  wait?: () => Promise<void>
  maxAttempts?: number
}

const waitForNextPoll = () => new Promise<void>(resolve => window.setTimeout(resolve, 1500))

export async function waitForJob(initialJob: JobStatus, options: WaitForJobOptions): Promise<JobStatus> {
  if (initialJob.status === 'SUCCEEDED') return initialJob

  const wait = options.wait ?? waitForNextPoll
  const maxAttempts = options.maxAttempts ?? 480
  for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
    await wait()
    const currentJob = await options.load(initialJob.id)
    if (currentJob.status === 'SUCCEEDED') return currentJob
    if (currentJob.status === 'FAILED' || currentJob.status === 'DEAD_LETTER') {
      throw new Error(currentJob.last_error || options.failureMessage)
    }
  }
  throw new Error(options.timeoutMessage)
}
