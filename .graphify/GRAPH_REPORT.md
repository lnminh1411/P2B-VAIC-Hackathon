# Graph Report - .  (2026-07-17)

## Corpus Check
- Corpus is ~26,736 words - fits in a single context window. You may not need a graph.

## Summary
- 168 nodes · 298 edges · 10 communities detected
- Extraction: 63% EXTRACTED · 37% INFERRED · 0% AMBIGUOUS · INFERRED: 109 edges (avg confidence: 0.5)
- Token cost: 0 input · 0 output
- Edge kinds: uses: 109 · contains: 79 · rationale_for: 42 · inherits: 25 · calls: 20 · method: 20 · imports_from: 3


## Input Scope
- Requested: auto
- Resolved: committed (source: default-auto)
- Included files: 41 · Candidates: 93
- Excluded: 102 untracked · 5172 ignored · 0 sensitive · 0 missing committed
- Recommendation: Use --scope all or graphify.yaml inputs.corpus for a knowledge-base folder.

## Graph Freshness
- Built from Git commit: `9a2a7a9`
- Compare this hash to `git rev-parse HEAD` before trusting freshness-sensitive graph output.
## God Nodes (most connected - your core abstractions)
1. `PolicyOpportunity` - 34 edges
2. `CompanyPassport` - 32 edges
3. `HybridRetrievalEngine` - 22 edges
4. `RuleOperator` - 10 edges
5. `GroupLogic` - 10 edges
6. `ExtractedPolicyOpportunity` - 9 edges
7. `Rule` - 9 edges
8. `RuleGroup` - 9 edges
9. `TestP2BGoldenPath` - 8 edges
10. `Evaluates a single deterministic rule against the CompanyPassport.     Returns` - 7 edges

## Surprising Connections (you probably didn't know these)
- `Recursively evaluates a RuleGroup using ALL/ANY logic.     Returns (status, gro` --uses--> `CompanyPassport`  [INFERRED]
  backend/app/engine/rule_evaluator.py → backend/app/schemas/passport.py
- `Evaluates a single deterministic rule against the CompanyPassport.     Returns` --uses--> `CompanyPassport`  [INFERRED]
  backend/app/engine/rule_evaluator.py → backend/app/schemas/passport.py
- `ExtractedPolicyOpportunity` --uses--> `PolicyOpportunity`  [INFERRED]
  backend/app/pipeline/search_crawler.py → backend/app/schemas/policy.py
- `Programmatically builds a matching RuleGroup mapping extracted criteria fields t` --uses--> `PolicyOpportunity`  [INFERRED]
  backend/app/pipeline/search_crawler.py → backend/app/schemas/policy.py
- `Core search worker: fetches/generates decrees, calls Gemini to structure PolicyO` --uses--> `PolicyOpportunity`  [INFERRED]
  backend/app/pipeline/search_crawler.py → backend/app/schemas/policy.py

## Communities

### Community 0 - "Community 0"
Cohesion: 0.15
Nodes (30): DraftCreateRequest, DraftStatusUpdateRequest, LoginRequest, PassportUpdateRequest, PasswordChangeRequest, SignupRequest, UserModeUpdateRequest, BaseModel (+22 more)

### Community 1 - "Community 1"
Cohesion: 0.06
Nodes (13): daily_cron_worker(), dynamic_cors_middleware(), extract_document(), extract_multiple_documents(), is_allowed_origin(), Intelligent multi-document sorting and extraction agent (Goal 7), Verifiable SHA-256 diff hash policy sync engine (Goal 3), Verifiable SHA-256 diff hash policy sync engine (Goal 3) (+5 more)

