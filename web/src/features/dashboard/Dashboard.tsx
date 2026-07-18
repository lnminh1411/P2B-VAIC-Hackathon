import { ArrowUpRight, BellRing, Building2, CheckCircle2, CircleDashed, FileText, SearchCheck, Sparkles } from 'lucide-react'
import type { Page } from '../../components/Shell'
import { StatusBadge } from '../../components/StatusBadge'
import { displayValue } from '../../lib/format'
import type { Passport } from '../../lib/types'

export function Dashboard({ passport, onNavigate }: { passport: Passport; onNavigate: (page: Page) => void }) {
  const fields = Object.values(passport.fields)
  const confirmed = fields.filter(field => field.status === 'CONFIRMED').length
  const completeness = fields.length ? Math.round((confirmed / fields.length) * 100) : 0
  const keyFacts = ['tax_code', 'charter_capital', 'employee_count', 'province'].map(key => passport.fields[key]).filter(Boolean)
  return (
    <>
      <section className="page-heading split-heading"><div><span className="kicker">GOOD MORNING · WORKSPACE OVERVIEW</span><h1>Cơ hội phù hợp bắt đầu từ <em>dữ liệu đáng tin.</em></h1><p>Passport đang được theo dõi. P2B chỉ dùng dữ kiện đã xác nhận để kiểm tra điều kiện.</p></div><button className="button primary" onClick={() => onNavigate('opportunities')}>Tìm cơ hội ngay<SearchCheck /></button></section>
      <section className="metric-strip">
        <div><span>Độ hoàn thiện Passport</span><strong>{completeness}%</strong><small><CheckCircle2 />{confirmed} dữ kiện đã xác nhận</small></div>
        <div><span>Cơ hội đang theo dõi</span><strong>03</strong><small><CircleDashed />1 cơ hội cần bổ sung</small></div>
        <div><span>Deadline gần nhất</span><strong>68 ngày</strong><small><BellRing />Đã bật cảnh báo</small></div>
        <div><span>Hồ sơ đang chuẩn bị</span><strong>01</strong><small><FileText />Chưa gửi duyệt</small></div>
      </section>
      <div className="dashboard-grid">
        <section className="panel company-snapshot"><div className="panel-title"><div><span>COMPANY PASSPORT</span><h2>{passport.company_name}</h2></div><button className="text-button" onClick={() => onNavigate('passport')}>Xem đầy đủ<ArrowUpRight /></button></div><div className="company-meta"><span><Building2 />{passport.website || 'Chưa có website'}</span><span><Sparkles />{passport.support_needs.join(' · ')}</span></div><div className="fact-grid">{keyFacts.map(field => <div key={field.key}><span>{field.label}</span><strong>{displayValue(field.value, field.data_type)}</strong><StatusBadge status={field.status} /></div>)}</div></section>
        <aside className="panel action-rail"><div className="panel-title"><div><span>NEXT BEST ACTIONS</span><h2>Việc cần làm</h2></div></div><ol><li><b>01</b><div><strong>Xác nhận dữ liệu đã trích xuất</strong><span>Còn {fields.filter(field => field.status !== 'CONFIRMED').length} dữ kiện cần review</span></div><button onClick={() => onNavigate('passport')}>Review</button></li><li><b>02</b><div><strong>Kiểm tra chương trình GreenTech</strong><span>Khả năng phù hợp cao · còn thiếu 1 điều kiện</span></div><button onClick={() => onNavigate('opportunities')}>Mở</button></li><li><b>03</b><div><strong>Hoàn thiện kế hoạch sử dụng vốn</strong><span>Cần cho bộ hồ sơ đổi mới sáng tạo</span></div><button onClick={() => onNavigate('application')}>Soạn</button></li></ol></aside>
      </div>
      <section className="panel trust-banner"><div><CheckCircle2 /><span><strong>Evidence-first matching</strong> Mọi kết quả đều dẫn về dữ kiện doanh nghiệp và điều khoản chính sách.</span></div><p>AI đề xuất · Rule engine kiểm tra · Con người phê duyệt</p></section>
    </>
  )
}

