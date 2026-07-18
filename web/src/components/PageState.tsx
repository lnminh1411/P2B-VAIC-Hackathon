import { CircleAlert, LoaderCircle } from 'lucide-react'

export function LoadingState({ label = 'Đang tải dữ liệu…' }: { label?: string }) {
  return <div className="page-state" role="status" aria-busy="true"><LoaderCircle className="spin" /><strong>{label}</strong><span>Quá trình có thể mất vài giây.</span></div>
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return <div className="page-state error-state" role="alert"><CircleAlert /><strong>Chưa thể hoàn tất</strong><span>{message}</span>{onRetry && <button className="button secondary" onClick={onRetry}>Thử lại</button>}</div>
}

