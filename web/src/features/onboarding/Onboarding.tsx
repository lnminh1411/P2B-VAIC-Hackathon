import { ArrowRight, FileText, Link2, LockKeyhole, Sparkles, UploadCloud } from 'lucide-react'
import { motion } from 'motion/react'
import { useMemo, useState, type FormEvent } from 'react'

const needs = ['Vốn ưu đãi', 'Thuế', 'R&D', 'Chuyển đổi số', 'Đào tạo', 'Công nghệ xanh', 'Đổi mới sáng tạo']

export function Onboarding({ onSubmit, busy, error }: { onSubmit: (data: { company_name: string; website: string; support_needs: string[]; source_names: string[] }) => void; busy: boolean; error?: string }) {
  const [company, setCompany] = useState('Công ty Cổ phần GreenTech Việt Nam')
  const [website, setWebsite] = useState('https://greentech.example.vn')
  const [selected, setSelected] = useState(['Vốn ưu đãi', 'Công nghệ xanh', 'Đổi mới sáng tạo'])
  const [files, setFiles] = useState<string[]>(['dang-ky-doanh-nghiep.pdf', 'pitch-deck-2026.pdf'])
  const canSubmit = useMemo(() => company.trim().length >= 2 && selected.length > 0, [company, selected])
  const toggle = (need: string) => setSelected(current => current.includes(need) ? current.filter(item => item !== need) : [...current, need])
  const submit = (event: FormEvent) => { event.preventDefault(); if (canSubmit) onSubmit({ company_name: company.trim(), website: website.trim(), support_needs: selected, source_names: files }) }

  return (
    <div className="onboarding-shell">
      <header className="onboarding-header"><div className="brand-mark"><Sparkles /></div><strong>P2B</strong><span>Policy to Business</span><div className="header-trust"><LockKeyhole />Dữ liệu được bảo vệ</div></header>
      <main className="onboarding-grid">
        <motion.section className="onboarding-copy" initial={{ opacity: 0, x: -18 }} animate={{ opacity: 1, x: 0 }}>
          <span className="kicker">COMPANY PROFILING · BƯỚC 1/4</span>
          <h1>Biến hồ sơ doanh nghiệp thành <em>cơ hội hỗ trợ.</em></h1>
          <p>P2B đọc tài liệu, xây Company Passport có nguồn dẫn, rồi đối chiếu hàng trăm điều kiện chính sách.</p>
          <div className="proof-row"><span><b>01</b>Thông tin có dẫn nguồn</span><span><b>02</b>AI không tự quyết định</span><span><b>03</b>Review trước khi xuất</span></div>
          <blockquote><strong>51,3%</strong><span>doanh nghiệp khảo sát chưa biết đến Luật Hỗ trợ DNNVV.</span><cite>PCI 2021 · VCCI</cite></blockquote>
        </motion.section>
        <motion.form className="onboarding-form" onSubmit={submit} initial={{ opacity: 0, y: 18 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: .08 }}>
          <div className="form-heading"><span>Khởi tạo Company Passport</span><small>Khoảng 3–5 phút</small></div>
          <label>Tên doanh nghiệp<input value={company} onChange={event => setCompany(event.target.value)} maxLength={200} required /></label>
          <label>Website doanh nghiệp<div className="input-icon"><Link2 /><input value={website} onChange={event => setWebsite(event.target.value)} type="url" placeholder="https://" /></div></label>
          <fieldset><legend>Bạn đang tìm hỗ trợ gì?</legend><div className="choice-cloud">{needs.map(need => <button type="button" key={need} data-selected={selected.includes(need)} onClick={() => toggle(need)}>{need}</button>)}</div></fieldset>
          <label className="upload-zone"><UploadCloud /><strong>Thêm tài liệu PDF</strong><span>Đăng ký doanh nghiệp, pitch deck, hồ sơ năng lực · tối đa 20 MB/file</span><input type="file" accept="application/pdf,.pdf" multiple onChange={event => setFiles(Array.from(event.target.files ?? []).slice(0, 10).map(file => file.name))} /></label>
          {files.length > 0 && <div className="file-list">{files.map(file => <span key={file}><FileText />{file}</span>)}</div>}
          {error && <p className="inline-error" role="alert">{error}</p>}
          <button className="button primary wide" disabled={!canSubmit || busy}>{busy ? 'Đang phân tích hồ sơ…' : 'Xây Company Passport'}<ArrowRight /></button>
          <p className="form-note">AI chỉ tạo đề xuất. Bạn sẽ kiểm tra từng dữ kiện trước khi matching.</p>
        </motion.form>
      </main>
    </div>
  )
}

