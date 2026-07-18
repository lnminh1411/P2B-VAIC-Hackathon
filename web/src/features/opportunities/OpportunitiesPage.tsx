import { ArrowRight, Banknote, CalendarClock, Check, ExternalLink, FileSearch, MapPin, Search, Sparkles, X } from 'lucide-react'
import { AnimatePresence, motion } from 'motion/react'
import { useState } from 'react'
import { StatusBadge } from '../../components/StatusBadge'
import { displayValue, formatDate } from '../../lib/format'
import { isRetrievedDocument } from '../../lib/policy'
import type { EnrichmentRun, MatchResult, MatchRun } from '../../lib/types'
import { useTranslation } from '../../lib/i18n'

export function OpportunitiesPage({ run, onMatch, matching, selected, onSelect, onPrepare, enrichment, onEnrich, onAcceptEvidence, busy, error }: {
  run?: MatchRun; onMatch: () => void; matching: boolean; selected?: MatchResult; onSelect: (result?: MatchResult) => void; onPrepare: (result: MatchResult) => void; enrichment?: EnrichmentRun; onEnrich: (policyId: string) => void; onAcceptEvidence: (id: string) => void; busy: boolean; error?: string
}) {
  const { t } = useTranslation()
  const o = t('opportunities')
  const [query, setQuery] = useState('')
  const runResults = run?.results ?? []
  const results = runResults.filter(result => (result.title + result.benefit + result.agency).toLowerCase().includes(query.toLowerCase()))
  const retrievalMode = runResults[0]?.retrieval_mode ?? 'NO_PUBLISHED_CORPUS'
  
  if (!run) return <EmptyMatching onMatch={onMatch} matching={matching} error={error} />
  
  return (
    <>
      <section className="page-heading split-heading">
        <div>
          <span className="kicker">POLICY MATCHING</span>
          <h1>{o.h1}<em>{o.h1_em}</em></h1>
        </div>
        <button className="button secondary" onClick={onMatch} disabled={matching}><Sparkles />{o.rerun_btn}</button>
      </section>
      
      {error && <p className="inline-error" role="alert">{error}</p>}
      
      <div className="search-toolbar">
        <div className="search-input">
          <Search />
          <input value={query} onChange={event => setQuery(event.target.value)} placeholder={o.search_placeholder} aria-label="Tìm cơ hội" />
        </div>
        <span><b>{results.length}</b>{o.results_suffix}</span>
      </div>
      
      <div className="opportunity-layout">
        <div className="opportunity-list">
          {results.length === 0 ? (
            <div className="page-state">
              <FileSearch />
              <strong>{o.empty_title}</strong>
              <span>{o.empty_desc}</span>
            </div>
          ) : (
            results.map((result, index) => (
              <motion.article 
                className="opportunity-card" 
                key={result.policy_id} 
                initial={{ opacity: 0, y: 10 }} 
                animate={{ opacity: 1, y: 0 }} 
                transition={{ delay: index * .05 }}
              >
                <div className="match-score">
                  <strong>{result.score}</strong>
                  <span>{o.match_score_label}</span>
                </div>
                <div className="opportunity-main">
                  <div className="opportunity-head">
                    <div>
                      <span>{result.agency}</span>
                      <h2>{result.title}</h2>
                    </div>
                    <StatusBadge status={result.eligibility.status} />
                  </div>
                  <p>{result.benefit}</p>
                  <div className="policy-meta">
                    <span><Banknote />{result.benefit_amount || o.legal_document}</span>
                    <span><CalendarClock />{deadlineLabel(result.deadline, o.no_deadline, o.deadline_prefix)}</span>
                    <span><MapPin />{o.vietnam}</span>
                  </div>
                  <div className="reason-row">
                    {result.ranking_reasons.slice(0, 2).map(reason => (
                      <span key={reason}><Check />{reason}</span>
                    ))}
                  </div>
                </div>
                <button className="card-open" aria-label={`Mở ${result.title}`} onClick={() => onSelect(result)}>
                  <ArrowRight />
                </button>
              </motion.article>
            ))
          )}
        </div>
        

      </div>
      
      <AnimatePresence>
        {selected && (
          <PolicyDrawer 
            result={selected} 
            onClose={() => onSelect(undefined)} 
            onPrepare={() => onPrepare(selected)} 
            enrichment={enrichment} 
            onEnrich={() => onEnrich(selected.policy_id)} 
            onAccept={onAcceptEvidence} 
            busy={busy} 
          />
        )}
      </AnimatePresence>
    </>
  )
}

