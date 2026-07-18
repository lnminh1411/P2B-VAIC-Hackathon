# P2B MVP threat model

## Trust boundaries

- Browser to Go API: Supabase JWT, untrusted form values and idempotency keys.
- Browser to private Storage: short-lived signed upload/download URLs only.
- PDF and websites to worker: hostile content, prompt injection, decompression and resource exhaustion.
- Gemini responses to domain: untrusted JSON candidates; schema and evidence validation required.
- Crawler to policy corpus: untrusted web content until an admin activates a version.

## Assets

Company documents, Passport facts, policy review decisions, application snapshots, access tokens and provider keys.

## Required controls

- Every workspace-owned query scopes by authenticated `workspace_id`; unknown/cross-workspace IDs return 404.
- PDF MIME sniffing, 20 MB/file and 200-page/workspace build caps; no user input enters shell commands.
- HTTPS-only website collection; reject loopback, private, link-local and metadata IPs and revalidate redirects.
- AI output never executes code, SQL, markup or network calls; strict typed decode and evidence matching.
- Policy and application activation are human actions with version checks and append-only audit events.
- CORS allowlist, CSP/security headers, bounded bodies/timeouts and generic external errors.
- No document text, tokens, service keys or full Passport payloads in logs.

## Abuse cases covered by tests

- Unknown Passport field and unsupported rule operator.
- Missing/conflicted facts incorrectly treated as `NOT_MET`.
- Cross-workspace resource enumeration.
- Private-IP enrichment URL and DNS-rebinding redirect.
- Duplicate async requests and stale aggregate updates.
- Unreviewed policy becoming searchable or exportable.

