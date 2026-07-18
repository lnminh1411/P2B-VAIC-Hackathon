# P2B Glossary

This document defines the ubiquitous language and core domain terms used throughout the P2B project.

## Core Terms

### Decree (Nghị định)
A formal legal document issued by the Government of Vietnam. Decrees contain detailed regulations, guidelines, and policies supporting specific activities (e.g., business incentives, tax exemptions, credit programs). Decrees are versioned and can have statuses like *In effect* (Còn hiệu lực) or *Partially expired* (Hết hiệu lực một phần).

### Company Passport
A consolidated digital profile of a company containing structured business facts (e.g., legal name, website, sector, revenue, employee count). Each fact in a Company Passport is backed by at least one piece of verifiable evidence. Company Passports are versioned to preserve historical state.

### Policy Version
A structured representation of a specific grant, benefit, or support program. It includes eligibility rules (e.g., sector matches, numeric thresholds) and checklist templates. Policy Versions are linked to raw *Decree* source documents via version IDs.

### Document Chunk
A semantic segment of a legal document (typically split at the Article/Điều level) containing raw markdown text, full-text search vectors, and vector embeddings. Chunks are used for semantic retrieval and context grounding.

### Eligibility Rule
A deterministic criteria check (e.g., `EQ`, `IN`, `CONTAINS`, `GT`, `GTE`, `LT`, `LTE`, `EXISTS`, `DATE_BEFORE`, `DATE_AFTER`) mapped to a specific *Company Passport* field. Rules evaluate to `MET`, `NOT_MET`, or `MISSING_INFO`.

### Crawler (Trình quét)
An automated background worker that queries the legal corpus (VBPL registry) for new or modified decrees. It identifies updates to decree text, metadata, or lifecycle states (e.g., transitioning to *Expired*).

### Alert (Cảnh báo)
A workspace-specific notification generated when the Crawler identifies a change relevant to that company's profile (e.g., a matching policy's deadline is modified, or a decree backing verified evidence goes stale/expires).

### Watchlist (Danh sách theo dõi)
A set of active monitoring configurations for a workspace that filters and routes newly generated crawler alerts (e.g., New Policies, Deadline Changes, Stale Evidence, Upcoming Deadlines).
