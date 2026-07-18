import { CheckCircle2, FileClock, FileSearch, ShieldAlert } from 'lucide-react'
import { StatusBadge } from '../../components/StatusBadge'
import { formatDate } from '../../lib/format'
import { useTranslation } from '../../lib/i18n'

type AdminPolicy = { id: string; title: string; agency: string; lifecycle: string; version: number; verified_at: string; template_ready: boolean }

export function AdminPage({ policies }: { policies: AdminPolicy[] }) {
  const { t } = useTranslation()
  const a = t('admin')
  const pending = policies.filter(policy => policy.lifecycle === 'PENDING_REVIEW')

  return (
    <>
      <section className="page-heading split-heading">
        <div>
          <span className="kicker">ADMIN · LEGAL CORPUS</span>
          <h1>{a.h1}<em>{a.h1_em}</em></h1>
          <p>{a.desc}</p>
        </div>
        <span className="admin-lock"><ShieldAlert />{a.admin_lock}</span>
      </section>
      
      <section className="admin-metrics">
        <div>
          <CheckCircle2 />
          <span><strong>{policies.filter(policy => policy.lifecycle === 'ACTIVE').length}</strong>{a.active_policies}</span>
        </div>
        <div>
          <FileClock />
          <span><strong>{pending.length}</strong>{a.pending_review}</span>
        </div>
        <div>
          <FileSearch />
          <span><strong>{policies.filter(policy => policy.template_ready).length}</strong>{a.templates_ready}</span>
        </div>
      </section>
      
      <section className="panel admin-table">
        <div className="panel-title">
          <div>
            <span>POLICY VERSIONS</span>
            <h2>{a.review_queue}</h2>
          </div>
          <button className="button secondary">{a.start_crawl_btn}</button>
        </div>
        
        <div className="table-head">
          <span>{a.table_header_policy}</span>
          <span>{a.table_header_version}</span>
          <span>{a.table_header_template}</span>
          <span>{a.table_header_status}</span>
          <span>{a.table_header_last_checked}</span>
        </div>
        
        {policies.map(policy => (
          <article key={`${policy.id}-${policy.version}`}>
            <div>
              <strong>{policy.title}</strong>
              <span>{policy.agency}</span>
            </div>
            <b>v{policy.version}</b>
            <span>{policy.template_ready ? a.template_approved : a.template_missing}</span>
            <StatusBadge status={policy.lifecycle} />
            <span>{formatDate(policy.verified_at)}</span>
          </article>
        ))}
      </section>
      
      <p className="admin-warning"><ShieldAlert />{a.mvp_warning}</p>
    </>
  )
}

