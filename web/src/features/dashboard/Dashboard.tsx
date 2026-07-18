import { ArrowUpRight, BellRing, Building2, CheckCircle2, CircleDashed, FileText, SearchCheck, Sparkles } from 'lucide-react'
import type { Page } from '../../components/Shell'
import { StatusBadge } from '../../components/StatusBadge'
import { displayValue } from '../../lib/format'
import type { Passport } from '../../lib/types'
import { useTranslation } from '../../lib/i18n'

export function Dashboard({ passport, onNavigate }: { passport: Passport; onNavigate: (page: Page) => void }) {
  const { t } = useTranslation()
  const d = t('dashboard')

  const fields = Object.values(passport.fields)
  const confirmed = fields.filter(field => field.status === 'CONFIRMED').length
  const completeness = fields.length ? Math.round((confirmed / fields.length) * 100) : 0
  const keyFacts = ['tax_code', 'charter_capital', 'employee_count', 'province'].map(key => passport.fields[key]).filter(Boolean)
  
  const pendingCount = fields.filter(field => field.status !== 'CONFIRMED').length
  const action1Desc = fields.length === 0
    ? d.action_1_desc_empty
    : d.action_1_desc_pending.replace('{count}', String(pendingCount))

  return (
    <>
      <section className="page-heading split-heading">
        <div>
          <span className="kicker">{d.kicker}</span>
          <h1>{d.h1}<em>{d.h1_em}</em></h1>
          <p>{d.p}</p>
        </div>
        <button className="button primary" onClick={() => onNavigate('opportunities')}>{d.find_opportunities_btn}<SearchCheck /></button>
      </section>
      
      <section className="metric-strip">
        <div>
          <span>{d.metric_completeness}</span>
          <strong>{completeness}%</strong>
          <small><CheckCircle2 />{confirmed}{d.metric_confirmed_suffix}</small>
        </div>
        <div>
          <span>{d.metric_opportunities}</span>
          <strong>—</strong>
          <small><CircleDashed />{d.metric_opportunities_desc}</small>
        </div>
        <div>
          <span>{d.metric_deadline}</span>
          <strong>—</strong>
          <small><BellRing />{d.metric_deadline_desc}</small>
        </div>
        <div>
          <span>{d.metric_application}</span>
          <strong>—</strong>
          <small><FileText />{d.metric_application_desc}</small>
        </div>
      </section>
      
      <div className="dashboard-grid">
        <section className="panel company-snapshot">
          <div className="panel-title">
            <div>
              <span>{d.company_passport_title}</span>
              <h2>{passport.company_name}</h2>
            </div>
            <button className="text-button" onClick={() => onNavigate('passport')}>{d.view_full_btn}<ArrowUpRight /></button>
          </div>
          <div className="company-meta">
            <span><Building2 />{passport.website || d.no_website}</span>
            <span><Sparkles />{passport.support_needs.join(' · ')}</span>
          </div>
          <div className="fact-grid">
            {keyFacts.map(field => (
              <div key={field.key}>
                <span>{field.label}</span>
                <strong>{displayValue(field.value, field.data_type)}</strong>
                <StatusBadge status={field.status} />
              </div>
            ))}
          </div>
        </section>
        
        <aside className="panel action-rail">
          <div className="panel-title">
            <div>
              <span>{d.next_actions_title}</span>
              <h2>{d.next_actions_subtitle}</h2>
            </div>
          </div>
          <ol>
            <li>
              <b>01</b>
              <div>
                <strong>{d.action_1_title}</strong>
                <span>{action1Desc}</span>
              </div>
              <button onClick={() => onNavigate('passport')}>{d.action_open_btn}</button>
            </li>
            <li>
              <b>02</b>
              <div>
                <strong>{d.action_2_title}</strong>
                <span>{d.action_2_desc}</span>
              </div>
              <button onClick={() => onNavigate('opportunities')}>{d.action_open_btn}</button>
            </li>
            <li>
              <b>03</b>
              <div>
                <strong>{d.action_3_title}</strong>
                <span>{d.action_3_desc}</span>
              </div>
              <button onClick={() => onNavigate('application')}>{d.action_open_btn}</button>
            </li>
          </ol>
        </aside>
      </div>
      
      <section className="panel trust-banner">
        <div>
          <CheckCircle2 />
          <span><strong>{d.trust_banner_title}</strong>{d.trust_banner_desc}</span>
        </div>
        <p>{d.trust_banner_footer}</p>
      </section>
    </>
  )
}
