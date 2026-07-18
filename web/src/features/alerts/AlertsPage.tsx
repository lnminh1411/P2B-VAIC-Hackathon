import { BellRing, CalendarClock, Check, FileWarning, RefreshCw } from 'lucide-react'
import { formatDate } from '../../lib/format'
import type { Alert } from '../../lib/types'

const icons = { POLICY_NEW: BellRing, POLICY_CHANGED: RefreshCw, DEADLINE: CalendarClock, EVIDENCE_STALE: FileWarning }

export function AlertsPage({ alerts, onRead }: { alerts: Alert[]; onRead: (id: string) => void }) {
  return <><section className="page-heading"><span className="kicker">WATCHLIST · POLICY MONITORING</span><h1>Thay đổi quan trọng, <em>không để doanh nghiệp bỏ lỡ.</em></h1><p>P2B so sánh nguồn chính thức. Thay đổi chỉ tạo cảnh báo sau khi admin duyệt.</p></section><div className="alerts-layout"><section className="panel alert-list"><div className="panel-title"><div><span>NOTIFICATION CENTER</span><h2>Cảnh báo gần đây</h2></div><strong>{alerts.filter(alert => !alert.read).length} chưa đọc</strong></div>{alerts.length === 0 ? <div className="page-state"><Check /><strong>Không có cảnh báo mới</strong><span>Watchlist vẫn đang hoạt động.</span></div> : alerts.map(alert => { const Icon = icons[alert.type as keyof typeof icons] ?? BellRing; return <article key={alert.id} data-read={alert.read}><div className="alert-icon"><Icon /></div><div><span>{alert.type.replaceAll('_', ' ')}</span><h3>{alert.title}</h3><p>{alert.message}</p><small>{formatDate(alert.occurred_at)} · Nguồn đã review</small></div>{!alert.read && <button onClick={() => onRead(alert.id)} aria-label={`Đánh dấu đã đọc: ${alert.title}`}><Check /></button>}</article> })}</section><aside className="panel watch-settings"><span>ĐANG THEO DÕI</span><h3>Watchlist doanh nghiệp</h3><dl><div><dt>Chính sách mới</dt><dd>Bật</dd></div><div><dt>Thay đổi deadline</dt><dd>Bật</dd></div><div><dt>Evidence sắp cũ</dt><dd>Bật</dd></div><div><dt>Hồ sơ gần hạn</dt><dd>Bật</dd></div></dl><p>Email và push notification sẽ có sau MVP. Hiện cảnh báo chỉ nằm trong workspace.</p></aside></div></>
}

