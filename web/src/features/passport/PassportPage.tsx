import * as Tabs from '@radix-ui/react-tabs'
import { Check, ChevronRight, FileSearch, PencilLine, Quote, ShieldCheck } from 'lucide-react'
import { useState } from 'react'
import { StatusBadge } from '../../components/StatusBadge'
import { displayValue, formatDate } from '../../lib/format'
import type { Candidate, Passport, PassportField } from '../../lib/types'

const groups: Record<string, string[]> = {
  'Danh tính': ['legal_name', 'tax_code', 'legal_form', 'incorporation_date', 'operating_status'],
  'Quy mô & tài chính': ['charter_capital', 'revenue', 'assets', 'employee_count'],
  'Địa lý & hoạt động': ['registered_address', 'province', 'industrial_zone', 'industry_codes', 'products', 'markets'],
  'Công nghệ & năng lực': ['technologies', 'rd_capacity', 'intellectual_property', 'certifications', 'green_project'],
}

export function PassportPage({ passport, candidates, onConfirm, busy }: { passport: Passport; candidates: Candidate[]; onConfirm: (candidate: Candidate) => Promise<void>; busy: boolean }) {
  const [selected, setSelected] = useState<PassportField | null>(null)
  const pending = candidates.filter(candidate => candidate.status !== 'ACCEPTED')
  const confirmAll = async () => { for (const candidate of pending) await onConfirm(candidate) }
  return (
    <>
      <section className="page-heading split-heading"><div><span className="kicker">COMPANY PASSPORT · VERSION {passport.version}</span><h1>Hồ sơ doanh nghiệp <em>có thể kiểm chứng.</em></h1><p>Mỗi dữ kiện đi cùng bằng chứng, mức tin cậy và lịch sử thay đổi.</p></div><div className="heading-actions">{pending.length > 0 && <button className="button primary" disabled={busy} onClick={confirmAll}><Check />Xác nhận tất cả ({pending.length})</button>}</div></section>
      <div className="passport-summary"><div className="company-monogram">{companyMonogram(passport.company_name)}</div><div><h2>{passport.company_name}</h2><p>{passport.website}</p><span>Workspace riêng tư · cập nhật {formatDate(passport.updated_at)}</span></div><div className="passport-score"><strong>{Object.values(passport.fields).filter(field => field.status === 'CONFIRMED').length}/{Object.values(passport.fields).length}</strong><span>dữ kiện xác nhận</span></div></div>
      <Tabs.Root className="passport-tabs" defaultValue="fields">
        <Tabs.List aria-label="Nội dung Company Passport"><Tabs.Trigger value="fields">Dữ kiện</Tabs.Trigger><Tabs.Trigger value="candidates">AI đề xuất <b>{pending.length}</b></Tabs.Trigger><Tabs.Trigger value="history">Lịch sử phiên bản</Tabs.Trigger></Tabs.List>
        <Tabs.Content value="fields"><div className="passport-layout"><div className="field-groups">{Object.entries(groups).map(([group, keys]) => <section className="panel field-group" key={group}><div className="panel-title"><div><span>FIELD GROUP</span><h2>{group}</h2></div></div>{keys.map(key => passport.fields[key]).filter(Boolean).map(field => <button className="field-row" key={field.key} onClick={() => setSelected(field)}><div><span>{field.label}</span><strong>{displayValue(field.value, field.data_type)}</strong></div><StatusBadge status={field.status} /><ChevronRight /></button>)}</section>)}</div><EvidencePanel field={selected ?? Object.values(passport.fields)[0]} /></div></Tabs.Content>
        <Tabs.Content value="candidates"><div className="candidate-list">{pending.length === 0 ? <div className="page-state"><ShieldCheck /><strong>Không còn đề xuất chờ duyệt</strong><span>Passport chỉ dùng các dữ kiện bạn đã xác nhận.</span></div> : pending.map(candidate => <article className="candidate-card" key={candidate.id}><div className="candidate-top"><div><span>{candidate.field_key.replaceAll('_', ' ')}</span><h3>{displayValue(candidate.value, candidate.data_type)}</h3></div><strong>{Math.round(candidate.confidence * 100)}% tin cậy</strong></div><blockquote><Quote />{candidate.evidence.quote}</blockquote><div className="candidate-meta"><span><FileSearch />{candidate.evidence.source_name} · Trang {candidate.evidence.page || 1}</span><button className="button secondary" disabled={busy} onClick={() => onConfirm(candidate)}><Check />Xác nhận</button></div></article>)}</div></Tabs.Content>
        <Tabs.Content value="history"><div className="timeline"><div><b>v{passport.version}</b><span><strong>Phiên bản hiện tại</strong>Đã cập nhật từ xác nhận của người dùng</span></div><div><b>v1</b><span><strong>Khởi tạo Passport</strong>Tạo từ tên doanh nghiệp, website và nhu cầu hỗ trợ</span></div></div></Tabs.Content>
      </Tabs.Root>
    </>
  )
}

function companyMonogram(companyName: string) {
  return companyName.split(/\s+/).filter(Boolean).slice(0, 2).map(word => word[0]?.toLocaleUpperCase('vi')).join('') || 'P2B'
}

function EvidencePanel({ field }: { field?: PassportField }) {
  if (!field) return <aside className="panel evidence-panel"><FileSearch /><h3>Chọn một dữ kiện</h3><p>Nguồn và bằng chứng sẽ xuất hiện tại đây.</p></aside>
  return <aside className="panel evidence-panel"><div className="evidence-head"><span>PROVENANCE</span><button aria-label="Chỉnh sửa dữ kiện"><PencilLine /></button></div><h3>{field.label}</h3><strong className="evidence-value">{displayValue(field.value, field.data_type)}</strong><StatusBadge status={field.status} /><div className="confidence-line"><span>Mức tin cậy</span><strong>{Math.round(field.confidence * 100)}%</strong></div>{field.evidence.map(evidence => <div className="evidence-source" key={evidence.content_hash}><FileSearch /><div><span>{evidence.source_name}</span><small>{evidence.page ? `Trang ${evidence.page} · ` : ''}{formatDate(evidence.observed_at)}</small></div><blockquote>“{evidence.quote}”</blockquote><code>{evidence.content_hash.slice(0, 22)}…</code></div>)}<p className="evidence-note"><ShieldCheck />Confidence chỉ hỗ trợ review, không ảnh hưởng kết quả eligibility.</p></aside>
}
