# P2B MVP tasks

- [x] Foundation, OpenAPI contract, migrations and deploy configuration
- [x] Passport domain, evidence provenance and human review workflow
- [x] Published-policy retrieval demo and deterministic eligibility
- [x] Candidate-only enrichment and evidence-gated checklist
- [x] Application review, version-pinning boundary and PDF export
- [x] Admin policy review queue and in-app alerts
- [x] Responsive black-purple frontend
- [x] Browser golden path, mobile visual QA and API hardening tests

Production wiring after credentials and official source allowlists are supplied:

- [ ] Replace in-memory repository with Supabase PostgreSQL/Storage adapters
- [ ] Enable Supabase JWT membership lookup and admin `app_metadata` enforcement
- [ ] Connect Gemini extraction/embedding/rerank/drafting adapters
- [ ] Configure VBPL and official-program crawler seeds, legal access review and schedules
- [ ] Connect the trusted-source fetcher to SSRF/DNS-rebinding and prompt-injection test suites
- [ ] Upload and approve real DOCX templates; switch demo PDF fallback to LibreOffice merge worker
