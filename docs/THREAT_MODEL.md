# P2B MVP threat model

## Trust boundaries

- Browser to Go API: Supabase JWT, untrusted form values and idempotency keys.
- Browser to private Storage: short-lived signed upload/download URLs only.
- PDF and websites to worker: hostile content, prompt injection, decompression and resource exhaustion.
- Gemini responses to domain: untrusted JSON candidates; schema and evidence validation required.
- Crawler to policy corpus: untrusted web content until an admin activates a version.

## Assets

Company documents, Passport facts, policy review decisions, application snapshots, access tokens and provider keys.

## Implemented controls in the runnable MVP

- Demo resources are scoped by `workspace_id`; the Supabase schema has RLS enabled with no browser policies.
- Upload presign validates PDF type and 20 MB/file; Passport build caps the source list at 10.
- AI candidates are typed, evidence-grounded and never auto-confirmed.
- Policy and application activation are explicit human actions with version checks.
- CORS allowlist, CSP/security headers, bounded bodies/timeouts, replay-safe idempotency and generic errors.
- No document text, tokens, service keys or full Passport payloads in logs.

## Covered by automated tests

- Unknown Passport field and unsupported rule operator.
- Missing/conflicted facts incorrectly treated as `NOT_MET`.
- Duplicate mutation replay, idempotency-key misuse and stale aggregate updates.
- Multi-field checklist items incorrectly treated as available.
- Null API collections crashing application review.
- Full application approval and generated PDF validity boundary.

## Production release gates

- Verify Supabase JWT and membership on every API request; enforce admin role from trusted claims.
- MIME sniff, ClamAV scan, page/text caps and OCR sandbox before any document enters extraction.
- Reject loopback, private, link-local and metadata IPs; pin DNS decisions and revalidate every redirect.
- Isolate untrusted web text from agent instructions and reject unsupported evidence.
- Test cross-tenant enumeration against the PostgreSQL adapter and signed Storage paths.
- Confirm unreviewed policies can never enter retrieval and crawl changes require admin approval.
