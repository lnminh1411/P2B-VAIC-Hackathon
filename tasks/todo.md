# P2B MVP tasks

- [x] Foundation, OpenAPI contract, migrations and deploy configuration
- [x] Passport domain, evidence provenance and human review workflow
- [x] Deterministic eligibility rule engine
- [x] Evidence-gated checklist and application state machine using test fixtures
- [x] Application review and version-pinning boundary
- [x] Admin/auth route boundaries and empty states
- [x] Responsive black-purple frontend
- [x] Browser golden path, mobile visual QA and API hardening tests

Production wiring after credentials and official source allowlists are supplied:

- [x] Persist company sources, Passport versions, extraction jobs and candidates in Railway PostgreSQL
- [x] Enable Supabase JWT verification, Railway workspace bootstrap and admin `app_metadata` enforcement
- [x] Connect MarkItDown PDF conversion and Gemini 3.1 Flash-Lite structured extraction
- [ ] Add OCR fallback for image-only PDFs and store page-level coordinates
- [ ] Move policy, matching, checklist and application repositories from memory to Railway PostgreSQL
- [ ] Configure VBPL and official-program crawler seeds, legal access review and schedules
- [ ] Connect the trusted-source fetcher to SSRF/DNS-rebinding and prompt-injection test suites
- [ ] Upload and approve real DOCX templates; replace minimal PDF writer with LibreOffice merge worker

## Multi-business passport refresh

- [x] Allow one account to own multiple business workspaces
- [x] Add authenticated workspace list/create/switch API
- [x] Add workspace switcher to the shell
- [x] Add incremental PDF refresh jobs for the selected business
- [x] Preserve existing passport facts during refresh
- [x] Add field-specific semantic evidence validation
- [x] Add extraction quality gates and regression coverage for charter capital and employee concepts

## Production extraction recall repair

- [x] Add layout-preserving PDF text supplementation
- [x] Add canonical field catalog to Gemini structured extraction
- [x] Add targeted completeness pass for missing fields
- [x] Add privacy-safe rejection diagnostics
- [x] Run full quality gates and deploy Railway backend/worker
