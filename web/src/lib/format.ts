const viLabels: Record<string, string> = {
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

const enLabels: Record<string, string> = {
  MET: 'Met',
  NOT_MET: 'Not Met',
  MISSING_INFO: 'Missing Info',
  MISSING: 'Missing',
  EXTRACTED: 'AI Extracted',
  NEEDS_REVIEW: 'Needs Review',
  CONFIRMED: 'Confirmed',
  CONFLICTED: 'Conflicted',
  STALE: 'Stale',
  AVAILABLE: 'Available',
  DRAFT_READY: 'Draft Ready',
  PENDING_REVIEW: 'Pending Review',
  APPROVED: 'Approved',
  GENERATED: 'PDF Generated',
}

function getLang(): 'vi' | 'en' {
  if (typeof document !== 'undefined') {
    const htmlLang = document.documentElement.lang
    if (htmlLang === 'en' || htmlLang === 'vi') {
      return htmlLang
    }
  }
  return 'vi'
}

export function statusLabel(status: string) {
  const lang = getLang()
  const labels = lang === 'en' ? enLabels : viLabels
  return labels[status] ?? status.replaceAll('_', ' ')
}

export function formatMoney(value: number) {
  const lang = getLang()
  return new Intl.NumberFormat(lang === 'en' ? 'en-US' : 'vi-VN', { style: 'currency', currency: 'VND', maximumFractionDigits: 0 }).format(value)
}

export function formatDate(value: string) {
  const lang = getLang()
  return new Intl.DateTimeFormat(lang === 'en' ? 'en-US' : 'vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' }).format(new Date(value))
}

export function displayValue(value: unknown, dataType?: string) {
  const lang = getLang()
  if (value === null || value === undefined || value === '') {
    return lang === 'en' ? 'No data' : 'Chưa có dữ liệu'
  }
  if (Array.isArray(value)) return value.join(', ')
  if (dataType === 'money' && typeof value === 'number') return formatMoney(value)
  if (typeof value === 'boolean') {
    if (lang === 'en') return value ? 'Yes' : 'No'
    return value ? 'Có' : 'Không'
  }
  return String(value)
}
