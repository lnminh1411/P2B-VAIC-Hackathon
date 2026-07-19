import * as Tabs from '@radix-ui/react-tabs'
import { Check, ChevronRight, FileSearch, PencilLine, Quote, ShieldCheck, UploadCloud, X } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { StatusBadge } from '../../components/StatusBadge'
import { displayValue, formatDate } from '../../lib/format'
import type { Candidate, Passport, PassportField, PassportVersion } from '../../lib/types'
import { useTranslation } from '../../lib/i18n'

const groupedFieldKeys = [
  'legal_name', 'tax_code', 'legal_form', 'incorporation_date', 'operating_status',
  'charter_capital', 'revenue', 'assets', 'employee_count', 'fdi_status', 'foreign_ownership_percent', 'women_owned', 'funding_need',
  'registered_address', 'province', 'industrial_zone', 'industry_codes', 'products', 'markets',
  'technologies', 'rd_capacity', 'intellectual_property', 'certifications', 'green_project', 'support_plan'
]

type PassportPageProps = {
  passport: Passport
  candidates: Candidate[]
  versions: PassportVersion[]
  onConfirm: (candidate: Candidate) => Promise<void>
  onSaveField: (fieldKey: string, value: unknown) => Promise<void>
  onRefresh?: (files: File[]) => Promise<void>
  refreshBusy?: boolean
  busy: boolean
}

