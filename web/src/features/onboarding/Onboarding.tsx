import { ArrowRight, FileText, Link2, LockKeyhole, Sparkles, UploadCloud } from 'lucide-react'
import { motion } from 'motion/react'
import { useMemo, useState, type FormEvent } from 'react'
import type { CompanyOnboardingInput } from './buildPassportPayload'
import { useTranslation } from '../../lib/i18n'

export function Onboarding({ onSubmit, busy, error }: { onSubmit: (data: CompanyOnboardingInput) => void; busy: boolean; error?: string }) {
  const { t } = useTranslation()
  const onboardingLabels = t('onboarding')
  const needs = onboardingLabels.needs

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
    if (next.length > 10) { setFileError(onboardingLabels.err_max_files); return }
    if (next.some(file => file.size > 20 * 1024 * 1024)) { setFileError(onboardingLabels.err_file_size); return }
    if (next.some(file => file.type !== 'application/pdf' && !file.name.toLowerCase().endsWith('.pdf'))) { setFileError(onboardingLabels.err_file_type); return }
    setFiles(next)
  }

  return (
    <div className="onboarding-shell">
      <header className="onboarding-header"><div className="brand-mark"><Sparkles /></div><strong>P2B</strong><span>Policy to Business</span><div className="header-trust"><LockKeyhole />{onboardingLabels.data_protected}</div></header>
      <main className="onboarding-grid">
        <motion.section className="onboarding-copy" initial={{ opacity: 0, x: -18 }} animate={{ opacity: 1, x: 0 }}>
          <span className="kicker">{onboardingLabels.kicker}</span>
          <h1>{onboardingLabels.h1}</h1>
          <p>{onboardingLabels.p}</p>
          <div className="proof-row"><span><b>01</b>{onboardingLabels.proof_1}</span><span><b>02</b>{onboardingLabels.proof_2}</span><span><b>03</b>{onboardingLabels.proof_3}</span></div>
          <blockquote><strong>51,3%</strong><span>{onboardingLabels.quote_text}</span><cite>{onboardingLabels.quote_author}</cite></blockquote>
        </motion.section>
        <motion.form className="onboarding-form" onSubmit={submit} initial={{ opacity: 0, y: 18 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: .08 }}>
          <div className="form-heading"><span>{onboardingLabels.form_title}</span><small>{onboardingLabels.form_subtitle}</small></div>
          <label>{onboardingLabels.company_name_label}<input value={company} onChange={event => setCompany(event.target.value)} maxLength={200} required /></label>
          <label>{onboardingLabels.website_label}<div className="input-icon"><Link2 /><input value={website} onChange={event => setWebsite(event.target.value)} type="url" placeholder={onboardingLabels.website_placeholder} /></div></label>
          <fieldset><legend>{onboardingLabels.legend}</legend><div className="choice-cloud">{needs.map(need => <button type="button" key={need} data-selected={selected.includes(need)} onClick={() => toggle(need)}>{need}</button>)}</div></fieldset>
          <label className="upload-zone"><UploadCloud /><strong>{onboardingLabels.upload_title}</strong><span>{onboardingLabels.upload_desc}</span><input type="file" accept="application/pdf,.pdf" multiple onChange={event => selectFiles(Array.from(event.target.files ?? []))} /></label>
          {files.length > 0 && <div className="file-list">{files.map(file => <span key={`${file.name}-${file.size}`}><FileText />{file.name}</span>)}</div>}
          {fileError && <p className="inline-error" role="alert">{fileError}</p>}
          {error && <p className="inline-error" role="alert">{error}</p>}
          <button className="button primary wide" disabled={!canSubmit || busy}>{busy ? onboardingLabels.busy_text : onboardingLabels.submit_text}<ArrowRight /></button>
          <p className="form-note">{onboardingLabels.form_note}</p>
        </motion.form>
      </main>
    </div>
  )
}
