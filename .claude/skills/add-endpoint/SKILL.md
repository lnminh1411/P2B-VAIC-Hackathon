---
name: add-endpoint
description: Checklist for adding or changing an HTTP endpoint in api/internal/httpapi/server.go. Use whenever adding a new route/handler, or changing an existing handler's behavior, in the P2B Go API — ensures the dev in-memory path and the Postgres production path both get updated.
---

`server.go` handlers branch on nil-checks (`s.config.ExtractionStore != nil`, and similarly for `WorkspaceManager`, `PolicyStore`, `DocumentSearcher`, `UploadSigner`, `ReadinessChecker`) to pick between two implementations of the same behavior:

- **dev/in-memory**: `s.service` (`*platform.Service`) — active when `DEV_AUTH=true` and no real stores are configured (`make api`)
- **prod/Postgres**: the injected store from `Config` (e.g. `pipeline.Store` via `ExtractionStore`) — active on Railway

When adding a new route or changing a handler's behavior, work through this checklist:

1. **Register the route** in `NewServerWithConfig`'s `router.Route("/v1", ...)` block.
2. **Add the method to the relevant interface** in the `Config` struct (e.g. `ExtractionStore`, `PolicyStore`, `ApplicationStore`) if the prod path needs a new store operation. These are anonymous inline interfaces, not named top-level ones — match that style.
3. **Implement the prod path**: add the method to the concrete store (usually `api/internal/pipeline/store.go`), with real SQL. Double-check every `$N` placeholder in the query has a matching bound argument passed to `Query`/`QueryRow`/`Exec` — a mismatched placeholder fails silently as a runtime DB error, not a compile error.
4. **Implement the dev path**: add the equivalent method to `platform.Service` (`api/internal/platform/*.go`) so `make api` (`DEV_AUTH=true`, no Postgres) keeps working.
5. **Wire the handler** to branch: `if s.config.ExtractionStore != nil { ... } else { s.service.X(...) }`, matching the existing pattern in nearby handlers.
6. **Verify both paths**: run `make api` (dev/in-memory) and exercise the endpoint, then check the prod-path code compiles and its SQL is correct (integration-test against Postgres if available, or at minimum re-read the query/args pairing).
7. Run `make test && make lint` before considering the change done — this matches CI exactly.
