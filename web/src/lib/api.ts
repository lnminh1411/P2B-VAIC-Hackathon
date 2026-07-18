import type { Alert, Application, Candidate, Checklist, EnrichmentRun, MatchRun, Passport, Workspace } from './types'

const API_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'
const WORKSPACE = 'p2b-local-development'
const DEV_AUTH = import.meta.env.VITE_DEV_AUTH === 'true'
let accessToken: string | undefined
let activeWorkspaceId: string | undefined
let unauthorizedHandler: (() => void) | undefined

export function setApiAccessToken(token?: string) {
  accessToken = token
}

export function setApiWorkspaceId(workspaceId?: string) {
  activeWorkspaceId = workspaceId
}

export function getApiWorkspaceId() { return activeWorkspaceId }

export function setApiUnauthorizedHandler(handler?: () => void) {
  unauthorizedHandler = handler
}

class ApiError extends Error {
  constructor(message: string, public status: number, public code?: string, public details?: string[]) { super(message) }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Content-Type', 'application/json')
  if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`)
  const workspaceId = activeWorkspaceId ?? (DEV_AUTH ? WORKSPACE : undefined)
  if (workspaceId) headers.set('X-Workspace-ID', workspaceId)
  if (init.method && init.method !== 'GET') headers.set('Idempotency-Key', crypto.randomUUID())
  const response = await fetch(`${API_URL}${path}`, { ...init, headers })
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ error: { message: 'Không thể kết nối hệ thống' } }))
    if (response.status === 401) unauthorizedHandler?.()
    throw new ApiError(payload.error?.message ?? 'Yêu cầu thất bại', response.status, payload.error?.code, payload.error?.details)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export const api = {
  health: () => request<{ status: string; mode: string }>('/health/ready'),
  workspaces: () => request<{ workspaces: Workspace[]; active_workspace_id: string }>('/v1/workspaces'),
  createWorkspace: (displayName: string) => request<Workspace>('/v1/workspaces', { method: 'POST', body: JSON.stringify({ display_name: displayName }) }),
  passport: () => request<Passport>('/v1/passport'),
  candidates: () => request<{ candidates: Candidate[] }>('/v1/passport/candidates'),
  uploadPDF: async (file: File) => {
    const signed = await request<{ source_id: string; object_key?: string; upload_url: string }>('/v1/uploads/presign', {
      method: 'POST',
      body: JSON.stringify({ filename: file.name, content_type: file.type || 'application/pdf', size_bytes: file.size }),
    })
    if (!signed.upload_url.startsWith('http')) return signed
    const form = new FormData()
    form.append('cacheControl', '3600')
    form.append('', file)
    const response = await fetch(signed.upload_url, { method: 'PUT', headers: { 'x-upsert': 'false' }, body: form })
    if (!response.ok) throw new ApiError(`Không thể tải lên ${file.name}`, response.status)
	await request<void>(`/v1/uploads/${signed.source_id}/complete`, { method: 'POST' })
    return signed
  },
  buildPassport: (input: { company_name: string; website: string; support_needs: string[]; source_names: string[]; source_ids: string[] }) => request<{ id: string; status: string; progress: number }>('/v1/passports/build', { method: 'POST', body: JSON.stringify(input) }),
  refreshPassport: (sourceIds: string[]) => request<{ id: string; status: string; progress: number }>('/v1/passports/refresh', { method: 'POST', body: JSON.stringify({ source_ids: sourceIds }) }),
	job: (id: string) => request<{ id: string; status: string; progress: number; last_error?: string }>(`/v1/jobs/${id}`),
  confirmField: (fieldKey: string, value: unknown, version: number) => request<Passport>(`/v1/passport/fields/${fieldKey}`, { method: 'PUT', body: JSON.stringify({ value, expected_version: version }) }),
  match: () => request<MatchRun>('/v1/matches', { method: 'POST', body: '{}' }),
  getMatch: () => request<MatchRun>('/v1/matches'),
  startEnrichment: (policyId: string) => request<EnrichmentRun>('/v1/enrichment-runs', { method: 'POST', body: JSON.stringify({ policy_id: policyId }) }),
  acceptEnrichment: (candidateId: string, version: number) => request<Passport>(`/v1/enrichment-candidates/${candidateId}/accept`, { method: 'POST', body: JSON.stringify({ expected_version: version }) }),
  rejectEnrichment: (candidateId: string) => request<void>(`/v1/enrichment-candidates/${candidateId}/reject`, { method: 'POST' }),
  createChecklist: (policyId: string) => request<Checklist>('/v1/checklists', { method: 'POST', body: JSON.stringify({ policy_id: policyId }) }),
  updateChecklist: (checklistId: string, itemId: string, version: number, status: string, evidenceSource: string) => request<Checklist>(`/v1/checklists/${checklistId}/items/${itemId}`, { method: 'PUT', body: JSON.stringify({ status, evidence_source: evidenceSource, expected_version: version }) }),
  createApplication: (checklistId: string) => request<Application>('/v1/applications', { method: 'POST', body: JSON.stringify({ checklist_id: checklistId }) }),
  updateApplication: (applicationId: string, version: number, sections: Record<string, string>) => request<Application>(`/v1/applications/${applicationId}`, { method: 'PUT', body: JSON.stringify({ expected_version: version, sections }) }),
  applicationAction: (applicationId: string, action: 'submit' | 'approve' | 'generate') => request<Application>(`/v1/applications/${applicationId}/${action}`, { method: 'POST' }),
  downloadApplication: async (applicationId: string) => {
    const headers = new Headers()
    if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`)
    const workspaceId = activeWorkspaceId ?? (DEV_AUTH ? WORKSPACE : undefined)
    if (workspaceId) headers.set('X-Workspace-ID', workspaceId)
    const response = await fetch(`${API_URL}/v1/applications/${applicationId}/download`, { headers })
    if (!response.ok) throw new ApiError('Không thể tải PDF', response.status)
    return response.blob()
  },
  alerts: () => request<{ alerts: Alert[] }>('/v1/alerts'),
  readAlert: (id: string) => request<Alert>(`/v1/alerts/${id}/read`, { method: 'POST' }),
  adminPolicies: () => request<{ policies: Array<{ id: string; title: string; agency: string; lifecycle: string; version: number; verified_at: string; template_ready: boolean }> }>('/v1/admin/policies'),
}

export { ApiError }
