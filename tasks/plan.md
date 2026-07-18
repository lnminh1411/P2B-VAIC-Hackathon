# P2B MVP implementation plan

1. Foundation and security boundaries.
2. Company Passport extraction/review slice.
3. Versioned policy corpus and deterministic matching slice.
4. Enrichment, checklist and application preparation slice.
5. Black-purple responsive UI, deployment and browser verification.

Each slice must leave `make test`, `make lint` and `make build` green.

## Cached application drafts and LLM templates

### Architecture decisions

- Add workspace-scoped PostgreSQL repositories without changing existing immutable core tables.
- Convert uploaded PDF/DOCX/TXT templates to bounded plain text with existing MarkItDown runtime.
- Reuse Passport's `GEMINI_API_KEY`, `GEMINI_MODEL` and Gemini endpoint behavior.
- Generate structured editable sections; autosave them with optimistic version checks.
- Restore latest draft and selected template through TanStack Query.

### Ordered slices

1. Add template/draft contracts, additive migration, repository and tests.
2. Add Gemini application generation and template upload/list APIs.
3. Add template picker, generation flow and automatic draft caching UI.
4. Run full quality gates and manual browser verification.

## Multi-business passport refresh plan

### Overview

Allow one authenticated account to own and switch between multiple business workspaces. Add an incremental document refresh flow that analyzes new PDFs for the selected business and produces grounded passport candidates without overwriting existing facts.

### Architecture decisions

- Keep business isolation at workspace membership boundary; selected workspace ID from browser is accepted only after production membership validation.
- Reuse the existing private upload and durable job pipeline; add refresh mode to job payload instead of duplicating extraction code.
- Keep human confirmation for extracted facts. AI may propose, but only evidence-backed candidates become passport facts after review.
- Strengthen extraction with field-specific semantic evidence rules, deterministic validation, and conversion quality gates.

### Ordered slices

1. Multi-business tenancy contract, migration, API, and tests.
2. Workspace switcher and create-business UI, with query isolation.
3. Refresh-document API and durable worker flow, with no destructive passport reset.
4. Passport upload/update UI and candidate refresh behavior.
5. Extraction accuracy gates, semantic rules, regression tests, and final review.

## Production extraction recall repair

### Acceptance criteria

1. PDF có bảng được bổ sung text `pdftotext -layout` để giữ quan hệ nhãn–giá trị; PDF ít text vẫn dùng OCR.
2. Gemini nhận catalog có label/type/semantic guidance, dùng `thinkingLevel=high`, và chạy completeness pass có mục tiêu cho field còn thiếu.
3. Worker log số raw/valid/rejected candidate và nhóm lý do, không log quote hoặc giá trị nhạy cảm.
4. Candidate vẫn phải qua exact-evidence và semantic validation; không tự động xác nhận Passport.
5. Test, lint, build và Railway production health đều đạt.
