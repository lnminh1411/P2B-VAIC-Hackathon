package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p2b/p2b/internal/domain"
)

const (
	maxPolicies         = 5000
	maxDocumentMatches  = 20
	embeddingDimensions = 768
)

type Embedder interface {
	Embed(context.Context, string) ([]float32, error)
}

type Store struct {
	database *pgxpool.Pool
	embedder Embedder
}

func NewStore(database *pgxpool.Pool, embedders ...Embedder) *Store {
	store := &Store{database: database}
	if len(embedders) > 0 {
		store.embedder = embedders[0]
	}
	return store
}

// Policies returns at most 5,000 newest policy versions. Public matching only
// receives ACTIVE versions which have an explicit review timestamp.
func (s *Store) Policies(ctx context.Context, activeOnly bool) ([]domain.Policy, error) {
	rows, err := s.database.Query(ctx, `
		SELECT pv.policy_key, pv.version, pv.title, pv.agency, pv.benefit,
		       COALESCE(pv.benefit_amount, ''), pv.support_type, pv.sectors,
		       pv.geographies, pv.deadline, pv.rules, pv.checklist_template,
		       pv.lifecycle, pv.verified_at, COALESCE(source.canonical_url, ''),
		       pv.template_ready
		FROM policy_versions pv
		LEFT JOIN LATERAL (
			SELECT ld.canonical_url
			FROM unnest(pv.source_document_version_ids) AS sources(source_version_id)
			JOIN document_versions dv ON dv.id = source_version_id
			JOIN legal_documents ld ON ld.id = dv.legal_document_id
			ORDER BY dv.version DESC
			LIMIT 1
		) source ON true
		WHERE NOT $1::boolean OR (pv.lifecycle = 'ACTIVE' AND pv.verified_at IS NOT NULL)
		ORDER BY pv.verified_at DESC NULLS LAST, pv.created_at DESC
		LIMIT $2`, activeOnly, maxPolicies)
	if err != nil {
		return nil, fmt.Errorf("query policies: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Policy, 0)
	for rows.Next() {
		var item domain.Policy
		var deadline, verifiedAt *time.Time
		var encodedRules, encodedChecklist []byte
		if err = rows.Scan(
			&item.ID, &item.Version, &item.Title, &item.Agency, &item.Benefit,
			&item.BenefitAmount, &item.SupportType, &item.Sectors, &item.Geographies,
			&deadline, &encodedRules, &encodedChecklist, &item.Lifecycle, &verifiedAt,
			&item.SourceURL, &item.TemplateReady,
		); err != nil {
			return nil, fmt.Errorf("scan policy: %w", err)
		}
		if deadline != nil {
			item.Deadline = deadline.UTC()
		}
		if verifiedAt != nil {
			item.VerifiedAt = verifiedAt.UTC()
		}
		if err = json.Unmarshal(encodedRules, &item.Rules); err != nil {
			return nil, fmt.Errorf("decode rules for policy %q: %w", item.ID, err)
		}
		if err = json.Unmarshal(encodedChecklist, &item.Checklist); err != nil {
			return nil, fmt.Errorf("decode checklist for policy %q: %w", item.ID, err)
		}
		result = append(result, item)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate policies: %w", err)
	}
	return result, nil
}

// SearchDocuments retrieves currently effective legal documents using reciprocal
// rank fusion over PostgreSQL full-text rank and 768-dimensional cosine distance.
// If ONNX inference is temporarily unavailable, matching degrades to FTS instead
// of returning an empty corpus.
func (s *Store) SearchDocuments(ctx context.Context, pass domain.Passport) ([]domain.DocumentMatch, string, error) {
	semanticText, lexicalText := buildPassportSearchText(pass)
	if s.embedder == nil {
		matches, err := s.searchDocumentsFTS(ctx, lexicalText)
		return matches, "RULE_FTS_FALLBACK", err
	}

	vector, err := s.embedder.Embed(ctx, semanticText)
	if err != nil {
		slog.WarnContext(ctx, "query embedding unavailable; using full-text fallback", "error", err)
		matches, searchErr := s.searchDocumentsFTS(ctx, lexicalText)
		return matches, "RULE_FTS_FALLBACK", searchErr
	}
	encodedVector, err := formatVector(vector)
	if err != nil {
		slog.WarnContext(ctx, "query embedding invalid; using full-text fallback", "error", err)
		matches, searchErr := s.searchDocumentsFTS(ctx, lexicalText)
		return matches, "RULE_FTS_FALLBACK", searchErr
	}

	rows, err := s.database.Query(ctx, hybridDocumentSearchSQL, lexicalText, encodedVector, maxDocumentMatches)
	if err != nil {
		return nil, "", fmt.Errorf("hybrid document search: %w", err)
	}
	defer rows.Close()

	matches := make([]domain.DocumentMatch, 0, maxDocumentMatches)
	for rows.Next() {
		var match domain.DocumentMatch
		if err = rows.Scan(
			&match.ID, &match.Version, &match.Title, &match.Agency, &match.Excerpt,
			&match.SourceURL, &match.LexicalScore, &match.VectorScore, &match.HybridScore,
		); err != nil {
			return nil, "", fmt.Errorf("scan hybrid document: %w", err)
		}
		match.SourceURL = safeSourceURL(match.SourceURL)
		matches = append(matches, match)
	}
	if err = rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate hybrid documents: %w", err)
	}
	return matches, "HYBRID_RULE_VECTOR", nil
}

func (s *Store) searchDocumentsFTS(ctx context.Context, lexicalText string) ([]domain.DocumentMatch, error) {
	rows, err := s.database.Query(ctx, fullTextDocumentSearchSQL, lexicalText, maxDocumentMatches)
	if err != nil {
		return nil, fmt.Errorf("full-text document search: %w", err)
	}
	defer rows.Close()

	matches := make([]domain.DocumentMatch, 0, maxDocumentMatches)
	for rows.Next() {
		var match domain.DocumentMatch
		if err = rows.Scan(
			&match.ID, &match.Version, &match.Title, &match.Agency, &match.Excerpt,
			&match.SourceURL, &match.LexicalScore, &match.VectorScore, &match.HybridScore,
		); err != nil {
			return nil, fmt.Errorf("scan full-text document: %w", err)
		}
		match.SourceURL = safeSourceURL(match.SourceURL)
		matches = append(matches, match)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate full-text documents: %w", err)
	}
	return matches, nil
}

func buildPassportSearchText(pass domain.Passport) (string, string) {
	needs := cleanSearchParts(pass.SupportNeeds)
	facts := make([]string, 0, len(pass.Fields))
	fieldKeys := make([]string, 0, len(pass.Fields))
	for key := range pass.Fields {
		fieldKeys = append(fieldKeys, key)
	}
	sort.Strings(fieldKeys)
	for _, key := range fieldKeys {
		field := pass.Fields[key]
		if field.Status != domain.FieldConfirmed || field.Value == nil {
			continue
		}
		value := strings.TrimSpace(fmt.Sprint(field.Value))
		if value == "" {
			continue
		}
		label := strings.TrimSpace(field.Label)
		if label == "" {
			label = key
		}
		facts = append(facts, label+": "+value)
	}

	semanticParts := []string{"query: Tìm chính sách, ưu đãi và chương trình hỗ trợ phù hợp cho doanh nghiệp Việt Nam."}
	if company := strings.TrimSpace(pass.CompanyName); company != "" {
		semanticParts = append(semanticParts, "Doanh nghiệp: "+company+".")
	}
	if len(needs) > 0 {
		semanticParts = append(semanticParts, "Nhu cầu hỗ trợ: "+strings.Join(needs, "; ")+".")
	}
	if len(facts) > 0 {
		semanticParts = append(semanticParts, "Thông tin đã xác nhận: "+strings.Join(facts, "; ")+".")
	}
	semantic := truncateRunes(strings.Join(semanticParts, " "), 4000)

	lexicalParts := []string{"doanh nghiệp", "hỗ trợ", "ưu đãi"}
	lexicalParts = append(lexicalParts, needs...)
	for _, fact := range facts {
		if _, value, ok := strings.Cut(fact, ":"); ok {
			lexicalParts = append(lexicalParts, strings.TrimSpace(value))
		}
	}
	quoted := make([]string, 0, len(lexicalParts))
	for _, part := range cleanSearchParts(lexicalParts) {
		part = strings.ReplaceAll(part, `"`, " ")
		quoted = append(quoted, `"`+part+`"`)
	}
	return semantic, strings.Join(quoted, " OR ")
}

func cleanSearchParts(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.Join(strings.Fields(value), " ")
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func formatVector(vector []float32) (string, error) {
	if len(vector) != embeddingDimensions {
		return "", fmt.Errorf("embedding has %d dimensions, want %d", len(vector), embeddingDimensions)
	}
	var builder strings.Builder
	builder.Grow(len(vector) * 12)
	builder.WriteByte('[')
	for index, value := range vector {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return "", fmt.Errorf("embedding dimension %d is not finite", index)
		}
		if index > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(strconv.FormatFloat(float64(value), 'g', -1, 32))
	}
	builder.WriteByte(']')
	return builder.String(), nil
}

func safeSourceURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return ""
	}
	return parsed.String()
}