export function PassportPage({ passport, candidates, versions, onConfirm, onSaveField, onRefresh, refreshBusy = false, busy }: PassportPageProps) {
  const { t } = useTranslation()
  const fieldsLabels = t('fields') as Record<string, string>
  const p = t('passport')

  const groups: Record<string, string[]> = {
    [p.group_identity]: ['legal_name', 'tax_code', 'legal_form', 'incorporation_date', 'operating_status'],
    [p.group_scale_finance]: ['charter_capital', 'revenue', 'assets', 'employee_count', 'fdi_status', 'foreign_ownership_percent', 'women_owned', 'funding_need'],
    [p.group_geography_activity]: ['registered_address', 'province', 'industrial_zone', 'industry_codes', 'products', 'markets'],
    [p.group_tech_capacity]: ['technologies', 'rd_capacity', 'intellectual_property', 'certifications', 'green_project', 'support_plan'],
  }

  const fields = passport.fields
  const visibleFields = groupedFieldKeys.map(key => fields[key]).filter((field): field is PassportField => Boolean(field))
  const [selectedKey, setSelectedKey] = useState('legal_name')
  const [editingKey, setEditingKey] = useState<string>()
  const [refreshError, setRefreshError] = useState<string>()
  const [candidateError, setCandidateError] = useState<string>()
  const selected = fields[selectedKey] ?? visibleFields[0]
  const pending = candidates.filter(candidate => candidate.status !== 'ACCEPTED')
  const confirmedCount = visibleFields.filter(field => field.status === 'CONFIRMED').length
  const confirmOne = async (candidate: Candidate) => {
    setCandidateError(undefined)
    try {
      await onConfirm(candidate)
    } catch (error) {
      setCandidateError(error instanceof Error ? error.message : p.err_confirm_fail_single)
    }
  }
  const confirmAll = async () => {
    setCandidateError(undefined)
    const failed: string[] = []
    for (const candidate of pending) {
      try {
        await onConfirm(candidate)
      } catch {
        failed.push(fieldsLabels[candidate.field_key] || candidate.field_key)
      }
    }
    if (failed.length > 0) setCandidateError(p.err_confirm_fail.replace('{fields}', failed.join(', ')))
  }
  const edit = (field: PassportField) => { setSelectedKey(field.key); setEditingKey(field.key) }
  const save = async (fieldKey: string, value: unknown) => { await onSaveField(fieldKey, value); setEditingKey(undefined) }
  
  const refresh = (files: File[]) => {
    setRefreshError(undefined)
    if (files.length > 10) { setRefreshError(p.err_max_files); return }
    if (files.some(file => file.size > 20 * 1024 * 1024)) { setRefreshError(p.err_file_size); return }
    if (files.some(file => file.type !== 'application/pdf' && !file.name.toLowerCase().endsWith('.pdf'))) { setRefreshError(p.err_file_type); return }
    void onRefresh?.(files).catch(error => setRefreshError(error instanceof Error ? error.message : p.err_update_fail))
  }

  return (
    <>
      <section className="page-heading split-heading">
        <div>
          <span className="kicker">COMPANY PASSPORT</span>
          <h1>{p.h1}<em>{p.h1_em}</em></h1>
          <p>{p.desc}</p>
          {refreshError && <p className="inline-error" role="alert">{refreshError}</p>}
        </div>
        <div className="heading-actions">
          {onRefresh && (
            <label className="button secondary upload-refresh">
              <UploadCloud />
              {refreshBusy ? p.refresh_busy : p.refresh_btn}
              <input aria-label="Tài liệu cập nhật" type="file" accept="application/pdf,.pdf" multiple disabled={refreshBusy} onChange={event => { const files = Array.from(event.target.files ?? []); if (files.length) refresh(files); event.currentTarget.value = '' }} />
            </label>
          )}
          {pending.length > 0 && <button className="button primary" disabled={busy} onClick={confirmAll}><Check />{p.confirm_all_btn.replace('{count}', String(pending.length))}</button>}
        </div>
      </section>
      
      <div className="passport-summary">
        <div className="company-monogram">{companyMonogram(passport.company_name)}</div>
        <div>
          <h2>{passport.company_name}</h2>
          <p>{passport.website}</p>
          <span>{p.workspace_private}{formatDate(passport.updated_at)}</span>
        </div>
        <div className="passport-score">
          <strong>{confirmedCount}/{visibleFields.length}</strong>
          <span>{p.confirmed_facts_label}</span>
        </div>
      </div>
      
      <Tabs.Root className="passport-tabs" defaultValue="fields">
        <Tabs.List aria-label="Nội dung Company Passport">
          <Tabs.Trigger value="fields">{p.tab_facts}</Tabs.Trigger>
          <Tabs.Trigger value="candidates">{p.tab_candidates}<b>{pending.length}</b></Tabs.Trigger>
          <Tabs.Trigger value="history">{p.tab_history}</Tabs.Trigger>
        </Tabs.List>
        
        <Tabs.Content value="fields">
          <div className="passport-layout">
            <div className="field-groups">
              {Object.entries(groups).map(([group, keys]) => (
                <section className="panel field-group" key={group}>
                  <div className="panel-title">
                    <div>
                      <h2>{group}</h2>
                    </div>
                  </div>
                  {keys.map(key => fields[key]).filter((field): field is PassportField => Boolean(field)).map(field => (
                    <div className="field-row" key={field.key}>
                      <button className="field-select" onClick={() => setSelectedKey(field.key)}>
                        <div>
                          <span>{fieldsLabels[field.key] || field.label}</span>
                          <strong>{displayValue(field.value, field.data_type)}</strong>
                        </div>
                        <ChevronRight />
                      </button>
                      <StatusBadge status={field.status} />
                      <button className="field-edit" aria-label={`Chỉnh sửa ${fieldsLabels[field.key] || field.label}`} onClick={() => edit(field)}>
                        <PencilLine />
                      </button>
                    </div>
                  ))}
                </section>
              ))}
            </div>
            {selected && <EvidencePanel key={`${selected.key}-${editingKey === selected.key}`} field={selected} editing={editingKey === selected.key} busy={busy} onEdit={() => edit(selected)} onCancel={() => setEditingKey(undefined)} onSave={save} />}
          </div>
        </Tabs.Content>
        
        <Tabs.Content value="candidates">
          {candidateError && <p className="inline-error" role="alert">{candidateError}</p>}
          <div className="candidate-list">
            {pending.length === 0 ? (
              <div className="page-state">
                <ShieldCheck />
                <strong>{p.no_candidates}</strong>
                <span>{p.no_candidates_desc}</span>
              </div>
            ) : (
              pending.map(candidate => <CandidateCard key={candidate.id} candidate={candidate} busy={busy} onConfirm={confirmOne} />)
            )}
          </div>
        </Tabs.Content>
        
        <Tabs.Content value="history">
          {versions.length === 0 ? (
            <div className="page-state">
              <ShieldCheck />
              <strong>{p.history_empty}</strong>
              <span>{p.history_empty_desc}</span>
            </div>
          ) : (
            <div className="timeline">
              {versions.map((version, index) => (
                <div key={version.version}>
                  <b>v{version.version}</b>
                  <span>
                    <strong>{version.version === 1 ? p.version_initial : index === 0 ? p.version_current : p.history_updated_label}</strong>
                    <small>{formatDate(version.created_at)}</small>
                    {version.changed_fields.length > 0
                      ? p.history_changed_prefix + version.changed_fields.join(', ')
                      : version.version === 1
                        ? p.version_initial_desc
                        : p.history_no_changes}
                  </span>
                </div>
              ))}
            </div>
          )}
        </Tabs.Content>
      </Tabs.Root>
    </>
  )
}

export function CandidateCard({ candidate, busy, onConfirm }: { candidate: Candidate; busy: boolean; onConfirm: (candidate: Candidate) => Promise<void> }) {
  const { t } = useTranslation()
  const p = t('passport')
  return (
    <article className="candidate-card">
      <div className="candidate-top">
        <div>
          <span>{candidate.field_key.replaceAll('_', ' ')}</span>
          <h3>{displayValue(candidate.value, candidate.data_type)}</h3>
        </div>
        <strong>{Math.round(candidate.confidence * 100)}{p.confidence_suffix}</strong>
      </div>
      <blockquote><Quote />{candidate.evidence.quote}</blockquote>
      <div className="candidate-meta">
        <span><FileSearch />{candidate.evidence.source_name}</span>
        <button className="button secondary" disabled={busy} onClick={() => onConfirm(candidate)}><Check />{p.confirm_btn}</button>
      </div>
    </article>
  )
}

function companyMonogram(companyName: string) {
  return companyName.split(/\s+/).filter(Boolean).slice(0, 2).map(word => word[0]?.toLocaleUpperCase('vi')).join('') || 'P2B'
}

