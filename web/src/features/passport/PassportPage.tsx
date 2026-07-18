import * as Tabs from '@radix-ui/react-tabs'
import { Check, ChevronRight, FileSearch, PencilLine, Quote, ShieldCheck, UploadCloud, X } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { StatusBadge } from '../../components/StatusBadge'
import { displayValue, formatDate } from '../../lib/format'
import type { Candidate, Passport, PassportField } from '../../lib/types'

const groups: Record<string, string[]> = {
  'Danh tính': ['legal_name', 'tax_code', 'legal_form', 'incorporation_date', 'operating_status'],
  'Quy mô & tài chính': ['charter_capital', 'revenue', 'assets', 'employee_count', 'fdi_status', 'foreign_ownership_percent', 'women_owned', 'funding_need'],
  'Địa lý & hoạt động': ['registered_address', 'province', 'industrial_zone', 'industry_codes', 'products', 'markets'],
  'Công nghệ & năng lực': ['technologies', 'rd_capacity', 'intellectual_property', 'certifications', 'green_project', 'support_plan'],
}

const groupedFieldKeys = Object.values(groups).flat()

type PassportPageProps = {
  passport: Passport
  candidates: Candidate[]
  onConfirm: (candidate: Candidate) => Promise<void>
  onSaveField: (fieldKey: string, value: unknown) => Promise<void>
  onRefresh?: (files: File[]) => Promise<void>
  refreshBusy?: boolean
  busy: boolean
}

export function PassportPage({ passport, candidates, onConfirm, onSaveField, onRefresh, refreshBusy = false, busy }: PassportPageProps) {
  const fields = passport.fields
  const visibleFields = groupedFieldKeys.map(key => fields[key]).filter((field): field is PassportField => Boolean(field))
  const [selectedKey, setSelectedKey] = useState('legal_name')
  const [editingKey, setEditingKey] = useState<string>()
  const [refreshError, setRefreshError] = useState<string>()
  const selected = fields[selectedKey] ?? visibleFields[0]
  const pending = candidates.filter(candidate => candidate.status !== 'ACCEPTED')
  const confirmedCount = visibleFields.filter(field => field.status === 'CONFIRMED').length
  const confirmAll = async () => { for (const candidate of pending) await onConfirm(candidate) }
  const edit = (field: PassportField) => { setSelectedKey(field.key); setEditingKey(field.key) }
  const save = async (fieldKey: string, value: unknown) => { await onSaveField(fieldKey, value); setEditingKey(undefined) }
  const refresh = (files: File[]) => {
    setRefreshError(undefined)
    if (files.length > 10) { setRefreshError('Tối đa 10 file PDF.'); return }
    if (files.some(file => file.size > 20 * 1024 * 1024)) { setRefreshError('Mỗi file PDF phải nhỏ hơn hoặc bằng 20 MB.'); return }
    if (files.some(file => file.type !== 'application/pdf' && !file.name.toLowerCase().endsWith('.pdf'))) { setRefreshError('Chỉ chấp nhận file PDF.'); return }
    void onRefresh?.(files).catch(error => setRefreshError(error instanceof Error ? error.message : 'Không thể cập nhật tài liệu'))
  }

  return (
    <>
      <section className="page-heading split-heading"><div><span className="kicker">COMPANY PASSPORT · VERSION {passport.version}</span><h1>Hồ sơ doanh nghiệp <em>có thể kiểm chứng.</em></h1><p>Mỗi dữ kiện đi cùng bằng chứng, mức tin cậy và lịch sử thay đổi.</p>{refreshError && <p className="inline-error" role="alert">{refreshError}</p>}</div><div className="heading-actions">{onRefresh && <label className="button secondary upload-refresh"><UploadCloud />{refreshBusy ? 'Đang phân tích…' : 'Cập nhật tài liệu'}<input aria-label="Tài liệu cập nhật" type="file" accept="application/pdf,.pdf" multiple disabled={refreshBusy} onChange={event => { const files = Array.from(event.target.files ?? []); if (files.length) refresh(files); event.currentTarget.value = '' }} /></label>}{pending.length > 0 && <button className="button primary" disabled={busy} onClick={confirmAll}><Check />Xác nhận tất cả ({pending.length})</button>}</div></section>
      <div className="passport-summary"><div className="company-monogram">{companyMonogram(passport.company_name)}</div><div><h2>{passport.company_name}</h2><p>{passport.website}</p><span>Workspace riêng tư · cập nhật {formatDate(passport.updated_at)}</span></div><div className="passport-score"><strong>{confirmedCount}/{visibleFields.length}</strong><span>dữ kiện xác nhận</span></div></div>
      <Tabs.Root className="passport-tabs" defaultValue="fields">
        <Tabs.List aria-label="Nội dung Company Passport"><Tabs.Trigger value="fields">Dữ kiện</Tabs.Trigger><Tabs.Trigger value="candidates">AI đề xuất <b>{pending.length}</b></Tabs.Trigger><Tabs.Trigger value="history">Lịch sử phiên bản</Tabs.Trigger></Tabs.List>
        <Tabs.Content value="fields"><div className="passport-layout"><div className="field-groups">{Object.entries(groups).map(([group, keys]) => <section className="panel field-group" key={group}><div className="panel-title"><div><span>FIELD GROUP</span><h2>{group}</h2></div></div>{keys.map(key => fields[key]).filter((field): field is PassportField => Boolean(field)).map(field => <div className="field-row" key={field.key}><button className="field-select" onClick={() => setSelectedKey(field.key)}><div><span>{field.label}</span><strong>{displayValue(field.value, field.data_type)}</strong></div><ChevronRight /></button><StatusBadge status={field.status} /><button className="field-edit" aria-label={`Chỉnh sửa ${field.label}`} onClick={() => edit(field)}><PencilLine /></button></div>)}</section>)}</div>{selected && <EvidencePanel key={`${selected.key}-${editingKey === selected.key}`} field={selected} editing={editingKey === selected.key} busy={busy} onEdit={() => edit(selected)} onCancel={() => setEditingKey(undefined)} onSave={save} />}</div></Tabs.Content>
        <Tabs.Content value="candidates"><div className="candidate-list">{pending.length === 0 ? <div className="page-state"><ShieldCheck /><strong>Không còn đề xuất chờ duyệt</strong><span>Passport chỉ dùng các dữ kiện bạn đã xác nhận.</span></div> : pending.map(candidate => <CandidateCard key={candidate.id} candidate={candidate} busy={busy} onConfirm={onConfirm} />)}</div></Tabs.Content>
        <Tabs.Content value="history"><div className="timeline"><div><b>v{passport.version}</b><span><strong>Phiên bản hiện tại</strong>Đã cập nhật từ xác nhận của người dùng</span></div><div><b>v1</b><span><strong>Khởi tạo Passport</strong>Tạo từ tên doanh nghiệp, website và nhu cầu hỗ trợ</span></div></div></Tabs.Content>
      </Tabs.Root>
    </>
  )
}

