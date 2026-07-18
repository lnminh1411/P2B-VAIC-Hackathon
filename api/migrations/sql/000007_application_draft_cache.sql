CREATE TABLE IF NOT EXISTS application_draft_templates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 160),
    filename text NOT NULL CHECK (char_length(filename) BETWEEN 1 AND 255),
    content_type text NOT NULL CHECK (content_type IN ('application/pdf', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', 'text/plain')),
    source_text text NOT NULL CHECK (char_length(source_text) BETWEEN 1 AND 500000),
    placeholders jsonb NOT NULL DEFAULT '[]',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS application_draft_templates_workspace_idx ON application_draft_templates (workspace_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS application_draft_cache (
    application_id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    checklist_id text NOT NULL,
    policy_id text NOT NULL,
    policy_title text NOT NULL DEFAULT '',
    policy_agency text NOT NULL DEFAULT '',
    passport_version integer NOT NULL,
    policy_version integer NOT NULL,
    template_id uuid REFERENCES application_draft_templates(id) ON DELETE SET NULL,
    template_name text NOT NULL DEFAULT 'Mẫu P2B mặc định',
    template_version integer NOT NULL DEFAULT 1,
    version integer NOT NULL,
    status text NOT NULL,
    sections jsonb NOT NULL DEFAULT '{}',
    blocking_reasons jsonb NOT NULL DEFAULT '[]',
    generation_warning text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS application_draft_cache_workspace_idx ON application_draft_cache (workspace_id, updated_at DESC);
