# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

P2B: Go 1.26 API (`api/`, module `github.com/p2b/p2b`, chi router, pgx/v5) + React 19/TypeScript/Vite frontend (`web/`). Deployed as Railway (API + Postgres, `infra/Dockerfile`) and Vercel (`web/` SPA). Supabase provides both auth (Google OAuth) and private file storage — it is not just an auth provider, treat it as part of the data layer alongside Railway Postgres.

## Commands

Always use the Makefile targets — they are the exact CI gate (`.github/workflows/ci.yml` runs `make test && make lint && make build`, nothing more):

- `make api` — run API in dev mode (`DEV_AUTH=true`, in-memory store, no Postgres/Supabase needed)
- `make web` — run Vite dev server
- `make test` — `go test ./...` + `npm test -- --run` (vitest, non-watch)
- `make lint` — `go vet ./...` + `npm run lint` (eslint flat config). There is no golangci-lint, Prettier, or Biome config in this repo — `go vet` and `eslint` are the only checks.
- `make build` — `go build ./cmd/...` + `tsc --noEmit && vite build` (web build type-checks first)
- `make migrate` — run Go migrations against `DATABASE_URL`

## Critical gotcha: dual dev/prod code paths in `api/internal/httpapi/server.go`

Nearly every handler branches on `s.config.ExtractionStore != nil` (and similar nil-checks for `WorkspaceManager`, `PolicyStore`, `DocumentSearcher`, `UploadSigner`, `ReadinessChecker`) to choose between:
- **dev/in-memory path**: `s.service` (`*platform.Service`), used when `DEV_AUTH=true` and no real stores are wired
- **prod path**: the injected Postgres-backed store (e.g. `pipeline.Store`), used on Railway

When adding or changing an endpoint, implement **both** branches — the in-memory `platform.Service` method and the corresponding store interface method — or `make api` (dev) and production will silently diverge in behavior.

## i18n

All UI strings live in one file: `web/src/lib/i18n.tsx`, as a single `translations = { vi: {...}, en: {...} }` object literal (not JSON, not per-feature files). When adding UI text, add the key under **both** `vi` and `en` with identical key structure. `useTranslation()` is exported from the same file.

## Env vars

Required names are documented in `.env.example` (no real values there). Notable: `DEV_AUTH` / `VITE_DEV_AUTH` toggle the in-memory dev mode described above. Never prefix `SUPABASE_SECRET_KEY` or `DATABASE_URL` with `VITE_` — that would expose them to the client bundle.

## Git workflow

Commit directly to `main` — no feature-branch/PR requirement for this repo.
