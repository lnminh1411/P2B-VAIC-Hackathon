export type FieldStatus = 'MISSING' | 'EXTRACTED' | 'NEEDS_REVIEW' | 'CONFIRMED' | 'CONFLICTED' | 'STALE'

export interface Evidence {
  source_id: string
  source_name: string
  url?: string
  page?: number
  quote: string
  content_hash: string
  observed_at: string
}

export interface PassportField {
  key: string
  label: string
  value?: unknown
  data_type: string
  status: FieldStatus
  confidence: number
  evidence: Evidence[]
}

export interface Passport {
  id: string
  company_name: string
  website?: string
  support_needs: string[]
  version: number
  fields: Record<string, PassportField>
  updated_at: string
}

export interface Candidate {
  id: string
  field_key: string
  value: unknown
  data_type: string
  confidence: number
  evidence: Evidence
  status: string
}

export interface CriterionResult {
  rule_id: string
  field_key: string
  description: string
  status: 'MET' | 'NOT_MET' | 'MISSING_INFO'
  observed?: unknown
  expected?: unknown
  operator: string
  evidence: Evidence[]
  citation: Evidence
  required: boolean
}

export interface MatchResult {
  policy_id: string
  policy_version: number
  title: string
  agency: string
  benefit: string
  benefit_amount: string
  deadline: string
  score: number
  eligibility: { status: 'MET' | 'NOT_MET' | 'MISSING_INFO'; criteria: CriterionResult[] }
  ranking_reasons: string[]
  template_ready: boolean
  retrieval_mode: string
}

export interface MatchRun { id: string; passport_version: number; created_at: string; results: MatchResult[] }

export interface EnrichmentCandidate {
  id: string
  field_key: string
  label: string
  value: unknown
  confidence: number
  evidence: Evidence
  status: string
  warning: string
}

export interface EnrichmentRun { id: string; policy_id: string; status: string; candidates: EnrichmentCandidate[]; created_at: string }

export interface ChecklistItem {
  id: string
  title: string
  description: string
  required: boolean
  status: string
  field_keys: string[]
  evidence_source?: string
}

export interface Checklist { id: string; policy_id: string; policy_version: number; version: number; items: ChecklistItem[]; updated_at: string }

export interface Application {
  id: string
  checklist_id: string
  policy_id: string
  passport_version: number
  policy_version: number
  template_version: number
  version: number
  status: string
  sections: Record<string, string>
  blocking_reasons: string[] | null
}

export interface Alert { id: string; type: string; title: string; message: string; policy_id?: string; severity: string; read: boolean; occurred_at: string }
