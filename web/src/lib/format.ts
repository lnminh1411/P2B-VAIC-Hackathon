const labels: Record<string, string> = {
  MET: 'Đã đáp ứng',
  NOT_MET: 'Chưa đáp ứng',
  MISSING_INFO: 'Thiếu thông tin',
  MISSING: 'Còn thiếu',
  EXTRACTED: 'AI trích xuất',
  NEEDS_REVIEW: 'Cần xác nhận',
  CONFIRMED: 'Đã xác nhận',
  CONFLICTED: 'Có mâu thuẫn',
  STALE: 'Cần cập nhật',
  AVAILABLE: 'Đã có',
  DRAFT_READY: 'Sẵn sàng chỉnh sửa',
  PENDING_REVIEW: 'Đang chờ duyệt',
  APPROVED: 'Đã duyệt',
  GENERATED: 'Đã tạo PDF',
}

export function statusLabel(status: string) {
  return labels[status] ?? status.replaceAll('_', ' ')
}

export function formatMoney(value: number) {
  return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND', maximumFractionDigits: 0 }).format(value)
}

export function formatDate(value: string) {
  return new Intl.DateTimeFormat('vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' }).format(new Date(value))
}

export function displayValue(value: unknown, dataType?: string) {
  if (value === null || value === undefined || value === '') return 'Chưa có dữ liệu'
  if (Array.isArray(value)) return value.join(', ')
  if (dataType === 'number' && typeof value === 'number' && value >= 1_000_000) return formatMoney(value)
  if (typeof value === 'boolean') return value ? 'Có' : 'Không'
  return String(value)
}

