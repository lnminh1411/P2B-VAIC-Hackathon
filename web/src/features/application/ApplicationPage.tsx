import { Check, Download, FileCheck2, FileText, LockKeyhole, Send, Sparkles } from 'lucide-react'
import { useState } from 'react'
import { StatusBadge } from '../../components/StatusBadge'
import { isRetrievedDocument } from '../../lib/policy'
import type { Application, Checklist, MatchResult } from '../../lib/types'
import { useTranslation } from '../../lib/i18n'

export function ApplicationPage({ policy, checklist, application, onCreateChecklist, onMarkAvailable, onCreateApplication, onSave, onAction, onDownload, busy, error }: {
  policy?: MatchResult; checklist?: Checklist; application?: Application; onCreateChecklist: () => void; onMarkAvailable: (itemId: string) => void; onCreateApplication: () => void; onSave: (sections: Record<string, string>) => void; onAction: (action: 'submit' | 'approve' | 'generate') => void; onDownload: () => void; busy: boolean; error?: string
}) {
  const { t } = useTranslation()
  const app = t('application')

  if (!policy) return <div className="application-empty"><FileCheck2 /><span className="kicker">APPLICATION PREPARATION</span><h1>{app.empty_h1}</h1><p>{app.empty_desc}</p></div>
  const retrieved = isRetrievedDocument(policy)
  return <><section className="page-heading split-heading"><div><span className="kicker">APPLICATION · {policy.agency}</span><h1>{app.h1}<em>{app.h1_em}</em></h1><p>{policy.title}</p></div>{application && <StatusBadge status={application.status} />}</section>{!checklist ? <StartCard title={retrieved ? app.document_step1_title : app.step1_title} copy={retrieved ? app.document_step1_desc : app.step1_desc} action={app.step1_btn} onClick={onCreateChecklist} busy={busy} /> : !application ? <ChecklistReview checklist={checklist} retrieved={retrieved} onMarkAvailable={onMarkAvailable} onCreate={onCreateApplication} busy={busy} /> : <ApplicationEditor application={application} onSave={onSave} onAction={onAction} onDownload={onDownload} busy={busy} error={error} />}</>
}

function StartCard({ title, copy, action, onClick, busy }: { title: string; copy: string; action: string; onClick: () => void; busy: boolean }) {
  const { t } = useTranslation()
  const app = t('application')
  return <section className="start-card panel"><div className="step-orb"><Sparkles /></div><span>{app.step_1_of_3}</span><h2>{title}</h2><p>{copy}</p><button className="button primary" onClick={onClick} disabled={busy}>{action}<Check /></button></section>
}

function ChecklistReview({ checklist, retrieved, onMarkAvailable, onCreate, busy }: { checklist: Checklist; retrieved: boolean; onMarkAvailable: (itemId: string) => void; onCreate: () => void; busy: boolean }) {
  const { t } = useTranslation()
  const app = t('application')
  const missing = checklist.items.filter(item => item.required && item.status !== 'AVAILABLE')
  return <div className="application-grid"><section className="panel checklist-panel"><div className="panel-title"><div><span>{app.step_1_of_3} · DOCUMENT CHECKLIST</span><h2>{app.checklist_subtitle}</h2></div><strong>{checklist.items.length - missing.length}/{checklist.items.length}</strong></div><div className="checklist-list">{checklist.items.map(item => <article key={item.id}><div className="doc-icon"><FileText /></div><div><strong>{item.title}</strong><p>{item.description || app.default_desc}</p><small>{item.required ? app.required : app.optional}{(item.field_keys ?? []).length > 0 ? ` · ${(item.field_keys ?? []).join(', ')}` : ''}</small></div><StatusBadge status={item.status} />{item.status !== 'AVAILABLE' && <button className="button secondary" disabled={busy} onClick={() => onMarkAvailable(item.id)}>{app.confirm_available_btn}</button>}</article>)}</div></section><aside className="panel approval-card"><LockKeyhole /><h3>Human approval gate</h3><p>{app.gate_desc}</p><ul><li><Check />{app.gate_policy_pinned}</li><li><Check />{app.gate_passport_pinned}</li><li><Check />{retrieved ? app.gate_template_working : app.gate_template_approved}</li></ul><button className="button primary wide" disabled={missing.length > 0 || busy} onClick={onCreate}>{app.draft_btn}</button>{missing.length > 0 && <small>{app.missing_items_footer.replace('{count}', String(missing.length))}</small>}</aside></div>
}

function ApplicationEditor({ application, onSave, onAction, onDownload, busy, error }: { application: Application; onSave: (sections: Record<string, string>) => void; onAction: (action: 'submit' | 'approve' | 'generate') => void; onDownload: () => void; busy: boolean; error?: string }) {
  const { t } = useTranslation()
  const app = t('application')
  const [sections, setSections] = useState(application.sections)
  const blockingReasons = application.blocking_reasons ?? []
  const update = (key: string, value: string) => setSections(current => ({ ...current, [key]: value }))
  const nextAction = application.status === 'DRAFT_READY' ? 'submit' : application.status === 'PENDING_REVIEW' ? 'approve' : application.status === 'APPROVED' ? 'generate' : undefined
  const nextLabel = nextAction === 'submit' ? app.submit_btn : nextAction === 'approve' ? app.approve_btn : app.generate_btn

  const sectionLabel = (key: string) => {
    return ({
      company_overview: app.label_overview,
      support_need: app.label_need,
      proposal: app.label_proposal
    } as Record<string, string>)[key] ?? key
  }

  return <div className="application-grid"><section className="panel editor-panel"><div className="panel-title"><div><span>{app.step_2_of_3} · HUMAN REVIEW</span><h2>{app.editor_subtitle}</h2></div><StatusBadge status={application.status} /></div>{Object.entries(sections).map(([key, value]) => <label key={key}><span>{sectionLabel(key)}</span><textarea value={value} maxLength={10_000} onChange={event => update(key, event.target.value)} disabled={application.status !== 'DRAFT_READY'} /></label>)}<div className="editor-actions"><button className="button secondary" onClick={() => onSave(sections)} disabled={busy || application.status !== 'DRAFT_READY'}>{app.save_version_btn}</button>{nextAction && <button className="button primary" onClick={() => onAction(nextAction)} disabled={busy}>{busy ? '...' : nextLabel}<Send /></button>}{application.status === 'GENERATED' && <button className="button primary" onClick={onDownload}><Download />{app.download_pdf_btn}</button>}</div>{error && <p className="inline-error" role="alert">{error}</p>}</section><aside className="panel snapshot-panel"><FileCheck2 /><h3>Approval snapshot</h3><p>{app.snapshot_desc}</p><dl><div><dt>Passport</dt><dd>v{application.passport_version}</dd></div><div><dt>Policy</dt><dd>v{application.policy_version}</dd></div><div><dt>Template</dt><dd>v{application.template_version} · DOCX</dd></div></dl>{blockingReasons.length > 0 && <div className="blockers"><strong>{app.blockers}</strong>{blockingReasons.map(reason => <span key={reason}>{reason}</span>)}</div>}</aside></div>
}
