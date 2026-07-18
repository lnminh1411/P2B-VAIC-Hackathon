import { Bell, Building2, FileCheck2, LayoutDashboard, LibraryBig, LogOut, Menu, SearchCheck, ShieldCheck, X } from 'lucide-react'
import { AnimatePresence, motion } from 'motion/react'
import { useState, type ReactNode } from 'react'
import { useAuth } from '../auth/context'
import type { Workspace } from '../lib/types'
import { WorkspaceSwitcher } from './WorkspaceSwitcher'
import { useTranslation } from '../lib/i18n'

export type Page = 'overview' | 'passport' | 'opportunities' | 'application' | 'alerts' | 'admin'

const navigation: Array<{ id: Page; icon: typeof LayoutDashboard }> = [
  { id: 'overview', icon: LayoutDashboard },
  { id: 'passport', icon: Building2 },
  { id: 'opportunities', icon: SearchCheck },
  { id: 'application', icon: FileCheck2 },
  { id: 'alerts', icon: Bell },
  { id: 'admin', icon: ShieldCheck },
]

export function Shell({ page, companyName, workspaces, activeWorkspaceId, onWorkspaceChange, onCreateWorkspace, onNavigate, children, unreadAlerts = 0 }: { page: Page; companyName: string; workspaces: Workspace[]; activeWorkspaceId?: string; onWorkspaceChange: (workspaceId: string) => void; onCreateWorkspace: () => void; onNavigate: (page: Page) => void; children: ReactNode; unreadAlerts?: number }) {
  const { user, signOut } = useAuth()
  const { t, lang, setLang } = useTranslation()
  const [mobileOpen, setMobileOpen] = useState(false)
  const [accountOpen, setAccountOpen] = useState(false)
  const navigate = (next: Page) => { onNavigate(next); setMobileOpen(false) }
  const initials = (user?.name || user?.email || 'P2B').split(/\s+/).slice(0, 2).map(part => part[0]).join('').toUpperCase()
  
  const navLabels = t('navigation')
  const shellLabels = t('shell')

  return (
    <div className="app-shell">
      <button className="mobile-menu" aria-label={shellLabels.open_nav} onClick={() => setMobileOpen(true)}><Menu /></button>
      <AnimatePresence>
        {mobileOpen && <motion.button className="nav-scrim" aria-label={shellLabels.close_nav} initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} onClick={() => setMobileOpen(false)} />}
      </AnimatePresence>
      <aside className="sidebar" data-open={mobileOpen}>
        <div className="brand-row">
          <div className="brand-mark"><LibraryBig /></div>
          <div><strong>P2B</strong><span>{shellLabels.brand_sub}</span></div>
          <button className="sidebar-close" aria-label={shellLabels.close_nav} onClick={() => setMobileOpen(false)}><X /></button>
        </div>
        <WorkspaceSwitcher workspaces={workspaces} activeWorkspaceId={activeWorkspaceId} onChange={onWorkspaceChange} onCreate={onCreateWorkspace} />
        <div className="workspace-chip"><span className="live-dot" />{shellLabels.workspace_label}<strong>{companyName || shellLabels.not_set}</strong></div>
        <nav aria-label="Điều hướng chính">
          {navigation.filter(item => item.id !== 'admin' || user?.isAdmin).map(({ id, icon: Icon }) => (
            <button key={id} className="nav-item" data-active={page === id} onClick={() => navigate(id)}>
              <Icon aria-hidden="true" /><span>{navLabels[id]}</span>{id === 'alerts' && unreadAlerts > 0 && <b>{unreadAlerts}</b>}
            </button>
          ))}
        </nav>
        <div className="sidebar-foot">
          <div className="trust-seal"><ShieldCheck /><span><strong>{shellLabels.trust_seal_title}</strong>{shellLabels.trust_seal_desc}</span></div>
          <p>{shellLabels.disclaimer}</p>
        </div>
      </aside>
      <main className="main-content">
        <header className="topbar">
          <div><span className="eyebrow">{shellLabels.eyebrow}</span><p>{shellLabels.verified_banner}</p></div>
          <div className="topbar-actions">
            <span className="system-state"><span />{shellLabels.system_stable}</span>
            <div className="lang-switcher">
              <select
                value={lang}
                onChange={e => setLang(e.target.value as 'vi' | 'en')}
                className="lang-select"
                aria-label="Chọn ngôn ngữ / Select language"
              >
                <option value="vi">Tiếng Việt</option>
                <option value="en">English</option>
              </select>
            </div>
            <div className="account-menu">
              <button className="avatar" aria-label={`Tài khoản ${user?.name ?? user?.email}`} aria-expanded={accountOpen} onClick={() => setAccountOpen(open => !open)}>
                {user?.avatarUrl ? <img src={user.avatarUrl} alt="" referrerPolicy="no-referrer" /> : initials}
              </button>
              <AnimatePresence>
                {accountOpen && (
                  <motion.div className="account-popover" role="menu" initial={{ opacity: 0, y: -5 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -5 }}>
                    <div><strong>{user?.name}</strong><span>{user?.email}</span></div>
                    <button role="menuitem" onClick={() => void signOut()}><LogOut aria-hidden="true" />{shellLabels.logout}</button>
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          </div>
        </header>
        <motion.div className="page-wrap" key={page} initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: .22 }}>{children}</motion.div>
      </main>
    </div>
  )
}
