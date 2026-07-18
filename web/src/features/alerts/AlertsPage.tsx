import { BellRing, CalendarClock, Check, FileWarning, RefreshCw } from 'lucide-react'
import { formatDate } from '../../lib/format'
import type { Alert, WatchlistSettings } from '../../lib/types'
import { useTranslation } from '../../lib/i18n'

const icons = { POLICY_NEW: BellRing, POLICY_CHANGED: RefreshCw, DEADLINE: CalendarClock, EVIDENCE_STALE: FileWarning }

export function AlertsPage({ 
  alerts, 
  settings, 
  onRead, 
  onUpdateSettings 
}: { 
  alerts: Alert[]
  settings: WatchlistSettings
  onRead: (id: string) => void
  onUpdateSettings: (settings: WatchlistSettings) => void 
}) {
  const { t } = useTranslation()
  const a = t('alerts')

  const unreadCount = alerts.filter(alert => !alert.read).length
  const isAnyActive = settings.new_policies || settings.deadline_changes || settings.stale_evidence || settings.upcoming_deadlines

  const toggle = (key: keyof WatchlistSettings) => {
    onUpdateSettings({
      ...settings,
      [key]: !settings[key]
    })
  }

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
              <span>{isAnyActive ? 'Policy monitoring production is active.' : a.not_connected}</span>
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
              <dd>
                <button 
                  onClick={() => toggle('new_policies')} 
                  className={`watchlist-toggle-btn ${settings.new_policies ? 'active' : ''}`}
                >
                  {settings.new_policies ? a.active : a.not_activated}
                </button>
              </dd>
            </div>
            <div>
              <dt>{a.setting_deadline}</dt>
              <dd>
                <button 
                  onClick={() => toggle('deadline_changes')} 
                  className={`watchlist-toggle-btn ${settings.deadline_changes ? 'active' : ''}`}
                >
                  {settings.deadline_changes ? a.active : a.not_activated}
                </button>
              </dd>
            </div>
            <div>
              <dt>{a.setting_evidence}</dt>
              <dd>
                <button 
                  onClick={() => toggle('stale_evidence')} 
                  className={`watchlist-toggle-btn ${settings.stale_evidence ? 'active' : ''}`}
                >
                  {settings.stale_evidence ? a.active : a.not_activated}
                </button>
              </dd>
            </div>
            <div>
              <dt>{a.setting_upcoming}</dt>
              <dd>
                <button 
                  onClick={() => toggle('upcoming_deadlines')} 
                  className={`watchlist-toggle-btn ${settings.upcoming_deadlines ? 'active' : ''}`}
                >
                  {settings.upcoming_deadlines ? a.active : a.not_activated}
                </button>
              </dd>
            </div>
          </dl>
          <p>{a.settings_note}</p>
        </aside>
      </div>
    </>
  )
}
