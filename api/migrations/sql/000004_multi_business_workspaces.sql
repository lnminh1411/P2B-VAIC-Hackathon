ALTER TABLE workspaces DROP CONSTRAINT IF EXISTS workspaces_owner_subject_key;

CREATE INDEX IF NOT EXISTS workspaces_owner_subject_idx ON workspaces(owner_subject);
