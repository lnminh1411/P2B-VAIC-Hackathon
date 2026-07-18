import { Check, Download, FileCheck2, FileText, LockKeyhole, Send, Sparkles, Upload } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { StatusBadge } from '../../components/StatusBadge'
import { isRetrievedDocument } from '../../lib/policy'
import type { Application, ApplicationTemplate, Checklist, MatchResult } from '../../lib/types'
import { useTranslation } from '../../lib/i18n'

export function ApplicationPage({ policy, checklist, application, templates = [], onCreateChecklist, onMarkAvailable, onCreateApplication, onUploadTemplate, onSave, onAction, onDownload, busy, error }: {
  policy?: MatchResult; checklist?: Checklist; application?: Application; templates?: ApplicationTemplate[]; onCreateChecklist: () => void; onMarkAvailable: (itemId: string) => void; onCreateApplication: (templateId?: string) => void; onUploadTemplate?: (file: File) => Promise<void> | void; onSave: (sections: Record<string, string>) => Promise<Application | void> | Application | void; onAction: (action: 'submit' | 'approve' | 'generate') => void; onDownload: () => void; busy: boolean; error?: string
}) {
  const { t } = useTranslation()
  const app = t('application')

  if (!policy && !application) return <div className="application-empty"><FileCheck2 /><span className="kicker">APPLICATION PREPARATION</span><h1>{app.empty_h1}</h1><p>{app.empty_desc}</p></div>
  const retrieved = policy ? isRetrievedDocument(policy) : false
  const agency = policy?.agency || application?.policy_agency || 'P2B'
  const title = policy?.title || application?.policy_title || 'Bản nháp đã lưu'
  return <><section className="page-heading split-heading"><div><span className="kicker">APPLICATION · {agency}</span><h1>{app.h1}<em>{app.h1_em}</em></h1><p>{title}</p></div>{application && <StatusBadge status={application.status} />}</section>{application ? <ApplicationEditor key={application.id} application={application} onSave={onSave} onAction={onAction} onDownload={onDownload} busy={busy} error={error} /> : !checklist ? <StartCard title={retrieved ? app.document_step1_title : app.step1_title} copy={retrieved ? app.document_step1_desc : app.step1_desc} action={app.step1_btn} onClick={onCreateChecklist} busy={busy} /> : <ChecklistReview checklist={checklist} retrieved={retrieved} templates={templates} onUploadTemplate={onUploadTemplate} onMarkAvailable={onMarkAvailable} onCreate={onCreateApplication} busy={busy} />}</>
}

function StartCard({ title, copy, action, onClick, busy }: { title: string; copy: string; action: string; onClick: () => void; busy: boolean }) {
  const { t } = useTranslation()
  const app = t('application')
  return <section className="start-card panel"><div className="step-orb"><Sparkles /></div><span>{app.step_1_of_3}</span><h2>{title}</h2><p>{copy}</p><button className="button primary" onClick={onClick} disabled={busy}>{action}<Check /></button></section>
}

function ChecklistReview({ checklist, retrieved, templates, onUploadTemplate, onMarkAvailable, onCreate, busy }: { checklist: Checklist; retrieved: boolean; templates: ApplicationTemplate[]; onUploadTemplate?: (file: File) => Promise<void> | void; onMarkAvailable: (itemId: string) => void; onCreate: (templateId?: string) => void; busy: boolean }) {
  const { t } = useTranslation()
  const app = t('application')
  const missing = checklist.items.filter(item => item.required && item.status !== 'AVAILABLE')
  const [selectedTemplate, setSelectedTemplate] = useState('')
  return <div className="application-grid"><section className="panel checklist-panel"><div className="panel-title"><div><span>{app.step_1_of_3} · DOCUMENT CHECKLIST</span><h2>{app.checklist_subtitle}</h2></div><strong>{checklist.items.length - missing.length}/{checklist.items.length}</strong></div><div className="checklist-list">{checklist.items.map(item => <article key={item.id}><div className="doc-icon"><FileText /></div><div><strong>{item.title}</strong><p>{item.description || app.default_desc}</p><small>{item.required ? app.required : app.optional}{(item.field_keys ?? []).length > 0 ? ` · ${(item.field_keys ?? []).join(', ')}` : ''}</small></div><StatusBadge status={item.status} />{item.status !== 'AVAILABLE' && <button className="button secondary" disabled={busy} onClick={() => onMarkAvailable(item.id)}>{app.confirm_available_btn}</button>}</article>)}</div></section><aside className="panel approval-card template-card"><LockKeyhole /><h3>Mẫu hồ sơ</h3><p>Chọn mẫu đã lưu hoặc tải hồ sơ cũ để Gemini tạo bản nháp theo cấu trúc có sẵn.</p><div className="template-options"><label><input type="radio" name="application-template" value="" checked={selectedTemplate === ''} onChange={() => setSelectedTemplate('')} /><span><strong>Mẫu P2B mặc định</strong><small>Dùng dữ liệu Passport và policy</small></span></label>{templates.map(template => <label key={template.id}><input type="radio" name="application-template" value={template.id} checked={selectedTemplate === template.id} onChange={() => setSelectedTemplate(template.id)} /><span><strong>{template.name}</strong><small>{template.filename}{template.placeholders.length > 0 ? ` · ${template.placeholders.length} placeholder` : ''}</small></span></label>)}</div>{onUploadTemplate && <label className="button secondary template-upload"><Upload />Tải mẫu lên<input type="file" accept=".pdf,.docx,.txt" disabled={busy} onChange={event => { const file = event.target.files?.[0]; if (file) void onUploadTemplate(file); event.target.value = '' }} /></label>}<ul><li><Check />{app.gate_policy_pinned}</li><li><Check />{app.gate_passport_pinned}</li><li><Check />{retrieved ? app.gate_template_working : app.gate_template_approved}</li></ul><button className="button primary wide" disabled={missing.length > 0 || busy} onClick={() => onCreate(selectedTemplate || undefined)}>Tạo bản nháp</button>{missing.length > 0 && <small>{app.missing_items_footer.replace('{count}', String(missing.length))}</small>}</aside></div>
}

