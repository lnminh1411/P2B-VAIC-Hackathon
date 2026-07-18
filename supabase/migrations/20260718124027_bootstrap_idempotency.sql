CREATE UNIQUE INDEX audit_events_workspace_bootstrap_idx
ON audit_events(workspace_id, action)
WHERE action = 'auth.bootstrap';
