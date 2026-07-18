import { BellRing, CalendarClock, Check, FileWarning, RefreshCw } from 'lucide-react'
import { formatDate } from '../../lib/format'
import type { Alert } from '../../lib/types'
import { useTranslation } from '../../lib/i18n'

const icons = { POLICY_NEW: BellRing, POLICY_CHANGED: RefreshCw, DEADLINE: CalendarClock, EVIDENCE_STALE: FileWarning }

export function AlertsPage({ alerts, onRead }: { alerts: Alert[]; onRead: (id: string) => void }) {
  const { t } = useTranslation()
  const a = t('alerts')

  const unreadCount = alerts.filter(alert => !alert.read).length

  return (
    <>
      <section className="page-heading">
        <span className="kicker">WATCHLIST · POLICY MONITORING</span>
        <h1>{a.h1}<em>{a.h1_em}</em></h1>
        <p>{a.desc}</p>
      </section>
      
      <div className="alerts-layout">
        <section className="panel alert-list">
          <div className="panel-title">
            <div>
              <span>NOTIFICATION CENTER</span>
              <h2>{a.recent_alerts}</h2>
            </div>
            <strong>{unreadCount} {a.unread}</strong>
          </div>
          
          {alerts.length === 0 ? (
            <div className="page-state">
              <Check />
              <strong>{a.no_new_alerts}</strong>
              <span>{a.not_connected}</span>
            </div>
          ) : (
            alerts.map(alert => {
              const Icon = icons[alert.type as keyof typeof icons] ?? BellRing
              return (
                <article key={alert.id} data-read={alert.read}>
                  <div className="alert-icon">
                    <Icon />
                  </div>
                  <div>
                    <span>{alert.type.replaceAll('_', ' ')}</span>
                    <h3>{alert.title}</h3>
                    <p>{alert.message}</p>
                    <small>{formatDate(alert.occurred_at)} · {a.reviewed_source}</small>
                  </div>
                  {!alert.read && (
                    <button 
                      onClick={() => onRead(alert.id)} 
                      aria-label={a.mark_read_aria.replace('{title}', alert.title)}
                    >
                      <Check />
                    </button>
                  )}
                </article>
              )
            })
          )}
        </section>
        
        <aside className="panel watch-settings">
          <span>{a.watchlist_status}</span>
          <h3>{a.watchlist_title}</h3>
          <dl>
            <div>
              <dt>{a.setting_new_policy}</dt>
              <dd>{a.not_activated}</dd>
            </div>
            <div>
              <dt>{a.setting_deadline}</dt>
              <dd>{a.not_activated}</dd>
            </div>
            <div>
              <dt>{a.setting_evidence}</dt>
              <dd>{a.not_activated}</dd>
            </div>
            <div>
              <dt>{a.setting_upcoming}</dt>
              <dd>{a.not_activated}</dd>
            </div>
          </dl>
          <p>{a.settings_note}</p>
        </aside>
      </div>
    </>
  )
}