export function CandidateCard({ candidate, busy, onConfirm }: { candidate: Candidate; busy: boolean; onConfirm: (candidate: Candidate) => Promise<void> }) {
  return <article className="candidate-card"><div className="candidate-top"><div><span>{candidate.field_key.replaceAll('_', ' ')}</span><h3>{displayValue(candidate.value, candidate.data_type)}</h3></div><strong>{Math.round(candidate.confidence * 100)}% tin cậy</strong></div><blockquote><Quote />{candidate.evidence.quote}</blockquote><div className="candidate-meta"><span><FileSearch />{candidate.evidence.source_name}</span><button className="button secondary" disabled={busy} onClick={() => onConfirm(candidate)}><Check />Xác nhận</button></div></article>
}

function companyMonogram(companyName: string) {
  return companyName.split(/\s+/).filter(Boolean).slice(0, 2).map(word => word[0]?.toLocaleUpperCase('vi')).join('') || 'P2B'
}

function EvidencePanel({ field, editing, busy, onEdit, onCancel, onSave }: { field: PassportField; editing: boolean; busy: boolean; onEdit: () => void; onCancel: () => void; onSave: (fieldKey: string, value: unknown) => Promise<void> }) {
  return <aside className="panel evidence-panel"><div className="evidence-head"><span>{editing ? 'CHỈNH SỬA DỮ KIỆN' : 'PROVENANCE'}</span>{editing ? <button aria-label="Hủy chỉnh sửa" onClick={onCancel}><X /></button> : <button aria-label={`Chỉnh sửa dữ kiện ${field.label}`} onClick={onEdit}><PencilLine /></button>}</div>{editing ? <FieldEditor field={field} busy={busy} onCancel={onCancel} onSave={onSave} /> : <><h3>{field.label}</h3><strong className="evidence-value">{displayValue(field.value, field.data_type)}</strong><StatusBadge status={field.status} /><div className="confidence-line"><span>Mức tin cậy</span><strong>{Math.round(field.confidence * 100)}%</strong></div>{field.evidence.map(evidence => <div className="evidence-source" key={evidence.content_hash}><FileSearch /><div><span>{evidence.source_name}</span><small>{formatDate(evidence.observed_at)}</small></div><blockquote>“{evidence.quote}”</blockquote><code>{evidence.content_hash.slice(0, 22)}…</code></div>)}{field.evidence.length === 0 && <p className="empty-evidence">Không tìm thấy dữ kiện này trong tài liệu. Bạn có thể tự nhập bằng nút chỉnh sửa.</p>}<p className="evidence-note"><ShieldCheck />Confidence chỉ hỗ trợ review, không ảnh hưởng kết quả eligibility.</p></>}</aside>
}

