import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { ErrorState, LoadingState } from './components/PageState'
import { Shell, type Page } from './components/Shell'
import { AdminPage } from './features/admin/AdminPage'
import { AlertsPage } from './features/alerts/AlertsPage'
import { ApplicationPage } from './features/application/ApplicationPage'
import { Dashboard } from './features/dashboard/Dashboard'
import { Onboarding } from './features/onboarding/Onboarding'
import { OpportunitiesPage } from './features/opportunities/OpportunitiesPage'
import { PassportPage } from './features/passport/PassportPage'
import { api, ApiError } from './lib/api'
import type { Application, Checklist, EnrichmentRun, MatchResult, MatchRun } from './lib/types'

export default function App() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState<Page>('overview')
  const [matchRun, setMatchRun] = useState<MatchRun>()
  const [selectedPolicy, setSelectedPolicy] = useState<MatchResult>()
  const [enrichment, setEnrichment] = useState<EnrichmentRun>()
  const [checklist, setChecklist] = useState<Checklist>()
  const [application, setApplication] = useState<Application>()
  const [workflowError, setWorkflowError] = useState<string>()

  const passportQuery = useQuery({ queryKey: ['passport'], queryFn: api.passport, retry: 1 })
  const candidatesQuery = useQuery({ queryKey: ['candidates'], queryFn: api.candidates, enabled: Boolean(passportQuery.data?.company_name) })
  const alertsQuery = useQuery({ queryKey: ['alerts'], queryFn: api.alerts, enabled: Boolean(passportQuery.data?.company_name) })
  const adminQuery = useQuery({ queryKey: ['admin-policies'], queryFn: api.adminPolicies, enabled: page === 'admin' })

  const buildMutation = useMutation({
    mutationFn: async (input: { company_name: string; website: string; support_needs: string[]; files: File[] }) => {
      await Promise.all(input.files.map(file => api.uploadPDF(file)))
      return api.buildPassport({ ...input, source_names: input.files.map(file => file.name) })
    },
    onSuccess: async () => { await Promise.all([queryClient.invalidateQueries({ queryKey: ['passport'] }), queryClient.invalidateQueries({ queryKey: ['candidates'] })]); setPage('passport') },
  })
  const matchMutation = useMutation({ mutationFn: api.match, onSuccess: data => setMatchRun(data) })
  const enrichMutation = useMutation({ mutationFn: api.startEnrichment, onSuccess: setEnrichment })
  const checklistMutation = useMutation({ mutationFn: (policyId: string) => api.createChecklist(policyId), onSuccess: setChecklist })
  const applicationMutation = useMutation({ mutationFn: (checklistId: string) => api.createApplication(checklistId), onSuccess: setApplication })

  if (passportQuery.isLoading) return <LoadingState label="Đang mở workspace…" />
  if (passportQuery.error) return <ErrorState message={message(passportQuery.error)} onRetry={() => passportQuery.refetch()} />
  const passport = passportQuery.data!
  if (!passport.company_name) return <Onboarding onSubmit={input => buildMutation.mutate(input)} busy={buildMutation.isPending} error={buildMutation.error ? message(buildMutation.error) : undefined} />

  const confirmCandidate = async (candidate: NonNullable<typeof candidatesQuery.data>['candidates'][number]) => {
    const current = queryClient.getQueryData<typeof passport>(['passport']) ?? passportQuery.data!
    const updated = await api.confirmField(candidate.field_key, candidate.value, current.version)
    queryClient.setQueryData(['passport'], updated)
    await queryClient.invalidateQueries({ queryKey: ['candidates'] })
  }
  const acceptEvidence = async (candidateId: string) => {
    const current = queryClient.getQueryData<typeof passport>(['passport']) ?? passport
    const updated = await api.acceptEnrichment(candidateId, current.version)
    queryClient.setQueryData(['passport'], updated)
    setEnrichment(currentRun => currentRun ? { ...currentRun, candidates: currentRun.candidates.map(candidate => candidate.id === candidateId ? { ...candidate, status: 'ACCEPTED' } : candidate) } : currentRun)
  }
  const markAvailable = async (itemId: string) => { if (!checklist) return; const updated = await api.updateChecklist(checklist.id, itemId, checklist.version, 'AVAILABLE', 'Người dùng xác nhận tài liệu trong workspace'); setChecklist(updated) }
  const applicationAction = async (action: 'submit' | 'approve' | 'generate') => {
    if (!application) return
    setWorkflowError(undefined)
    try { setApplication(await api.applicationAction(application.id, action)) } catch (error) { setWorkflowError(message(error)) }
  }
  const download = async () => {
    if (!application) return
    const blob = await api.downloadApplication(application.id)
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a'); anchor.href = url; anchor.download = `P2B-${application.id}.pdf`; anchor.click(); URL.revokeObjectURL(url)
  }
  const prepare = (policy: MatchResult) => { setSelectedPolicy(policy); setChecklist(undefined); setApplication(undefined); setPage('application') }

  return <Shell page={page} companyName={passport.company_name} onNavigate={setPage} unreadAlerts={(alertsQuery.data?.alerts ?? []).filter(alert => !alert.read).length}>
    {page === 'overview' && <Dashboard passport={passport} onNavigate={setPage} />}
    {page === 'passport' && <PassportPage passport={passport} candidates={candidatesQuery.data?.candidates ?? []} onConfirm={confirmCandidate} busy={candidatesQuery.isFetching} />}
    {page === 'opportunities' && <OpportunitiesPage run={matchRun} onMatch={() => matchMutation.mutate()} matching={matchMutation.isPending} selected={selectedPolicy} onSelect={setSelectedPolicy} onPrepare={prepare} enrichment={enrichment} onEnrich={policyId => enrichMutation.mutate(policyId)} onAcceptEvidence={candidateId => void acceptEvidence(candidateId)} busy={enrichMutation.isPending} />}
    {page === 'application' && <ApplicationPage policy={selectedPolicy} checklist={checklist} application={application} onCreateChecklist={() => selectedPolicy && checklistMutation.mutate(selectedPolicy.policy_id)} onMarkAvailable={itemId => void markAvailable(itemId)} onCreateApplication={() => checklist && applicationMutation.mutate(checklist.id)} onSave={async sections => { if (application) setApplication(await api.updateApplication(application.id, application.version, sections)) }} onAction={action => void applicationAction(action)} onDownload={() => void download()} busy={checklistMutation.isPending || applicationMutation.isPending} error={workflowError} />}
    {page === 'alerts' && <AlertsPage alerts={alertsQuery.data?.alerts ?? []} onRead={id => api.readAlert(id).then(() => queryClient.invalidateQueries({ queryKey: ['alerts'] }))} />}
    {page === 'admin' && (adminQuery.isLoading ? <LoadingState label="Đang tải review queue…" /> : adminQuery.error ? <ErrorState message={message(adminQuery.error)} /> : <AdminPage policies={adminQuery.data?.policies ?? []} />)}
  </Shell>
}

function message(error: unknown) { return error instanceof ApiError || error instanceof Error ? error.message : 'Đã xảy ra lỗi không xác định' }
