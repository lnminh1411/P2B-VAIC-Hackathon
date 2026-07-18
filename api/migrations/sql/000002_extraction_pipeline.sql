ALTER TABLE company_sources
    ADD COLUMN extracted_markdown text,
    ADD COLUMN extraction_model text,
    ADD COLUMN extraction_error text,
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

ALTER TABLE jobs
    ADD COLUMN progress integer NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
    ADD COLUMN completed_at timestamptz;

ALTER TABLE field_candidates
    ADD COLUMN source_id uuid REFERENCES company_sources(id) ON DELETE CASCADE;

CREATE INDEX jobs_claim_idx ON jobs (available_at, created_at)
    WHERE status IN ('QUEUED', 'LEASED');

CREATE INDEX company_sources_status_idx ON company_sources (status, created_at);

CREATE UNIQUE INDEX field_candidates_source_fact_idx
    ON field_candidates (source_id, field_key, md5(value::text))
    WHERE source_id IS NOT NULL;