function FieldEditor({ field, busy, onCancel, onSave }: { field: PassportField; busy: boolean; onCancel: () => void; onSave: (fieldKey: string, value: unknown) => Promise<void> }) {
  const [draft, setDraft] = useState(toDraft(field.value, field.data_type))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string>()
  const inputId = `field-editor-${field.key}`
  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setError(undefined)
    try {
      const value = parseDraft(draft, field.data_type)
      setSaving(true)
      await onSave(field.key, value)
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Không thể lưu dữ kiện')
    } finally {
      setSaving(false)
    }
  }

  return <form className="field-editor" onSubmit={submit}><h3>{field.label}</h3><label htmlFor={inputId}>Giá trị {field.label}</label>{field.data_type === 'boolean' ? <select id={inputId} value={draft} onChange={event => setDraft(event.target.value)}><option value="">Chọn giá trị</option><option value="true">Có</option><option value="false">Không</option></select> : field.data_type === 'string_array' ? <textarea id={inputId} value={draft} onChange={event => setDraft(event.target.value)} placeholder="Nhập từng mục, ngăn cách bằng dấu phẩy" rows={4} /> : <input id={inputId} value={draft} onChange={event => setDraft(event.target.value)} type={field.data_type === 'date' ? 'date' : isNumeric(field.data_type) ? 'number' : 'text'} step={field.data_type === 'integer' ? '1' : isNumeric(field.data_type) ? 'any' : undefined} />}<small>{field.data_type === 'string_array' ? 'Có thể ngăn cách nhiều giá trị bằng dấu phẩy hoặc xuống dòng.' : 'Dữ kiện do bạn nhập sẽ được ghi nhận là nguồn người dùng xác nhận.'}</small>{error && <p className="inline-error" role="alert">{error}</p>}<div className="field-editor-actions"><button type="button" className="button secondary" onClick={onCancel}>Hủy</button><button className="button primary" disabled={busy || saving || !draft.trim()}>{saving ? 'Đang lưu…' : 'Lưu thay đổi'}</button></div></form>
}

function toDraft(value: unknown, dataType: string) {
  if (value === null || value === undefined) return ''
  if (dataType === 'string_array' && Array.isArray(value)) return value.join(', ')
  return String(value)
}

function parseDraft(draft: string, dataType: string): unknown {
  const value = draft.trim()
  if (!value) throw new Error('Vui lòng nhập giá trị')
  if (dataType === 'string_array') return value.split(/[,\n]/).map(item => item.trim()).filter(Boolean)
  if (dataType === 'boolean') return value === 'true'
  if (isNumeric(dataType)) {
    const parsed = Number(value)
    if (!Number.isFinite(parsed)) throw new Error('Giá trị phải là số hợp lệ')
    if (dataType === 'integer' && !Number.isInteger(parsed)) throw new Error('Giá trị phải là số nguyên')
    return parsed
  }
  return value
}

function isNumeric(dataType: string) {
  return dataType === 'integer' || dataType === 'number' || dataType === 'money'
}