const hybridDocumentSearchSQL = `
WITH query AS (
	SELECT websearch_to_tsquery('simple', $1) AS text_query, $2::vector(768) AS embedding
),
lexical AS (
	SELECT dc.id,
	       row_number() OVER (ORDER BY ts_rank_cd(dc.search_vector, query.text_query) DESC, dc.id) AS retrieval_rank,
	       ts_rank_cd(dc.search_vector, query.text_query)::double precision AS lexical_score
	FROM document_chunks dc
	JOIN document_versions dv ON dv.id = dc.document_version_id
	CROSS JOIN query
	WHERE dc.search_vector @@ query.text_query
	  AND dv.version = (SELECT max(latest.version) FROM document_versions latest WHERE latest.legal_document_id = dv.legal_document_id)
	  AND (dv.effective_from IS NULL OR dv.effective_from <= current_date)
	  AND (dv.effective_to IS NULL OR dv.effective_to >= current_date)
	ORDER BY lexical_score DESC, dc.id
	LIMIT 50
),
semantic AS (
	SELECT dc.id,
	       row_number() OVER (ORDER BY dc.embedding <=> query.embedding, dc.id) AS retrieval_rank,
	       (1 - (dc.embedding <=> query.embedding))::double precision AS vector_score
	FROM document_chunks dc
	JOIN document_versions dv ON dv.id = dc.document_version_id
	CROSS JOIN query
	WHERE dc.embedding IS NOT NULL
	  AND dv.version = (SELECT max(latest.version) FROM document_versions latest WHERE latest.legal_document_id = dv.legal_document_id)
	  AND (dv.effective_from IS NULL OR dv.effective_from <= current_date)
	  AND (dv.effective_to IS NULL OR dv.effective_to >= current_date)
	ORDER BY dc.embedding <=> query.embedding, dc.id
	LIMIT 50
),
fused AS (
	SELECT coalesce(lexical.id, semantic.id) AS chunk_id,
	       coalesce(lexical.lexical_score, 0)::double precision AS lexical_score,
	       coalesce(semantic.vector_score, 0)::double precision AS vector_score,
	       (coalesce(1.0 / (60 + lexical.retrieval_rank), 0) +
	        coalesce(1.0 / (60 + semantic.retrieval_rank), 0))::double precision AS hybrid_score
	FROM lexical
	FULL OUTER JOIN semantic ON semantic.id = lexical.id
),
candidates AS (
	SELECT ld.id::text AS document_id, dv.version,
	       coalesce(nullif(ld.document_number, ''), nullif(regexp_replace(dv.raw_object_key, '^.*/', ''), ''), 'Văn bản pháp luật') AS title,
	       ld.issuing_agency, left(dc.content, 1200) AS excerpt, ld.canonical_url,
	       fused.lexical_score, fused.vector_score, fused.hybrid_score,
	       row_number() OVER (PARTITION BY ld.id ORDER BY fused.hybrid_score DESC, dc.ordinal) AS document_rank
	FROM fused
	JOIN document_chunks dc ON dc.id = fused.chunk_id
	JOIN document_versions dv ON dv.id = dc.document_version_id
	JOIN legal_documents ld ON ld.id = dv.legal_document_id
)
SELECT document_id, version, title, issuing_agency, excerpt, canonical_url,
	   lexical_score, vector_score, hybrid_score
FROM candidates
WHERE document_rank = 1
ORDER BY hybrid_score DESC, document_id
LIMIT $3`

