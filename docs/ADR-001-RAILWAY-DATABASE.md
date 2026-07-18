# ADR-001: Railway PostgreSQL owns business data

Status: accepted, 2026-07-18.

## Decision

- Railway runs PostgreSQL 17 with the official `pgvector/pgvector:0.8.2-pg17-bookworm` image, private networking and a persistent volume.
- Railway stores all P2B business data. Go API is the only business-data access path.
- Supabase remains identity provider and private object storage. Browser receives only Supabase publishable key.
- Supabase user UUID is copied as an external subject; Railway does not create foreign keys to `auth.users`.
- `p2b-migrate` applies embedded migrations under an advisory lock. Applied version and SHA-256 checksum are immutable.

## Security boundary

Go verifies every Supabase bearer token server-side, derives `workspace_id` from verified subject and ignores client workspace headers in production. Every future repository query must include `workspace_id`; browser never receives `DATABASE_URL` or Supabase secret key.

## Deployment and rollback

Railway runs `/usr/local/bin/p2b-migrate` before starting API. `/health/ready` returns `503` whenever DB ping fails. Schema migrations must be additive and backward compatible before deploy; rollback uses previous Railway deployment while retaining forward-compatible schema.

## Known transition

Existing Supabase business tables are historical and empty. They are not deleted by this change because deletion is irreversible. Remove them only after Railway persistence verification and an explicit backup/approval step.
