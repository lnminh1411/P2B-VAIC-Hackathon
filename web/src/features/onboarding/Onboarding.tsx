import { ArrowRight, FileText, Link2, LockKeyhole, Sparkles, UploadCloud } from 'lucide-react'
import { motion } from 'motion/react'
import { useMemo, useState, type FormEvent } from 'react'
import type { CompanyOnboardingInput } from './buildPassportPayload'

const needs = ['Vốn ưu đãi', 'Thuế', 'R&D', 'Chuyển đổi số', 'Đào tạo', 'Công nghệ xanh', 'Đổi mới sáng tạo']

export function Onboarding({ onSubmit, busy, error }: { onSubmit: (data: CompanyOnboardingInput) => void; busy: boolean; error?: string }) {
  const [company, setCompany] = useState('')
  const [website, setWebsite] = useState('')
  const [selected, setSelected] = useState<string[]>([])
  const [files, setFiles] = useState<File[]>([])
  const [fileError, setFileError] = useState<string>()
  const canSubmit = useMemo(() => company.trim().length >= 2 && selected.length > 0, [company, selected])
  const toggle = (need: string) => setSelected(current => current.includes(need) ? current.filter(item => item !== need) : [...current, need])
  const submit = (event: FormEvent) => { event.preventDefault(); if (canSubmit) onSubmit({ company_name: company.trim(), website: website.trim(), support_needs: selected, files }) }
  const selectFiles = (next: File[]) => {
    setFileError(undefined)
    if (next.length > 10) { setFileError('Tối đa 10 file PDF.'); return }
    if (next.some(file => file.size > 20 * 1024 * 1024)) { setFileError('Mỗi file PDF phải nhỏ hơn hoặc bằng 20 MB.'); return }
    if (next.some(file => file.type !== 'application/pdf' && !file.name.toLowerCase().endsWith('.pdf'))) { setFileError('Chỉ chấp nhận file PDF.'); return }
    setFiles(next)
  }

  return (
    <div className="onboarding-shell">
      <header className="onboarding-header"><div className="brand-mark"><Sparkles /></div><strong>P2B</strong><span>Policy to Business</span><div className="header-trust"><LockKeyhole />Dữ liệu được bảo vệ</div></header>
      <main className="onboarding-grid">
        <motion.section className="onboarding-copy" initial={{ opacity: 0, x: -18 }} animate={{ opacity: 1, x: 0 }}>
          <span className="kicker">COMPANY PROFILING · BƯỚC 1/4</span>
          <h1>Biến hồ sơ doanh nghiệp thành <em>cơ hội hỗ trợ.</em></h1>
          <p>P2B tạo Company Passport từ dữ liệu bạn cung cấp. Dữ kiện tự động chỉ xuất hiện khi có nguồn và evidence kiểm chứng được.</p>
          <div className="proof-row"><span><b>01</b>Thông tin có dẫn nguồn</span><span><b>02</b>AI không tự quyết định</span><span><b>03</b>Review trước khi xuất</span></div>
          <blockquote><strong>51,3%</strong><span>doanh nghiệp khảo sát chưa biết đến Luật Hỗ trợ DNNVV.</span><cite>PCI 2021 · VCCI</cite></blockquote>
        </motion.section>
        <motion.form className="onboarding-form" onSubmit={submit} initial={{ opacity: 0, y: 18 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: .08 }}>
          <div className="form-heading"><span>Khởi tạo Company Passport</span><small>Bước thiết lập ban đầu</small></div>
          <label>Tên doanh nghiệp<input value={company} onChange={event => setCompany(event.target.value)} maxLength={200} required /></label>
          <label>Website doanh nghiệp<div className="input-icon"><Link2 /><input value={website} onChange={event => setWebsite(event.target.value)} type="url" placeholder="https://" /></div></label>
          <fieldset><legend>Bạn đang tìm hỗ trợ gì?</legend><div className="choice-cloud">{needs.map(need => <button type="button" key={need} data-selected={selected.includes(need)} onClick={() => toggle(need)}>{need}</button>)}</div></fieldset>
          <label className="upload-zone"><UploadCloud /><strong>Thêm tài liệu PDF</strong><span>Đăng ký doanh nghiệp, pitch deck, hồ sơ năng lực · tối đa 10 file, 20 MB/file</span><input type="file" accept="application/pdf,.pdf" multiple onChange={event => selectFiles(Array.from(event.target.files ?? []))} /></label>
          {files.length > 0 && <div className="file-list">{files.map(file => <span key={`${file.name}-${file.size}`}><FileText />{file.name}</span>)}</div>}
          {fileError && <p className="inline-error" role="alert">{fileError}</p>}
          {error && <p className="inline-error" role="alert">{error}</p>}
          <button className="button primary wide" disabled={!canSubmit || busy}>{busy ? 'AI đang đọc và kiểm chứng tài liệu…' : 'Xây Company Passport'}<ArrowRight /></button>
          <p className="form-note">PDF được lưu riêng tư, chuyển thành Markdown bằng MarkItDown rồi Gemini trích xuất dữ kiện có dẫn nguồn. Website chưa được crawl tự động.</p>
        </motion.form>
      </main>
    </div>
  )
}
