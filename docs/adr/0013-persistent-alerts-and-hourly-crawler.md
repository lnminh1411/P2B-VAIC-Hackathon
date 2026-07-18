# ADR-0013: Persistent Alerts and Hourly Crawler

Status: accepted, 2026-07-19.

## Context

Watchlist and policy monitoring settings are currently hardcoded as 'Not Activated' on the frontend. The `alerts` table exists in PostgreSQL, but there is no mechanism in the backend to query or persist alerts in the database, nor is there a functioning crawler worker that evaluates changes to generate real alerts.

## Decision

1. **Watchlist Preferences Storage**: Add a new JSONB column `watchlist_settings` to the `companies` table to store workspace-specific preferences for the four watchlist categories: `new_policies`, `deadline_changes`, `stale_evidence`, and `upcoming_deadlines`.
2. **Watchlist Toggle API**: Implement a `PUT /v1/watchlist/settings` endpoint in the Go API backend to persist a workspace's watchlist preferences.
3. **Persistent Alerts API**: Update `GET /v1/alerts` and `POST /v1/alerts/{alertID}/read` to read from and write to the database `alerts` table instead of using transient in-memory state.
4. **Hourly Background Crawler**: Implement a scheduled crawler inside `cmd/crawler/main.go` that runs every 1 hour to:
   - Identify active workspaces with enabled watchlist settings.
   - Scan for new policy changes (such as new `policy_versions` matching the company's profile or updated policy deadlines).
   - Generate relevant alerts (`POLICY_NEW`, `POLICY_CHANGED`, `DEADLINE`, `EVIDENCE_STALE`) and save them into the `alerts` database table for each matching workspace.
5. **UI Settings Toggles**: Refactor the frontend Alerts Page sidebar to display interactive toggles that query and update the workspace's watchlist settings via the new PUT API.

## Consequences

- Watchlist states will reflect the workspace's real settings stored in PostgreSQL.
- Alerts will persist across backend redeployments and server restarts.
- Alerts will no longer be simulated or transient; they will represent actual changes detected by the background crawler worker.
