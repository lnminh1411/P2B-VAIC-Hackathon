import { describe, expect, it } from 'vitest'
import type { MatchResult } from './types'
import { isRetrievedDocument } from './policy'

describe('isRetrievedDocument', () => {
  it('does not confuse a reviewed rule policy with a document in a hybrid run', () => {
    const policy = {
      retrieval_mode: 'HYBRID_RULE_VECTOR',
      eligibility: { criteria: [{ rule_id: 'reviewed-province-rule' }] },
    } as MatchResult

    expect(isRetrievedDocument(policy)).toBe(false)
  })

  it('recognizes generated document-review criteria', () => {
    const document = {
      retrieval_mode: 'HYBRID_RULE_VECTOR',
      eligibility: { criteria: [{ rule_id: 'document-review-1' }] },
    } as MatchResult

    expect(isRetrievedDocument(document)).toBe(true)
  })
})
