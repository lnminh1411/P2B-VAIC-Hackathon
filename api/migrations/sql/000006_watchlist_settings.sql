ALTER TABLE companies ADD COLUMN IF NOT EXISTS watchlist_settings JSONB NOT NULL DEFAULT '{"new_policies": false, "deadline_changes": false, "stale_evidence": false, "upcoming_deadlines": false}';
