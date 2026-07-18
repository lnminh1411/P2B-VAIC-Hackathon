ALTER TABLE company_sources
    ADD COLUMN IF NOT EXISTS extracted_markdown text,
    ADD COLUMN IF NOT EXISTS extraction_model text,
    ADD COLUMN IF NOT EXISTS extraction_error text,
    ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();

ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS progress integer NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
    ADD COLUMN IF NOT EXISTS completed_at timestamptz;

ALTER TABLE field_candidates
    ADD COLUMN IF NOT EXISTS source_id uuid REFERENCES company_sources(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS jobs_claim_idx ON jobs (available_at, created_at)
    WHERE status IN ('QUEUED', 'LEASED');

CREATE INDEX IF NOT EXISTS company_sources_status_idx ON company_sources (status, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS field_candidates_source_fact_idx
    ON field_candidates (source_id, field_key, md5(value::text))
    WHERE source_id IS NOT NULL;
