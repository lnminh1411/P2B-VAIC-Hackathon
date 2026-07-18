import { CircleAlert, LoaderCircle } from 'lucide-react'
import { useTranslation } from '../lib/i18n'

export function LoadingState({ label }: { label?: string }) {
  const { t } = useTranslation()
  const pageStateLabels = t('page_state')
  const displayLabel = label || pageStateLabels.loading_default
  
  return <div className="page-state" role="status" aria-busy="true">
    <LoaderCircle className="spin" />
    <strong>{displayLabel}</strong>
    <span>{pageStateLabels.loading_sub}</span>
  </div>
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  const { t } = useTranslation()
  const pageStateLabels = t('page_state')
  
  return <div className="page-state error-state" role="alert">
    <CircleAlert />
    <strong>{pageStateLabels.error_title}</strong>
    <span>{message}</span>
    {onRetry && <button className="button secondary" onClick={onRetry}>{pageStateLabels.retry}</button>}
  </div>
}

