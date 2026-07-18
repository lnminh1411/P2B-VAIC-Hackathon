import type { MatchResult } from './types'

export function isRetrievedDocument(result: MatchResult) {
  return result.eligibility?.criteria?.some(criterion => criterion.rule_id.startsWith('document-review-')) ?? false
}
