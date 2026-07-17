# P2B Implementation Plan v3 (Compressed)

**40h. July 17 14:00 → July 19 06:00 (GMT+7)**

---

## Golden Path (one demo flow, no extras)

```
Upload docs + website → Company Passport (provenance per field)
→ Hybrid RAG retrieval → Ranked PolicyOpportunities
→ Select one → Deterministic eligibility: MET / NOT_MET / MISSING_INFO
→ Click result → company evidence + policy citation side-by-side
→ MISSING_INFO → clarification questions inline
→ Checklist missing docs → Human review (approve/reject) → Export .docx
```

---

## Core Architecture

### Eligibility = Deterministic Rules, Not LLM

```
Legal text → LLM extracts candidate rules (offline) → Human validates → Rule JSON
Runtime: Rule engine evaluates Company Passport → LLM explains result
```

Rule schema:
```json
{"criterion_id":"str", "field":"str", "operator":"LTE|GTE|EQ|NEQ|IN|CONTAINS|DATE_BEFORE|DATE_AFTER",
 "expected_value":"any", "required":true, "citation":{"document_id":"str","article":"str","quote":"str"}}
```

Compound rules:
```json
{"criterion_group_id":"sme_eligibility", "logic":"ALL|ANY", "rules":[]}
```
Only `ALL` and `ANY`. Conditions that can't be expressed → `HUMAN_REVIEW` or `MISSING_INFO`. Never let LLM decide.

Tri-state: evidence confirms → `MET`. Evidence denies → `NOT_MET`. No data → `MISSING_INFO`. Never infer `NOT_MET` from absence.

### Company Passport = Provenance Per Field

```json
{"employee_count": {"value":18, "source_type":"PITCH_DECK", "source_uri":"pitch.pdf",
 "source_location":"page 8", "evidence_quote":"18 full-time...", "observed_at":"2026-07-17T16:30+07",
 "confidence":"HIGH", "status":"EXTRACTED", "conflicts":[]}}
```

Statuses: `EXTRACTED`, `USER_PROVIDED`, `USER_CONFIRMED`, `CONFLICTED`, `MISSING`.
Conflict: two sources disagree → mark `CONFLICTED` → block eligibility on that field → user resolves.
Confidence = evidence quality (HIGH: official docs/user-confirmed, MEDIUM: website/pitch deck). No LLM self-reported scores.

### Data Model

```
LegalDocument: id, title, issuing_body, source_url,
               issued_at, effective_from, effective_to, last_verified_at,
               status(CURRENT|EXPIRED|SUPERSEDED), content_hash, chunks

PolicyOpportunity: id, title, benefits, target_companies, geography, deadline,
                   required_documents, eligibility_rules[], source_legal_documents[]
```

6-10 curated PolicyOpportunities. 15-30 LegalDocuments as corpus.

### Hybrid RAG

```
Online:  Company Passport → metadata filter → FTS5/BM25 + vector → score fusion → optional LLM rerank → LLM explanation
Offline: Company Passport → metadata filter → FTS5/BM25 + vector → score fusion → deterministic explanation template
final_score = 0.4×bm25 + 0.4×vector + 0.2×metadata
```

Embedding: `multilingual-e5-base`. Locked. All embeddings cached pre-demo.
LLM reranking = **stretch goal**. Deterministic fusion sufficient for 6-10 opportunities.

### State Machine (SQLite status field, no LangGraph)

```
DRAFT → PENDING_REVIEW → APPROVED → GENERATED
                      ↘ REJECTED (comments) → DRAFT
```

Draft download: `/api/v1/drafts/{id}/download` (prototype — no auth; production uses signed URLs). No public `/static/`.

### Policy Change Detection (stretch)

Store `content_hash` per doc. On-demand ingestion/diff command compares hash (not scheduled — APScheduler is cut). Extract `deadline` at ingestion. Seed one pre-modified doc for live alert demo. Continuous scheduling = roadmap.

### Clarification Loop

`MISSING_INFO` → template-based question from missing field name (e.g., `"Vốn điều lệ hiện tại là bao nhiêu?"`) → rendered inline in UI as editable field. No LLM in critical path for clarification wording.

---

## Scope Cut (roadmap/slide only)

~~Telegram~~ ~~Webhooks~~ ~~APScheduler~~ ~~Cron UI~~ ~~LangGraph~~ ~~2nd template~~ ~~Multiple demo flows~~ ~~Complex animations~~

---

## Golden Dataset

- 1 primary + 2 secondary Company Passports
- 6-10 PolicyOpportunities (human-verified)
- 15-25 eligibility criteria with ground truth
- 5-10 retrieval queries with expected results
- Min 1× MET, 1× NOT_MET, 1× MISSING_INFO, 1× conflict

## Release Gates (never cut)

- Correct policy top-3 for 100% golden queries
- Eligibility = ground truth for 100% demo criteria
- 100% conclusions have citation
- 100% evaluated fields have evidence
- Missing data → `MISSING_INFO`, never `NOT_MET`
- No hallucinated .docx fields
- Golden path passes 5× consecutive
- Demo works offline (cached)

## Minimum Tests (never cut)

Rule engine operators · tri-state · provenance preservation · retrieval golden · citation presence · template no-hallucinate · e2e smoke

---

## Timeline

| Hours | Work | Gate |
|---|---|---|
| **0-4** | Lock schemas, curate 6-10 opportunities, label ground truth, choose demo persona, lock `multilingual-e5-base` | Team knows exact expected demo output |
| **4-10** | CLI vertical slice: parse docs → Passport w/ provenance → retrieve 1 opportunity → run 1 rule → return evidence+citation | One slice e2e before any UI |
| **10-18** | **Must-have:** cached corpus, metadata filter, FTS5+vector fusion, operators used in golden dataset, tri-state, checklist, golden tests. **Stretch:** LLM reranking, change detection/alerts, unused operators, LLM-generated clarification wording | Release metrics pass, no conclusion without citation |
| **18-24** | HITL: review/approve/reject, 1 .docx template fill (only evidence-backed fields), audit trail, download endpoint | Exported file = no hallucination, reflects review |
| **24-29** | React UI: 3 screens only — (1) Passport+provenance (2) Ranked opportunities+alerts (3) Eligibility+checklist+review | Golden path from UI, no manual DB/scripts |
| **29-32** | Demo freeze: 5× golden run, cached mode test, latency check, backup video, pitch finalize | Stable demo + fallback plan |
| **32-40** | Buffer+sleep. Bug fixes only. No refactor/new features/provider changes | — |

> [!CAUTION]
> Timeline slips → cut in order: change alerts → LLM reranking → secondary profiles in UI → UI polish. Never cut tests or release gates.

> [!IMPORTANT]
> Release metrics are validated on **golden demo set only** — do not claim production accuracy.


