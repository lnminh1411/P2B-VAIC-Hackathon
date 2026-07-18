package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p2b/p2b/internal/domain"
)

const maxPolicies = 5000

type Store struct {
	database *pgxpool.Pool
}

func NewStore(database *pgxpool.Pool) *Store {
	return &Store{database: database}
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
