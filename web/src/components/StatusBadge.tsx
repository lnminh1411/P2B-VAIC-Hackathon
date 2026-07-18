import { Check, CircleAlert, CircleDashed, Clock3, X } from 'lucide-react'
import { statusLabel } from '../lib/format'

const good = new Set(['MET', 'CONFIRMED', 'AVAILABLE', 'APPROVED', 'GENERATED', 'ACTIVE'])
const bad = new Set(['NOT_MET', 'CONFLICTED', 'FAILED', 'RETIRED'])
const waiting = new Set(['MISSING_INFO', 'MISSING', 'NEEDS_REVIEW', 'STALE', 'PENDING_REVIEW'])

export function StatusBadge({ status }: { status: string }) {
  const Icon = good.has(status) ? Check : bad.has(status) ? X : waiting.has(status) ? Clock3 : status === 'EXTRACTED' ? CircleDashed : CircleAlert
  const tone = good.has(status) ? 'positive' : bad.has(status) ? 'negative' : waiting.has(status) ? 'warning' : 'neutral'
  return <span className="status-badge" data-tone={tone}><Icon aria-hidden="true" />{statusLabel(status)}</span>
}

