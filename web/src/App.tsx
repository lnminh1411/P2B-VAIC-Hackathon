import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { ErrorState, LoadingState } from './components/PageState'
import { Shell, type Page } from './components/Shell'
import { AdminPage } from './features/admin/AdminPage'
import { AlertsPage } from './features/alerts/AlertsPage'
import { ApplicationPage } from './features/application/ApplicationPage'
import { Dashboard } from './features/dashboard/Dashboard'
import { Onboarding } from './features/onboarding/Onboarding'
import { buildPassportPayload, type CompanyOnboardingInput } from './features/onboarding/buildPassportPayload'
import { OpportunitiesPage } from './features/opportunities/OpportunitiesPage'
import { PassportPage } from './features/passport/PassportPage'
import { api, ApiError, getApiWorkspaceId, setApiWorkspaceId } from './lib/api'
import { waitForJob } from './lib/jobPolling'
import type { Application, Checklist, EnrichmentRun, MatchResult } from './lib/types'
import { useTranslation } from './lib/i18n'

export default function App() {
  const queryClient = useQueryClient()
  const { t } = useTranslation()
  const pageStateLabels = t('page_state')
  const shellLabels = t('shell')

  const [page, setPage] = useState<Page>('overview')
  const [selectedPolicy, setSelectedPolicy] = useState<MatchResult>()
  const [enrichment, setEnrichment] = useState<EnrichmentRun>()
  const [checklist, setChecklist] = useState<Checklist>()
  const [application, setApplication] = useState<Application>()
  const [workflowError, setWorkflowError] = useState<string>()
  const [activeWorkspaceId, setActiveWorkspaceId] = useState<string | undefined>(getApiWorkspaceId())

  const workspacesQuery = useQuery({ queryKey: ['workspaces'], queryFn: api.workspaces, retry: 1 })
  const selectedFromList = activeWorkspaceId && workspacesQuery.data?.workspaces.some(workspace => workspace.id === activeWorkspaceId) ? activeWorkspaceId : undefined
  const workspaceScope = selectedFromList ?? workspacesQuery.data?.active_workspace_id
  const passportQuery = useQuery({ queryKey: ['passport', workspaceScope], queryFn: api.passport, retry: 1, enabled: Boolean(workspaceScope) })
  const candidatesQuery = useQuery({ queryKey: ['candidates', workspaceScope], queryFn: api.candidates, enabled: Boolean(workspaceScope && passportQuery.data?.company_name) })
  const alertsQuery = useQuery({ queryKey: ['alerts', workspaceScope], queryFn: api.alerts, enabled: Boolean(workspaceScope && passportQuery.data?.company_name) })
  const adminQuery = useQuery({ queryKey: ['admin-policies'], queryFn: api.adminPolicies, enabled: page === 'admin' })
  const workspaces = workspacesQuery.data?.workspaces ?? []
  const selectedWorkspaceId = workspaceScope ?? workspaces[0]?.id
  const matchQuery = useQuery({ queryKey: ['match', workspaceScope], queryFn: api.getMatch, enabled: Boolean(workspaceScope && passportQuery.data?.company_name) })
  const matchRun = matchQuery.data?.results && matchQuery.data.results.length > 0 ? matchQuery.data : undefined
  const templatesQuery = useQuery({ queryKey: ['application-templates', workspaceScope], queryFn: api.applicationTemplates, enabled: Boolean(workspaceScope && passportQuery.data?.company_name) })
  const latestApplicationQuery = useQuery({ queryKey: ['application-draft', workspaceScope], queryFn: api.latestApplication, enabled: Boolean(workspaceScope && passportQuery.data?.company_name) })

  const latestCachedApplication = latestApplicationQuery.data?.application ?? undefined
  const matchingCachedApplication = selectedPolicy && latestCachedApplication?.policy_id !== selectedPolicy.policy_id ? undefined : latestCachedApplication
  const activeApplication = application ?? matchingCachedApplication
  const activeSelectedPolicy = selectedPolicy ?? matchRun?.results.find(result => result.policy_id === activeApplication?.policy_id)

  const buildMutation = useMutation({
    mutationFn: async (input: CompanyOnboardingInput) => {
      const uploads = await Promise.all(input.files.map(file => api.uploadPDF(file)))
      const job = await api.buildPassport(buildPassportPayload(input, uploads.map(upload => upload.source_id)))
      return waitForJob(job, { load: api.job, failureMessage: pageStateLabels.extract_fail, timeoutMessage: pageStateLabels.timeout })
    },
    onSuccess: async () => { await Promise.all([queryClient.invalidateQueries({ queryKey: ['passport', workspaceScope] }), queryClient.invalidateQueries({ queryKey: ['candidates', workspaceScope] }), queryClient.invalidateQueries({ queryKey: ['match', workspaceScope] })]); setPage('passport') },
  })
  const refreshMutation = useMutation({
    mutationFn: async (files: File[]) => {
      const uploads = await Promise.all(files.map(file => api.uploadPDF(file)))
      const job = await api.refreshPassport(uploads.map(upload => upload.source_id))
      return waitForJob(job, { load: api.job, failureMessage: pageStateLabels.update_fail, timeoutMessage: pageStateLabels.timeout })
    },
    onSuccess: async () => { await Promise.all([queryClient.invalidateQueries({ queryKey: ['passport', workspaceScope] }), queryClient.invalidateQueries({ queryKey: ['candidates', workspaceScope] }), queryClient.invalidateQueries({ queryKey: ['match', workspaceScope] })]) },
  })
  const createWorkspaceMutation = useMutation({ mutationFn: api.createWorkspace, onSuccess: async workspace => { await queryClient.invalidateQueries({ queryKey: ['workspaces'] }); setApiWorkspaceId(workspace.id); setActiveWorkspaceId(workspace.id); setPage('overview') } })
  const matchMutation = useMutation({ mutationFn: api.match, onSuccess: data => { queryClient.setQueryData(['match', workspaceScope], data) } })
  const enrichMutation = useMutation({ mutationFn: api.startEnrichment, onSuccess: setEnrichment })
  const checklistMutation = useMutation({ mutationFn: (policyId: string) => api.createChecklist(policyId), onSuccess: setChecklist })
  const applicationMutation = useMutation({ mutationFn: ({ checklistId, templateId }: { checklistId: string; templateId?: string }) => api.createApplication(checklistId, templateId), onSuccess: data => { setApplication(data); queryClient.setQueryData(['application-draft', workspaceScope], { application: data }) } })
  const templateUploadMutation = useMutation({ mutationFn: (file: File) => api.uploadApplicationTemplate(file), onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['application-templates', workspaceScope] }) } })
  const watchlistMutation = useMutation({ mutationFn: api.updateWatchlistSettings, onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['alerts', workspaceScope] }) } })

  if (workspacesQuery.isLoading || passportQuery.isLoading) return <LoadingState label={pageStateLabels.loading_workspace} />
  if (workspacesQuery.error) return <ErrorState message={message(workspacesQuery.error, pageStateLabels.unknown_error)} onRetry={() => workspacesQuery.refetch()} />
  if (passportQuery.error) return <ErrorState message={message(passportQuery.error, pageStateLabels.unknown_error)} onRetry={() => passportQuery.refetch()} />
  const passport = passportQuery.data
  const shellProps = { workspaces, activeWorkspaceId: selectedWorkspaceId, onWorkspaceChange: (workspaceId: string) => { setSelectedPolicy(undefined); setChecklist(undefined); setApplication(undefined); setWorkflowError(undefined); setApiWorkspaceId(workspaceId); setActiveWorkspaceId(workspaceId); setPage('overview') }, onCreateWorkspace: () => { createWorkspaceMutation.mutate(shellLabels.unnamed_workspace) } }
  if (!passport || !passport.company_name) return <Shell page="overview" {...shellProps} onNavigate={setPage}><Onboarding onSubmit={input => buildMutation.mutate(input)} busy={buildMutation.isPending} error={buildMutation.error ? message(buildMutation.error, pageStateLabels.unknown_error) : undefined} /></Shell>

  const saveField = async (fieldKey: string, value: unknown) => {
    const current = queryClient.getQueryData<typeof passport>(['passport', workspaceScope]) ?? passportQuery.data!
    const updated = await api.confirmField(fieldKey, value, current.version)
    queryClient.setQueryData(['passport', workspaceScope], updated)
    await queryClient.invalidateQueries({ queryKey: ['match', workspaceScope] })
  }
  const confirmCandidate = async (candidate: NonNullable<typeof candidatesQuery.data>['candidates'][number]) => {
    await saveField(candidate.field_key, candidate.value)
    await queryClient.invalidateQueries({ queryKey: ['candidates', workspaceScope] })
  }
  const acceptEvidence = async (candidateId: string) => {
    const current = queryClient.getQueryData<typeof passport>(['passport', workspaceScope]) ?? passport
    const updated = await api.acceptEnrichment(candidateId, current.version)
    queryClient.setQueryData(['passport', workspaceScope], updated)
    setEnrichment(currentRun => currentRun ? { ...currentRun, candidates: currentRun.candidates.map(candidate => candidate.id === candidateId ? { ...candidate, status: 'ACCEPTED' } : candidate) } : currentRun)
    await queryClient.invalidateQueries({ queryKey: ['match', workspaceScope] })
  }
  const markAvailable = async (itemId: string) => { if (!checklist) return; const updated = await api.updateChecklist(checklist.id, itemId, checklist.version, 'AVAILABLE', 'Người dùng xác nhận tài liệu trong workspace'); setChecklist(updated) }
  const applicationAction = async (action: 'submit' | 'approve' | 'generate') => {
    if (!activeApplication) return
    setWorkflowError(undefined)
    try { const updated = await api.applicationAction(activeApplication.id, action); setApplication(updated); queryClient.setQueryData(['application-draft', workspaceScope], { application: updated }) } catch (error) { setWorkflowError(message(error, pageStateLabels.unknown_error)) }
  }
  const download = async () => {
    if (!activeApplication) return
    const blob = await api.downloadApplication(activeApplication.id)
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a'); anchor.href = url; anchor.download = `P2B-${activeApplication.id}.pdf`; anchor.click(); URL.revokeObjectURL(url)
  }
  const prepare = (policy: MatchResult) => { setSelectedPolicy(policy); setChecklist(undefined); setApplication(undefined); setWorkflowError(undefined); setPage('application') }

  return <Shell page={page} {...shellProps} onNavigate={setPage} unreadAlerts={(alertsQuery.data?.alerts ?? []).filter(alert => !alert.read).length}>
    {page === 'overview' && <Dashboard passport={passport} matchRun={matchRun} selectedPolicy={activeSelectedPolicy} checklist={checklist} application={activeApplication} onNavigate={setPage} />}
    {page === 'passport' && <PassportPage passport={passport} candidates={candidatesQuery.data?.candidates ?? []} onConfirm={confirmCandidate} onSaveField={saveField} onRefresh={async files => { await refreshMutation.mutateAsync(files) }} refreshBusy={refreshMutation.isPending} busy={candidatesQuery.isFetching || refreshMutation.isPending} />}
    {page === 'opportunities' && <OpportunitiesPage run={matchRun} onMatch={() => matchMutation.mutate()} matching={matchMutation.isPending} selected={selectedPolicy} onSelect={setSelectedPolicy} onPrepare={prepare} enrichment={enrichment} onEnrich={policyId => enrichMutation.mutate(policyId)} onAcceptEvidence={candidateId => void acceptEvidence(candidateId)} busy={enrichMutation.isPending} error={matchMutation.error ? message(matchMutation.error, pageStateLabels.unknown_error) : undefined} />}
    {page === 'application' && <ApplicationPage policy={activeSelectedPolicy} checklist={checklist} application={activeApplication} templates={templatesQuery.data?.templates ?? []} onCreateChecklist={() => activeSelectedPolicy && checklistMutation.mutate(activeSelectedPolicy.policy_id)} onMarkAvailable={itemId => void markAvailable(itemId)} onCreateApplication={templateId => checklist && applicationMutation.mutate({ checklistId: checklist.id, templateId })} onUploadTemplate={async file => { await templateUploadMutation.mutateAsync(file) }} onSave={async sections => { if (!activeApplication) return; const updated = await api.updateApplication(activeApplication.id, activeApplication.version, sections); setApplication(updated); queryClient.setQueryData(['application-draft', workspaceScope], { application: updated }); return updated }} onAction={action => void applicationAction(action)} onDownload={() => void download()} busy={checklistMutation.isPending || applicationMutation.isPending || templateUploadMutation.isPending} error={workflowError || (templateUploadMutation.error ? message(templateUploadMutation.error, pageStateLabels.unknown_error) : undefined)} />}
    {page === 'alerts' && (
      <AlertsPage
        alerts={alertsQuery.data?.alerts ?? []}
        settings={alertsQuery.data?.watchlist_settings ?? { new_policies: false, deadline_changes: false, stale_evidence: false, upcoming_deadlines: false }}
        onRead={id => api.readAlert(id).then(() => queryClient.invalidateQueries({ queryKey: ['alerts', workspaceScope] }))}
        onUpdateSettings={settings => watchlistMutation.mutate(settings)}
      />
    )}
    {page === 'admin' && (adminQuery.isLoading ? <LoadingState label={shellLabels.system_loading_queue} /> : adminQuery.error ? <ErrorState message={message(adminQuery.error, pageStateLabels.unknown_error)} /> : <AdminPage policies={adminQuery.data?.policies ?? []} />)}
  </Shell>
}

function message(error: unknown, unknownLabel = 'Đã xảy ra lỗi không xác định') { return error instanceof ApiError || error instanceof Error ? error.message : unknownLabel }
