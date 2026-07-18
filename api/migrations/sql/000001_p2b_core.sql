CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE workspaces (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_subject uuid NOT NULL UNIQUE,
    display_name text NOT NULL DEFAULT 'Workspace doanh nghiệp',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE workspace_members (
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    subject uuid NOT NULL,
    role text NOT NULL CHECK (role IN ('OWNER','ADMIN','MEMBER','REVIEWER')),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, subject)
);

CREATE TABLE companies (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL UNIQUE REFERENCES workspaces(id) ON DELETE CASCADE,
    legal_name text NOT NULL,
    website text,
    support_needs text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE company_sources (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    filename text NOT NULL,
    content_type text NOT NULL CHECK (content_type = 'application/pdf'),
    size_bytes bigint NOT NULL CHECK (size_bytes BETWEEN 1 AND 20971520),
    object_key text NOT NULL UNIQUE,
    content_hash text,
    status text NOT NULL CHECK (status IN ('PENDING_UPLOAD','UPLOADED','EXTRACTING','EXTRACTED','FAILED')),
    page_count integer CHECK (page_count BETWEEN 0 AND 200),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE passports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL UNIQUE REFERENCES workspaces(id) ON DELETE CASCADE,
    current_version integer NOT NULL DEFAULT 1 CHECK (current_version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE passport_versions (
    passport_id uuid NOT NULL REFERENCES passports(id) ON DELETE CASCADE,
    version integer NOT NULL CHECK (version > 0),
    fields jsonb NOT NULL,
    support_needs text[] NOT NULL DEFAULT '{}',
    created_by uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (passport_id, version)
);

CREATE TABLE field_candidates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    passport_id uuid NOT NULL REFERENCES passports(id) ON DELETE CASCADE,
    field_key text NOT NULL,
    value jsonb NOT NULL,
    data_type text NOT NULL,
    status text NOT NULL CHECK (status IN ('EXTRACTED','NEEDS_REVIEW','CONFLICTED','ACCEPTED','REJECTED')),
    confidence numeric(4,3) NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    evidence jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE legal_documents (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_url text NOT NULL UNIQUE,
    issuing_agency text NOT NULL,
    document_number text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE document_versions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    legal_document_id uuid NOT NULL REFERENCES legal_documents(id) ON DELETE CASCADE,
    version integer NOT NULL CHECK (version > 0),
    content_hash text NOT NULL,
    raw_object_key text NOT NULL,
    effective_from date,
    effective_to date,
    crawled_at timestamptz NOT NULL,
    UNIQUE (legal_document_id, version),
    UNIQUE (legal_document_id, content_hash)
);

CREATE TABLE document_chunks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    document_version_id uuid NOT NULL REFERENCES document_versions(id) ON DELETE CASCADE,
    ordinal integer NOT NULL,
    content text NOT NULL,
    search_vector tsvector GENERATED ALWAYS AS (to_tsvector('simple', content)) STORED,
    embedding vector(768),
    embedding_model text,
    UNIQUE (document_version_id, ordinal)
);

CREATE TABLE policy_versions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_key text NOT NULL,
    version integer NOT NULL CHECK (version > 0),
    title text NOT NULL,
    agency text NOT NULL,
    support_type text NOT NULL,
    benefit text NOT NULL,
    benefit_amount text,
    sectors text[] NOT NULL DEFAULT '{}',
    geographies text[] NOT NULL DEFAULT '{}',
    deadline timestamptz,
    lifecycle text NOT NULL CHECK (lifecycle IN ('DRAFT','PENDING_REVIEW','ACTIVE','RETIRED','SUPERSEDED')),
    rules jsonb NOT NULL DEFAULT '[]',
    checklist_template jsonb NOT NULL DEFAULT '[]',
    source_document_version_ids uuid[] NOT NULL DEFAULT '{}',
    template_ready boolean NOT NULL DEFAULT false,
    verified_at timestamptz,
    reviewer_subject uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (policy_key, version)
);

CREATE TABLE match_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    passport_id uuid NOT NULL REFERENCES passports(id),
    passport_version integer NOT NULL,
    retrieval_mode text NOT NULL,
    results jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE enrichment_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    policy_version_id uuid NOT NULL REFERENCES policy_versions(id),
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE enrichment_candidates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    enrichment_run_id uuid NOT NULL REFERENCES enrichment_runs(id) ON DELETE CASCADE,
    field_key text NOT NULL,
    value jsonb NOT NULL,
    confidence numeric(4,3) NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    evidence jsonb NOT NULL,
    status text NOT NULL CHECK (status IN ('NEEDS_REVIEW','ACCEPTED','REJECTED')),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE checklists (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    policy_version_id uuid NOT NULL REFERENCES policy_versions(id),
    passport_version integer NOT NULL,
    version integer NOT NULL DEFAULT 1,
    items jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE application_templates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_version_id uuid NOT NULL REFERENCES policy_versions(id),
    version integer NOT NULL CHECK (version > 0),
    object_key text NOT NULL UNIQUE,
    placeholder_schema jsonb NOT NULL,
    active boolean NOT NULL DEFAULT false,
    reviewed_by uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (policy_version_id, version)
);

CREATE TABLE applications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    checklist_id uuid NOT NULL REFERENCES checklists(id),
    policy_version_id uuid NOT NULL REFERENCES policy_versions(id),
    template_id uuid NOT NULL REFERENCES application_templates(id),
    passport_version integer NOT NULL,
    version integer NOT NULL DEFAULT 1,
    status text NOT NULL CHECK (status IN ('PREPARING','DRAFT_READY','PENDING_REVIEW','APPROVED','GENERATING','GENERATED','FAILED','REJECTED')),
    sections jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE approval_snapshots (
    application_id uuid PRIMARY KEY REFERENCES applications(id) ON DELETE CASCADE,
    payload jsonb NOT NULL,
    payload_hash text NOT NULL,
    approved_by uuid NOT NULL,
    approved_at timestamptz NOT NULL
);

CREATE TABLE exports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    application_id uuid NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    object_key text NOT NULL UNIQUE,
    content_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE alerts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    type text NOT NULL,
    payload jsonb NOT NULL,
    severity text NOT NULL,
    read_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid REFERENCES workspaces(id) ON DELETE CASCADE,
    type text NOT NULL,
    payload jsonb NOT NULL,
    idempotency_key text NOT NULL UNIQUE,
    status text NOT NULL CHECK (status IN ('QUEUED','LEASED','SUCCEEDED','FAILED','DEAD_LETTER')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    max_attempts integer NOT NULL DEFAULT 5 CHECK (max_attempts > 0),
    available_at timestamptz NOT NULL DEFAULT now(),
    lease_expires_at timestamptz,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE audit_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid REFERENCES workspaces(id) ON DELETE CASCADE,
    actor_subject uuid NOT NULL,
    action text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE idempotency_keys (
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key text NOT NULL,
    request_hash text NOT NULL,
    response_status integer,
    response_body jsonb,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, key)
);

CREATE INDEX workspace_members_subject_idx ON workspace_members(subject, workspace_id);
CREATE INDEX company_sources_workspace_idx ON company_sources(workspace_id, created_at DESC);
CREATE INDEX passport_versions_created_by_idx ON passport_versions(created_by);
CREATE INDEX field_candidates_workspace_idx ON field_candidates(workspace_id, created_at DESC);
CREATE INDEX field_candidates_passport_idx ON field_candidates(passport_id, created_at DESC);
CREATE INDEX document_versions_document_idx ON document_versions(legal_document_id, version DESC);
CREATE INDEX document_chunks_version_idx ON document_chunks(document_version_id, ordinal);
CREATE INDEX document_chunks_fts_idx ON document_chunks USING gin(search_vector);
CREATE INDEX document_chunks_embedding_idx ON document_chunks USING hnsw (embedding vector_cosine_ops) WHERE embedding IS NOT NULL;
CREATE UNIQUE INDEX one_active_policy_version_idx ON policy_versions(policy_key) WHERE lifecycle = 'ACTIVE';
CREATE INDEX match_runs_workspace_idx ON match_runs(workspace_id, created_at DESC);
CREATE INDEX match_runs_passport_idx ON match_runs(passport_id, passport_version DESC);
CREATE INDEX enrichment_runs_policy_idx ON enrichment_runs(policy_version_id, created_at DESC);
CREATE INDEX enrichment_candidates_run_idx ON enrichment_candidates(enrichment_run_id, created_at DESC);
CREATE INDEX checklists_policy_idx ON checklists(policy_version_id, created_at DESC);
CREATE INDEX application_templates_policy_idx ON application_templates(policy_version_id, version DESC);
CREATE INDEX applications_workspace_idx ON applications(workspace_id, updated_at DESC);
CREATE INDEX applications_checklist_idx ON applications(checklist_id);
CREATE INDEX applications_policy_idx ON applications(policy_version_id);
CREATE INDEX applications_template_idx ON applications(template_id);
CREATE INDEX exports_application_idx ON exports(application_id, created_at DESC);
CREATE INDEX alerts_workspace_idx ON alerts(workspace_id, created_at DESC);
CREATE INDEX jobs_claim_idx ON jobs(status, available_at) WHERE status IN ('QUEUED','LEASED');
CREATE INDEX jobs_workspace_idx ON jobs(workspace_id, created_at DESC);
CREATE INDEX audit_events_workspace_idx ON audit_events(workspace_id, created_at DESC);
CREATE INDEX audit_events_actor_idx ON audit_events(actor_subject, created_at DESC);
CREATE UNIQUE INDEX audit_events_bootstrap_once_idx ON audit_events(workspace_id, action) WHERE action = 'auth.bootstrap';
