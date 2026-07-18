import { Bell, Building2, FileCheck2, LayoutDashboard, LibraryBig, Menu, SearchCheck, ShieldCheck, X } from 'lucide-react'
import { AnimatePresence, motion } from 'motion/react'
import { useState, type ReactNode } from 'react'

export type Page = 'overview' | 'passport' | 'opportunities' | 'application' | 'alerts' | 'admin'

const navigation: Array<{ id: Page; label: string; icon: typeof LayoutDashboard }> = [
  { id: 'overview', label: 'Tổng quan', icon: LayoutDashboard },
  { id: 'passport', label: 'Company Passport', icon: Building2 },
  { id: 'opportunities', label: 'Cơ hội phù hợp', icon: SearchCheck },
  { id: 'application', label: 'Hồ sơ ứng tuyển', icon: FileCheck2 },
  { id: 'alerts', label: 'Theo dõi & cảnh báo', icon: Bell },
  { id: 'admin', label: 'Policy review', icon: ShieldCheck },
]

export function Shell({ page, onNavigate, children, unreadAlerts = 0 }: { page: Page; onNavigate: (page: Page) => void; children: ReactNode; unreadAlerts?: number }) {
  const [mobileOpen, setMobileOpen] = useState(false)
  const navigate = (next: Page) => { onNavigate(next); setMobileOpen(false) }
  return (
    <div className="app-shell">
      <button className="mobile-menu" aria-label="Mở điều hướng" onClick={() => setMobileOpen(true)}><Menu /></button>
      <AnimatePresence>
        {mobileOpen && <motion.button className="nav-scrim" aria-label="Đóng điều hướng" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} onClick={() => setMobileOpen(false)} />}
      </AnimatePresence>
      <aside className="sidebar" data-open={mobileOpen}>
        <div className="brand-row">
          <div className="brand-mark"><LibraryBig /></div>
          <div><strong>P2B</strong><span>Policy to Business</span></div>
          <button className="sidebar-close" aria-label="Đóng điều hướng" onClick={() => setMobileOpen(false)}><X /></button>
        </div>
        <div className="workspace-chip"><span className="live-dot" />Workspace pilot<strong>GreenTech Demo</strong></div>
        <nav aria-label="Điều hướng chính">
          {navigation.map(({ id, label, icon: Icon }) => (
            <button key={id} className="nav-item" data-active={page === id} onClick={() => navigate(id)}>
              <Icon aria-hidden="true" /><span>{label}</span>{id === 'alerts' && unreadAlerts > 0 && <b>{unreadAlerts}</b>}
            </button>
          ))}
        </nav>
        <div className="sidebar-foot">
          <div className="trust-seal"><ShieldCheck /><span><strong>Human-in-control</strong>AI đề xuất · Bạn quyết định</span></div>
          <p>P2B không thay thế tư vấn pháp lý.</p>
        </div>
      </aside>
      <main className="main-content">
        <header className="topbar"><div><span className="eyebrow">P2B WORKSPACE</span><p>Dữ liệu cập nhật · 18/07/2026</p></div><div className="topbar-actions"><span className="system-state"><span />Hệ thống ổn định</span><button className="avatar" aria-label="Tài khoản GreenTech">GT</button></div></header>
        <motion.div className="page-wrap" key={page} initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: .22 }}>{children}</motion.div>
      </main>
    </div>
  )
}

