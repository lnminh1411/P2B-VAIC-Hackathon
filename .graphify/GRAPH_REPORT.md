# Graph Report - .  (2026-07-17)

## Corpus Check
- Corpus is ~16,938 words - fits in a single context window. You may not need a graph.

## Summary
- 77 nodes · 123 edges · 8 communities detected
- Extraction: 65% EXTRACTED · 35% INFERRED · 0% AMBIGUOUS · INFERRED: 43 edges (avg confidence: 0.5)
- Token cost: 0 input · 0 output
- Edge kinds: uses: 43 · contains: 38 · inherits: 16 · method: 9 · rationale_for: 9 · calls: 6 · imports_from: 2


## Input Scope
- Requested: auto
- Resolved: committed (source: default-auto)
- Included files: 32 · Candidates: 53
- Excluded: 15 untracked · 5170 ignored · 0 sensitive · 0 missing committed
- Recommendation: Use --scope all or graphify.yaml inputs.corpus for a knowledge-base folder.

## Graph Freshness
- Built from Git commit: `219d453`
- Compare this hash to `git rev-parse HEAD` before trusting freshness-sensitive graph output.
## God Nodes (most connected - your core abstractions)
1. `CompanyPassport` - 16 edges
2. `HybridRetrievalEngine` - 15 edges
3. `PolicyOpportunity` - 14 edges
4. `TestP2BGoldenPath` - 8 edges
5. `Evaluates a single deterministic rule against the CompanyPassport.     Returns (` - 7 edges
6. `Recursively evaluates a RuleGroup using ALL/ANY logic.     Returns (status, grou` - 7 edges
7. `DraftCreateRequest` - 5 edges
8. `DraftStatusUpdateRequest` - 5 edges
9. `PassportUpdateRequest` - 5 edges
10. `RuleOperator` - 5 edges

## Surprising Connections (you probably didn't know these)
- `DraftCreateRequest` --uses--> `HybridRetrievalEngine`  [INFERRED]
  backend/app/main.py → backend/app/engine/retrieval.py
- `DraftStatusUpdateRequest` --uses--> `HybridRetrievalEngine`  [INFERRED]
  backend/app/main.py → backend/app/engine/retrieval.py
- `PassportUpdateRequest` --uses--> `HybridRetrievalEngine`  [INFERRED]
  backend/app/main.py → backend/app/engine/retrieval.py
- `HybridRetrievalEngine` --uses--> `CompanyPassport`  [INFERRED]
  backend/app/engine/retrieval.py → backend/app/schemas/passport.py
- `HybridRetrievalEngine` --uses--> `PolicyOpportunity`  [INFERRED]
  backend/app/engine/retrieval.py → backend/app/schemas/policy.py

## Communities

### Community 0 - "Community 0"
Cohesion: 0.27
Nodes (12): Recursively evaluates a RuleGroup using ALL/ANY logic.     Returns (status, grou, Evaluates a single deterministic rule against the CompanyPassport.     Returns (, Enum, FieldProvenance, Citation, DocumentStatus, GroupLogic, LegalDocument (+4 more)

### Community 2 - "Community 2"
Cohesion: 0.54
Nodes (8): DraftCreateRequest, DraftStatusUpdateRequest, PassportUpdateRequest, BaseModel, CompanyPassport, PolicyOpportunity, Verify that eligibility verifier matches the labeled ground truth for all 3 comp, Verify that document template is filled correctly.

### Community 3 - "Community 3"
Cohesion: 0.33
Nodes (3): cosine_similarity(), Retrieves ranked PolicyOpportunities using BM25, Vector Search, and Metadata Fil, Returns a score in [0.0, 1.0] indicating geographic and basic profile alignment.

### Community 4 - "Community 4"
Cohesion: 0.33
Nodes (1): TestP2BGoldenPath

### Community 5 - "Community 5"
Cohesion: 0.40
Nodes (3): HybridRetrievalEngine, Saves current memory embedding cache to seed directory., Verify that RAG retrieves correct AI program for the AI query.

### Community 6 - "Community 6"
Cohesion: 0.83
Nodes (3): evaluate_rule_group(), evaluate_single_rule(), parse_date()

### Community 7 - "Community 7"
Cohesion: 1.00
Nodes (2): get_db_connection(), init_db()

### Community 10 - "Community 10"
Cohesion: 1.00
Nodes (1): Gets text embedding using local SentenceTransformer.          Prepends 'query: '

## Knowledge Gaps
- **Thin community `Community 4`** (1 nodes): `TestP2BGoldenPath`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 7`** (2 nodes): `get_db_connection()`, `init_db()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 10`** (1 nodes): `Gets text embedding using local SentenceTransformer.          Prepends 'query: '`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `CompanyPassport` connect `Community 2` to `Community 5`, `Community 3`, `Community 10`, `Community 0`, `Community 4`?**
  _High betweenness centrality (0.208) - this node is a cross-community bridge._
- **Why does `HybridRetrievalEngine` connect `Community 5` to `Community 2`, `Community 3`, `Community 10`, `Community 4`?**
  _High betweenness centrality (0.126) - this node is a cross-community bridge._
- **Why does `PolicyOpportunity` connect `Community 2` to `Community 5`, `Community 3`, `Community 10`, `Community 0`, `Community 4`?**
  _High betweenness centrality (0.093) - this node is a cross-community bridge._
- **Are the 14 inferred relationships involving `CompanyPassport` (e.g. with `DraftCreateRequest` and `DraftStatusUpdateRequest`) actually correct?**
  _`CompanyPassport` has 14 INFERRED edges - model-reasoned connections that need verification._
- **Are the 9 inferred relationships involving `HybridRetrievalEngine` (e.g. with `DraftCreateRequest` and `DraftStatusUpdateRequest`) actually correct?**
  _`HybridRetrievalEngine` has 9 INFERRED edges - model-reasoned connections that need verification._
- **Are the 12 inferred relationships involving `PolicyOpportunity` (e.g. with `DraftCreateRequest` and `DraftStatusUpdateRequest`) actually correct?**
  _`PolicyOpportunity` has 12 INFERRED edges - model-reasoned connections that need verification._
- **Are the 3 inferred relationships involving `TestP2BGoldenPath` (e.g. with `HybridRetrievalEngine` and `CompanyPassport`) actually correct?**
  _`TestP2BGoldenPath` has 3 INFERRED edges - model-reasoned connections that need verification._