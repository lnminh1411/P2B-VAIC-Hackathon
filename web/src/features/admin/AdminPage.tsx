import { CheckCircle2, FileClock, FileSearch, ShieldAlert } from 'lucide-react'
import { StatusBadge } from '../../components/StatusBadge'
import { formatDate } from '../../lib/format'

type AdminPolicy = { id: string; title: string; agency: string; lifecycle: string; version: number; verified_at: string; template_ready: boolean }

export function AdminPage({ policies }: { policies: AdminPolicy[] }) {
  const pending = policies.filter(policy => policy.lifecycle === 'PENDING_REVIEW')
  return <><section className="page-heading split-heading"><div><span className="kicker">ADMIN · LEGAL CORPUS</span><h1>AI thu thập. <em>Reviewer quyết định.</em></h1><p>Không policy candidate nào được tìm kiếm hoặc matching trước khi citation và rule được kiểm tra.</p></div><span className="admin-lock"><ShieldAlert />Admin role · Supabase app_metadata</span></section><section className="admin-metrics"><div><CheckCircle2 /><span><strong>{policies.filter(policy => policy.lifecycle === 'ACTIVE').length}</strong>policy đang active</span></div><div><FileClock /><span><strong>{pending.length}</strong>chờ review</span></div><div><FileSearch /><span><strong>{policies.filter(policy => policy.template_ready).length}</strong>template sẵn sàng</span></div></section><section className="panel admin-table"><div className="panel-title"><div><span>POLICY VERSIONS</span><h2>Review queue</h2></div><button className="button secondary">Bắt đầu crawl run</button></div><div className="table-head"><span>Chính sách</span><span>Phiên bản</span><span>Template</span><span>Trạng thái</span><span>Kiểm tra gần nhất</span></div>{policies.map(policy => <article key={`${policy.id}-${policy.version}`}><div><strong>{policy.title}</strong><span>{policy.agency}</span></div><b>v{policy.version}</b><span>{policy.template_ready ? 'Đã duyệt' : 'Chưa có'}</span><StatusBadge status={policy.lifecycle} /><span>{formatDate(policy.verified_at)}</span></article>)}</section><p className="admin-warning"><ShieldAlert />Màn hình MVP hiển thị review queue. Publish/retire phải thêm Supabase admin JWT trước production.</p></>
}

