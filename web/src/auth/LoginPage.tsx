import { ArrowRight, Building2, Check, FileSearch, LockKeyhole, Radar, ShieldCheck, Sparkles } from 'lucide-react'

export function LoginPage({ configured, error, onGoogleSignIn }: { configured: boolean; error?: string; onGoogleSignIn: () => void }) {
  return (
    <div className="auth-shell">
      <header className="auth-header">
        <a className="auth-brand" href="/" aria-label="P2B — trang đăng nhập">
          <span className="brand-mark"><Building2 aria-hidden="true" /></span>
          <span><strong>P2B</strong><small>Policy to Business</small></span>
        </a>
        <div className="auth-trust"><ShieldCheck aria-hidden="true" /> Dữ liệu doanh nghiệp được bảo vệ</div>
      </header>

      <main className="auth-main">
        <section className="auth-story" aria-labelledby="auth-title">
          <span className="kicker">AI-NATIVE POLICY INTELLIGENCE</span>
          <h1 id="auth-title">Từ hồ sơ doanh nghiệp đến <em>cơ hội có thể hành động.</em></h1>
          <p>P2B đọc tài liệu, kiểm chứng dữ kiện và đối chiếu điều kiện chính sách—để đội ngũ của bạn tập trung vào quyết định, không phải tìm kiếm.</p>
          <div className="auth-capabilities" aria-label="Năng lực chính của P2B">
            <article><FileSearch aria-hidden="true" /><div><strong>Company Passport</strong><span>Mọi dữ kiện có nguồn, trang và mức tin cậy.</span></div></article>
            <article><Radar aria-hidden="true" /><div><strong>Policy matching</strong><span>Hybrid RAG tìm cơ hội; rule engine kiểm eligibility.</span></div></article>
            <article><Sparkles aria-hidden="true" /><div><strong>Application copilot</strong><span>Checklist, bổ sung evidence và hồ sơ sẵn sàng review.</span></div></article>
          </div>
          <div className="auth-proof"><strong>51,3%</strong><span>doanh nghiệp khảo sát chưa biết đến Luật Hỗ trợ DNNVV.</span><small>PCI 2021 · VCCI</small></div>
        </section>

        <section className="auth-card" aria-labelledby="login-heading">
          <div className="auth-card-icon"><LockKeyhole aria-hidden="true" /></div>
          <span className="kicker">SECURE WORKSPACE</span>
          <h2 id="login-heading">Đăng nhập để bắt đầu Company Passport</h2>
          <p>Mỗi tài khoản Google mở một workspace riêng. Bạn kiểm soát dữ kiện nào được xác nhận và đưa vào hồ sơ.</p>
          <button className="google-button" disabled={!configured} onClick={onGoogleSignIn}>
            <GoogleMark />
            <span>Tiếp tục với Google</span>
            <ArrowRight aria-hidden="true" />
          </button>
          {!configured && <div className="auth-config-warning" role="status">Google Login chưa được cấu hình</div>}
          {error && <div className="auth-error" role="alert">{error}</div>}
          <ul>
            <li><Check aria-hidden="true" />Không cần tạo mật khẩu mới</li>
            <li><Check aria-hidden="true" />Workspace tách biệt theo doanh nghiệp</li>
            <li><Check aria-hidden="true" />AI không tự xác nhận dữ kiện</li>
          </ul>
          <p className="auth-legal">Bằng việc tiếp tục, bạn đồng ý với Điều khoản sử dụng và Chính sách bảo mật của P2B.</p>
        </section>
      </main>
      <footer className="auth-footer"><span>© 2026 P2B</span><span>Human-in-control · Evidence-first · Vietnam-ready</span></footer>
    </div>
  )
}

function GoogleMark() {
  return <svg className="google-mark" viewBox="0 0 24 24" aria-hidden="true"><path fill="#4285F4" d="M21.6 12.23c0-.71-.06-1.4-.18-2.07H12v3.92h5.38a4.6 4.6 0 0 1-2 3.02v2.55h3.24c1.9-1.75 2.98-4.33 2.98-7.42Z"/><path fill="#34A853" d="M12 22c2.7 0 4.97-.9 6.62-2.35l-3.24-2.55c-.9.6-2.05.96-3.38.96-2.61 0-4.82-1.76-5.61-4.13H3.05v2.63A10 10 0 0 0 12 22Z"/><path fill="#FBBC05" d="M6.39 13.93A6.02 6.02 0 0 1 6.07 12c0-.67.12-1.32.32-1.93V7.44H3.05A10 10 0 0 0 2 12c0 1.61.38 3.14 1.05 4.56l3.34-2.63Z"/><path fill="#EA4335" d="M12 5.94c1.47 0 2.78.5 3.81 1.49l2.88-2.88A9.65 9.65 0 0 0 12 2a10 10 0 0 0-8.95 5.44l3.34 2.63C7.18 7.7 9.39 5.94 12 5.94Z"/></svg>
}