function ApplicationEditor({ application, onSave, onAction, onDownload, busy, error }: { application: Application; onSave: (sections: Record<string, string>) => Promise<Application | void> | Application | void; onAction: (action: 'submit' | 'approve' | 'generate') => void; onDownload: () => void; busy: boolean; error?: string }) {
  const { t } = useTranslation()
  const app = t('application')
  const [sections, setSections] = useState(application.sections)
  const [saveStatus, setSaveStatus] = useState<'saved' | 'unsaved' | 'saving' | 'error'>('saved')
  const blockingReasons = application.blocking_reasons ?? []
  const update = (key: string, value: string) => { setSections(current => ({ ...current, [key]: value })); setSaveStatus('unsaved') }
  const save = useCallback(async () => {
    setSaveStatus('saving')
    try { await onSave(sections); setSaveStatus('saved') } catch { setSaveStatus('error') }
  }, [onSave, sections])
  useEffect(() => {
    if (application.status !== 'DRAFT_READY' || saveStatus !== 'unsaved' || JSON.stringify(sections) === JSON.stringify(application.sections)) return
    const timeout = window.setTimeout(() => { void save() }, 800)
    return () => window.clearTimeout(timeout)
  }, [application.sections, application.status, save, saveStatus, sections])
  const nextAction = application.status === 'DRAFT_READY' ? 'submit' : application.status === 'PENDING_REVIEW' ? 'approve' : application.status === 'APPROVED' ? 'generate' : undefined
  const nextLabel = nextAction === 'submit' ? app.submit_btn : nextAction === 'approve' ? app.approve_btn : app.generate_btn

  const sectionLabel = (key: string) => {
    return ({
      company_overview: app.label_overview,
      support_need: app.label_need,
      proposal: app.label_proposal
    } as Record<string, string>)[key] ?? key
  }

  const saveLabel = saveStatus === 'saving' ? 'Đang lưu…' : saveStatus === 'saved' ? 'Đã tự động lưu' : saveStatus === 'error' ? 'Lưu thất bại' : 'Chưa lưu'
  return <div className="application-grid"><section className="panel editor-panel"><div className="panel-title"><div><span>{app.step_2_of_3} · HUMAN REVIEW</span><h2>{app.editor_subtitle}</h2></div><StatusBadge status={application.status} /></div>{application.generation_warning && <p className="inline-warning" role="alert">{application.generation_warning}</p>}{Object.entries(sections).map(([key, value]) => <label key={key}><span>{sectionLabel(key)}</span><textarea value={value} maxLength={10_000} onChange={event => update(key, event.target.value)} disabled={application.status !== 'DRAFT_READY'} /></label>)}<div className="editor-actions"><span className={`autosave-status ${saveStatus}`} role="status" aria-live="polite">{saveLabel}</span><button className="button secondary" onClick={() => void save()} disabled={busy || saveStatus === 'saving' || application.status !== 'DRAFT_READY'}>Lưu ngay</button>{nextAction && <button className="button primary" onClick={() => onAction(nextAction)} disabled={busy || saveStatus !== 'saved'}>{busy ? '...' : nextLabel}<Send /></button>}{application.status === 'GENERATED' && <button className="button primary" onClick={onDownload}><Download />{app.download_pdf_btn}</button>}</div>{error && <p className="inline-error" role="alert">{error}</p>}</section><aside className="panel snapshot-panel"><FileCheck2 /><h3>Approval snapshot</h3><p>{app.snapshot_desc}</p><dl><div><dt>Passport</dt><dd>v{application.passport_version}</dd></div><div><dt>Policy</dt><dd>v{application.policy_version}</dd></div><div><dt>Template</dt><dd>{application.template_name || 'Mẫu P2B mặc định'} · v{application.template_version}</dd></div></dl>{blockingReasons.length > 0 && <div className="blockers"><strong>{app.blockers}</strong>{blockingReasons.map(reason => <span key={reason}>{reason}</span>)}</div>}</aside></div>
}