function EvidencePanel({ field, editing, busy, onEdit, onCancel, onSave }: { field: PassportField; editing: boolean; busy: boolean; onEdit: () => void; onCancel: () => void; onSave: (fieldKey: string, value: unknown) => Promise<void> }) {
  const { t } = useTranslation()
  const fieldsLabels = t('fields') as Record<string, string>
  const p = t('passport')
  return (
    <aside className="panel evidence-panel">
      <div className="evidence-head">
        <span>{editing ? p.edit_title : ''}</span>
        {editing ? <button aria-label="Hủy chỉnh sửa" onClick={onCancel}><X /></button> : <button aria-label={`Chỉnh sửa dữ kiện ${fieldsLabels[field.key] || field.label}`} onClick={onEdit}><PencilLine /></button>}
      </div>
      {editing ? (
        <FieldEditor field={field} busy={busy} onCancel={onCancel} onSave={onSave} />
      ) : (
        <>
          <h3>{fieldsLabels[field.key] || field.label}</h3>
          <strong className="evidence-value">{displayValue(field.value, field.data_type)}</strong>
          <StatusBadge status={field.status} />
          <div className="confidence-line">
            <span>{p.confidence_label}</span>
            <strong>{Math.round(field.confidence * 100)}%</strong>
          </div>
          {field.evidence.map(evidence => (
            <div className="evidence-source" key={evidence.content_hash}>
              <FileSearch />
              <div>
                <span>{evidence.source_name}</span>
                <small>{formatDate(evidence.observed_at)}</small>
              </div>
              <blockquote>“{evidence.quote}”</blockquote>
              <code>{evidence.content_hash.slice(0, 22)}…</code>
            </div>
          ))}
          {field.evidence.length === 0 && <p className="empty-evidence">{p.no_evidence}</p>}
          <p className="evidence-note"><ShieldCheck />{p.confidence_note}</p>
        </>
      )}
    </aside>
  )
}

function FieldEditor({ field, busy, onCancel, onSave }: { field: PassportField; busy: boolean; onCancel: () => void; onSave: (fieldKey: string, value: unknown) => Promise<void> }) {
  const { t } = useTranslation()
  const fieldsLabels = t('fields') as Record<string, string>
  const p = t('passport')
  const [draft, setDraft] = useState(toDraft(field.value, field.data_type))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string>()
  const inputId = `field-editor-${field.key}`
  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setError(undefined)
    try {
      const value = parseDraft(draft, field.data_type, p)
      setSaving(true)
      await onSave(field.key, value)
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : p.err_save_fail)
    } finally {
      setSaving(false)
    }
  }

  return (
    <form className="field-editor" onSubmit={submit}>
      <h3>{fieldsLabels[field.key] || field.label}</h3>
      <label htmlFor={inputId}>{p.editor_value_label}{fieldsLabels[field.key] || field.label}</label>
      {field.data_type === 'boolean' ? (
        <select id={inputId} value={draft} onChange={event => setDraft(event.target.value)}>
          <option value="">{p.editor_select_placeholder}</option>
          <option value="true">{p.yes}</option>
          <option value="false">{p.no}</option>
        </select>
      ) : field.data_type === 'string_array' ? (
        <textarea id={inputId} value={draft} onChange={event => setDraft(event.target.value)} placeholder={p.editor_textarea_placeholder} rows={4} />
      ) : (
        <input id={inputId} value={draft} onChange={event => setDraft(event.target.value)} type={field.data_type === 'date' ? 'date' : isNumeric(field.data_type) ? 'number' : 'text'} step={field.data_type === 'integer' ? '1' : isNumeric(field.data_type) ? 'any' : undefined} />
      )}
      <small>{field.data_type === 'string_array' ? p.editor_textarea_note : p.editor_input_note}</small>
      {error && <p className="inline-error" role="alert">{error}</p>}
      <div className="field-editor-actions">
        <button type="button" className="button secondary" onClick={onCancel}>{p.cancel_btn}</button>
        <button className="button primary" disabled={busy || saving || !draft.trim()}>{saving ? p.saving : p.save_btn}</button>
      </div>
    </form>
  )
}

function toDraft(value: unknown, dataType: string) {
  if (value === null || value === undefined) return ''
  if (dataType === 'string_array' && Array.isArray(value)) return value.join(', ')
  return String(value)
}

function parseDraft(draft: string, dataType: string, p: Record<string, string>): unknown {
  const value = draft.trim()
  if (!value) throw new Error(p.err_value_required)
  if (dataType === 'string_array') return value.split(/[,\n]/).map(item => item.trim()).filter(Boolean)
  if (dataType === 'boolean') return value === 'true'
  if (isNumeric(dataType)) {
    const parsed = Number(value)
    if (!Number.isFinite(parsed)) throw new Error(p.err_value_number)
    if (dataType === 'integer' && !Number.isInteger(parsed)) throw new Error(p.err_value_integer)
    return parsed
  }
  return value
}

function isNumeric(dataType: string) {
  return dataType === 'integer' || dataType === 'number' || dataType === 'money'
}