const fullTextDocumentSearchSQL = `
WITH query AS (
	SELECT websearch_to_tsquery('simple', $1) AS text_query
),
ranked AS (
	SELECT ld.id::text AS document_id, dv.version,
	       coalesce(nullif(ld.document_number, ''), nullif(regexp_replace(dv.raw_object_key, '^.*/', ''), ''), 'Văn bản pháp luật') AS title,
	       ld.issuing_agency, left(dc.content, 1200) AS excerpt, ld.canonical_url,
	       ts_rank_cd(dc.search_vector, query.text_query)::double precision AS lexical_score,
	       (1.0 / (60 + row_number() OVER (ORDER BY ts_rank_cd(dc.search_vector, query.text_query) DESC, dc.id)))::double precision AS hybrid_score,
	       row_number() OVER (PARTITION BY ld.id ORDER BY ts_rank_cd(dc.search_vector, query.text_query) DESC, dc.ordinal) AS document_rank
	FROM document_chunks dc
	JOIN document_versions dv ON dv.id = dc.document_version_id
	JOIN legal_documents ld ON ld.id = dv.legal_document_id
	CROSS JOIN query
	WHERE dc.search_vector @@ query.text_query
	  AND dv.version = (SELECT max(latest.version) FROM document_versions latest WHERE latest.legal_document_id = dv.legal_document_id)
	  AND (dv.effective_from IS NULL OR dv.effective_from <= current_date)
	  AND (dv.effective_to IS NULL OR dv.effective_to >= current_date)
)
SELECT document_id, version, title, issuing_agency, excerpt, canonical_url,
	   lexical_score, 0::double precision AS vector_score, hybrid_score
FROM ranked
WHERE document_rank = 1
ORDER BY hybrid_score DESC, document_id
LIMIT $2`