### Community 2 - "Community 2"
Cohesion: 0.23
Nodes (21): Recursively evaluates a RuleGroup using ALL/ANY logic.     Returns (status, gro, Evaluates a single deterministic rule against the CompanyPassport.     Returns, Enum, construct_policy_rules(), ExtractedPolicyOpportunity, fetch_external_decrees(), get_fallback_mock_decree(), Programmatically builds a matching RuleGroup mapping extracted criteria fields t (+13 more)

### Community 3 - "Community 3"
Cohesion: 0.15
Nodes (14): call_gemini_extraction(), extract_year_from_text(), ExtractedPassport, ExtractedPersonal, FactEvidence, get_embed_model(), PersonalEvidence, rank_documents_for_field() (+6 more)

### Community 4 - "Community 4"
Cohesion: 0.15
Nodes (6): Verify gated review blocks un-MET approvals and permits MET ones, Verify CORS whitelist enforcement, Run full golden path of registration, policy search, eligibility check, gated ap, Test signup, login, get profile, change password, and logout, Test avatar uploading and account deletion, TestIntegrationP2B

### Community 5 - "Community 5"
Cohesion: 0.33
Nodes (1): TestP2BGoldenPath

### Community 6 - "Community 6"
Cohesion: 0.40
Nodes (1): TestActiveSearchAndCache

### Community 7 - "Community 7"
Cohesion: 0.83
Nodes (3): evaluate_rule_group(), evaluate_single_rule(), parse_date()

### Community 9 - "Community 9"
Cohesion: 1.00
Nodes (2): get_db_connection(), init_db()

### Community 10 - "Community 10"
Cohesion: 0.67
Nodes (2): convert_to_markdown_local(), Converts a document (docx, xlsx, doc, csv, pptx, html, etc.) to markdown using l

## Knowledge Gaps
- **18 isolated node(s):** `Single file upload extraction endpoint (Goal 2)`, `Intelligent multi-document sorting and extraction agent (Goal 7)`, `Verifiable SHA-256 diff hash policy sync engine (Goal 3)`, `Finds the most recent year (e.g. 2024, 2025) mentioned in the document text.`, `Ranks documents based on semantic relevance to the query and date recency,     r` (+13 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 5`** (1 nodes): `TestP2BGoldenPath`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 6`** (1 nodes): `TestActiveSearchAndCache`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 9`** (2 nodes): `get_db_connection()`, `init_db()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 10`** (2 nodes): `convert_to_markdown_local()`, `Converts a document (docx, xlsx, doc, csv, pptx, html, etc.) to markdown using l`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `PolicyOpportunity` connect `Community 0` to `Community 2`, `Community 5`?**
  _High betweenness centrality (0.189) - this node is a cross-community bridge._
- **Why does `CompanyPassport` connect `Community 0` to `Community 2`, `Community 6`, `Community 5`?**
  _High betweenness centrality (0.163) - this node is a cross-community bridge._
- **Are the 32 inferred relationships involving `PolicyOpportunity` (e.g. with `DraftCreateRequest` and `DraftStatusUpdateRequest`) actually correct?**
  _`PolicyOpportunity` has 32 INFERRED edges - model-reasoned connections that need verification._
- **Are the 30 inferred relationships involving `CompanyPassport` (e.g. with `DraftCreateRequest` and `DraftStatusUpdateRequest`) actually correct?**
  _`CompanyPassport` has 30 INFERRED edges - model-reasoned connections that need verification._
- **Are the 14 inferred relationships involving `HybridRetrievalEngine` (e.g. with `DraftCreateRequest` and `DraftStatusUpdateRequest`) actually correct?**
  _`HybridRetrievalEngine` has 14 INFERRED edges - model-reasoned connections that need verification._
- **Are the 7 inferred relationships involving `RuleOperator` (e.g. with `Recursively evaluates a RuleGroup using ALL/ANY logic.     Returns (status, gro` and `Evaluates a single deterministic rule against the CompanyPassport.     Returns`) actually correct?**
  _`RuleOperator` has 7 INFERRED edges - model-reasoned connections that need verification._
- **What connects `Single file upload extraction endpoint (Goal 2)`, `Intelligent multi-document sorting and extraction agent (Goal 7)`, `Verifiable SHA-256 diff hash policy sync engine (Goal 3)` to the rest of the system?**
  _18 weakly-connected nodes found - possible documentation gaps or missing edges._