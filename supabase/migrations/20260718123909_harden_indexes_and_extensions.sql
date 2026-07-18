CREATE SCHEMA IF NOT EXISTS extensions;
ALTER EXTENSION vector SET SCHEMA extensions;

CREATE INDEX application_templates_reviewer_idx ON application_templates(reviewed_by);
CREATE INDEX approval_snapshots_approver_idx ON approval_snapshots(approved_by);
CREATE INDEX checklists_workspace_idx ON checklists(workspace_id, updated_at DESC);
CREATE INDEX enrichment_runs_workspace_idx ON enrichment_runs(workspace_id, updated_at DESC);
CREATE INDEX exports_workspace_idx ON exports(workspace_id, created_at DESC);
CREATE INDEX policy_versions_reviewer_idx ON policy_versions(reviewer_subject) WHERE reviewer_subject IS NOT NULL;