function deadlineLabel(deadline: string, noDeadlineText: string, prefix: string) {
  return !deadline || deadline.startsWith('0001-') ? noDeadlineText : `${prefix}${formatDate(deadline)}`
}

function EmptyMatching({ onMatch, matching, error }: { onMatch: () => void; matching: boolean; error?: string }) {
  const { t } = useTranslation()
  const o = t('opportunities')
  return (
    <div className="matching-empty">
      <h1>{o.empty_matching_h1}</h1>
      <p>{o.empty_matching_desc}</p>
      {error && <p className="inline-error" role="alert">{error}</p>}
      <button className="button primary" onClick={onMatch} disabled={matching}>
        {matching ? o.busy_matching : o.start_matching}
        <Search />
      </button>
    </div>
  )
}

function PolicyDrawer({ result, onClose, onPrepare, enrichment, onEnrich, onAccept, busy }: { result: MatchResult; onClose: () => void; onPrepare: () => void; enrichment?: EnrichmentRun; onEnrich: () => void; onAccept: (id: string) => void; busy: boolean }) {
  const { t } = useTranslation()
  const o = t('opportunities')
  const missing = result.eligibility.criteria.filter(item => item.status === 'MISSING_INFO')
  const retrieved = isRetrievedDocument(result)

  return (
    <>
      <motion.button className="drawer-scrim" aria-label="Đóng chi tiết" onClick={onClose} initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} />
      <motion.aside className="policy-drawer" initial={{ x: '100%' }} animate={{ x: 0 }} exit={{ x: '100%' }} transition={{ type: 'spring', damping: 30, stiffness: 280 }}>
        <header>
          <div>
            <span>POLICY DETAIL · v{result.policy_version}</span>
            <h2>{result.title}</h2>
            <p>{result.agency}</p>
            {result.source_url && <a href={result.source_url} target="_blank" rel="noreferrer">{o.open_source}<ExternalLink /></a>}
          </div>
          <button aria-label="Đóng" onClick={onClose}><X /></button>
        </header>
        
        <section className="benefit-callout">
          <Banknote />
          <div>
            <span>{o.benefit_section}</span>
            <strong>{result.benefit_amount}</strong>
            <p>{result.benefit}</p>
          </div>
        </section>
        
        <section>
          <div className="drawer-section-title">
            <h3>{retrieved ? o.criteria_review_title : o.eligibility_check}</h3>
            <StatusBadge status={result.eligibility.status} />
          </div>
          <div className="criteria-list">
            {result.eligibility.criteria.length === 0 ? (
              <p>{o.criteria_empty}</p>
            ) : (
              result.eligibility.criteria.map(criterion => (
                <article key={criterion.rule_id}>
                  <StatusBadge status={criterion.status} />
                  <div>
                    <strong>{criterion.description}</strong>
                    <p>{retrieved ? o.criteria_manual_review : <>{o.observed_prefix}{displayValue(criterion.observed)} {o.condition_prefix} {criterion.operator} {displayValue(criterion.expected)}</>}</p>
                    <a href={criterion.citation.url} target="_blank" rel="noreferrer">{criterion.citation.source_name}<ExternalLink /></a>
                  </div>
                </article>
              ))
            )}
          </div>
        </section>
        
        {missing.length > 0 && !retrieved && (
          <section className="missing-section">
            <div className="drawer-section-title">
              <h3>{o.missing_data}</h3>
              <span>{missing.length}{o.items_suffix}</span>
            </div>
            <p>{o.agent_search_desc}</p>
            <button className="button secondary wide" onClick={onEnrich} disabled={busy}><Search />{o.find_evidence_btn}</button>
            {enrichment && (
              <div className="enrichment-list">
                {enrichment.candidates.map(candidate => (
                  <article key={candidate.id}>
                    <div>
                      <strong>{candidate.label}</strong>
                      <span>{displayValue(candidate.value)}</span>
                      <small>{candidate.warning}</small>
                    </div>
                    <button disabled={candidate.status === 'ACCEPTED' || busy} onClick={() => onAccept(candidate.id)}>
                      {candidate.status === 'ACCEPTED' ? o.evidence_accepted : o.evidence_accept}
                    </button>
                  </article>
                ))}
              </div>
            )}
          </section>
        )}
        
        <footer>
          <span>{retrieved && result.template_ready ? o.template_working : result.template_ready ? o.template_available : o.template_missing}</span>
          <button className="button primary" onClick={onPrepare} disabled={!result.template_ready}>{o.prepare_btn}<ArrowRight /></button>
        </footer>
      </motion.aside>
    </>
  )
}
