# Spec: Cached application drafts and LLM templates

## Objective

Allow an authenticated workspace to upload and reuse previous applications or templates, generate a policy-specific application draft with the existing Gemini 3.1 Flash-Lite configuration, and automatically persist edits so refreshes and later sessions do not lose work.

## Tech stack

- Go API, PostgreSQL/pgx, Chi
- Existing MarkItDown converter for PDF/DOCX/TXT text extraction
- Existing Gemini API key, model and endpoint conventions used by Passport extraction
- React, TypeScript, TanStack Query

## Commands

- API test: `cd api && go test ./...`
- Web test: `cd web && npm test -- --run`
- Web lint: `cd web && npm run lint`
- Web build: `cd web && npm run build`

## Project structure

- `api/internal/application/`: persistent draft/template repository and Gemini generation contract
- `api/internal/httpapi/`: authenticated endpoints and upload validation
- `api/migrations/sql/`: additive PostgreSQL schema
- `web/src/features/application/`: template picker, uploader, draft editor and autosave UI
- `web/src/lib/`: API contract and shared types

## Code style

Use explicit workspace-scoped methods and optimistic version checks:

```go
SaveDraft(ctx context.Context, workspaceID string, draft Draft, expectedVersion int) (Draft, error)
```

## Testing strategy

- Unit tests for placeholder extraction/rendering and Gemini structured output.
- Repository/handler tests use existing fakes where production PostgreSQL is unavailable.
- Component tests cover upload, selection, automatic save state, and restored drafts.
- Full Go and web suites plus production build are final gates.

## Boundaries

- Always: workspace isolation, file type/size validation, optimistic versions, HTML-free plain text, human review before submission.
- Ask first: changing Gemini provider/model family, destructive migration, automatic submission.
- Never: send unrelated workspace data to Gemini, log template/application content, overwrite a newer draft, store secrets in source.

## Success criteria

1. PDF, DOCX, or TXT template upload creates a reusable named template scoped to the active workspace.
2. Multiple templates are listed and selectable; selection is persisted on the generated draft.
3. Creating an application uses selected template, passport facts, and policy context with configured Gemini 3.1 Flash-Lite.
4. Placeholder values such as `{{company_name}}`, `{{tax_code}}`, and `{{policy_title}}` are grounded in known data.
5. Draft edits automatically save after a short idle period and restore after reload/login.
6. Gemini failure returns a grounded deterministic draft and a visible warning; user work is not lost.

## Open questions

- Future: preserve and export original DOCX formatting. Current scope caches extracted template text and generates editable application sections/PDF.
