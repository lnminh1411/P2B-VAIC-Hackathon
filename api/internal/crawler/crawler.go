package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p2b/p2b/internal/domain"
	"github.com/p2b/p2b/internal/pipeline"
)

type WatchlistSettings struct {
	NewPolicies       bool `json:"new_policies"`
	DeadlineChanges   bool `json:"deadline_changes"`
	StaleEvidence     bool `json:"stale_evidence"`
	UpcomingDeadlines bool `json:"upcoming_deadlines"`
}

func RunCrawler(ctx context.Context, db *pgxpool.Pool, store *pipeline.Store) {
	slog.Info("Starting crawler pass")
	now := time.Now().UTC()

	// 1. Get all companies and their watchlist settings
	rows, err := db.Query(ctx, `SELECT workspace_id, watchlist_settings, support_needs FROM companies`)
	if err != nil {
		slog.Error("crawler failed to query companies", "error", err)
		return
	}
	defer rows.Close()

	type CompanyInfo struct {
		WorkspaceID  string
		Settings     WatchlistSettings
		SupportNeeds []string
	}
	var companies []CompanyInfo

	for rows.Next() {
		var workspaceID string
		var encodedSettings []byte
		var supportNeeds []string
		if err := rows.Scan(&workspaceID, &encodedSettings, &supportNeeds); err != nil {
			slog.Error("crawler failed to scan company", "error", err)
			continue
		}
		var settings WatchlistSettings
		if err := json.Unmarshal(encodedSettings, &settings); err != nil {
			slog.Error("crawler failed to unmarshal watchlist settings", "error", err)
			continue
		}
		companies = append(companies, CompanyInfo{
			WorkspaceID:  workspaceID,
			Settings:     settings,
			SupportNeeds: supportNeeds,
		})
	}
	rows.Close()

	slog.Info("crawler found companies", "count", len(companies))

	// 2. Loop through companies and evaluate watchlist triggers
	for _, comp := range companies {
		// A. Check NEW_POLICIES
		if comp.Settings.NewPolicies {
			policyRows, err := db.Query(ctx, `
				SELECT policy_key, title, agency, verified_at
				FROM policy_versions
				WHERE lifecycle = 'ACTIVE' AND verified_at IS NOT NULL AND support_type = ANY($1)
				ORDER BY verified_at DESC`, comp.SupportNeeds)
			if err == nil {
				for policyRows.Next() {
					var policyKey, title, agency string
					var verifiedAt *time.Time
					if err := policyRows.Scan(&policyKey, &title, &agency, &verifiedAt); err == nil {
						var exists bool
						err := db.QueryRow(ctx, `
							SELECT EXISTS(
								SELECT 1 FROM alerts
								WHERE workspace_id = $1::uuid AND type = 'POLICY_NEW' AND payload->>'policy_id' = $2
							)`, comp.WorkspaceID, policyKey).Scan(&exists)
						if err == nil && !exists {
							slog.Info("crawler generating POLICY_NEW alert", "workspace", comp.WorkspaceID, "policy", policyKey)
							alert := domain.Alert{
								ID:         uuid.NewString(),
								Type:       "POLICY_NEW",
								Severity:   "info",
								Title:      "Chính sách hỗ trợ mới: " + title,
								Message:    fmt.Sprintf("Một chính sách mới của %s phù hợp với hồ sơ doanh nghiệp của bạn vừa được ban hành.", agency),
								PolicyID:   policyKey,
								Read:       false,
								OccurredAt: now,
							}
							if err := store.SaveAlert(ctx, comp.WorkspaceID, alert); err != nil {
								slog.Error("failed to save crawler alert", "error", err)
							}
						}
					}
				}
				policyRows.Close()
			}
		}

		// B. Check DEADLINE (Upcoming Deadlines: within 7 days)
		if comp.Settings.UpcomingDeadlines {
			policyRows, err := db.Query(ctx, `
				SELECT policy_key, title, deadline
				FROM policy_versions
				WHERE lifecycle = 'ACTIVE' AND deadline IS NOT NULL AND deadline > $1 AND deadline <= $2 AND support_type = ANY($3)`,
				now, now.Add(7*24*time.Hour), comp.SupportNeeds)
			if err == nil {
				for policyRows.Next() {
					var policyKey, title string
					var deadline time.Time
					if err := policyRows.Scan(&policyKey, &title, &deadline); err == nil {
						var exists bool
						err := db.QueryRow(ctx, `
							SELECT EXISTS(
								SELECT 1 FROM alerts
								WHERE workspace_id = $1::uuid AND type = 'DEADLINE' AND payload->>'policy_id' = $2
							)`, comp.WorkspaceID, policyKey).Scan(&exists)
						if err == nil && !exists {
							slog.Info("crawler generating DEADLINE alert", "workspace", comp.WorkspaceID, "policy", policyKey)
							alert := domain.Alert{
								ID:         uuid.NewString(),
								Type:       "DEADLINE",
								Severity:   "warning",
								Title:      "Sắp hết hạn nộp hồ sơ: " + title,
								Message:    fmt.Sprintf("Hạn chót đăng ký hỗ trợ là ngày %s. Vui lòng nộp hồ sơ sớm.", deadline.Format("02/01/2006")),
								PolicyID:   policyKey,
								Read:       false,
								OccurredAt: now,
							}
							if err := store.SaveAlert(ctx, comp.WorkspaceID, alert); err != nil {
								slog.Error("failed to save crawler alert", "error", err)
							}
						}
					}
				}
				policyRows.Close()
			}
		}

		// C. Check EVIDENCE_STALE (Stale Evidence: document expired)
		if comp.Settings.StaleEvidence {
			pass, err := store.Passport(ctx, comp.WorkspaceID)
			if err == nil && pass.CompanyName != "" {
				for fieldKey, field := range pass.Fields {
					if field.Status == domain.FieldConfirmed {
						for _, ev := range field.Evidence {
							if ev.ContentHash != "" {
								var effectiveTo *time.Time
								var docTitle string
								err := db.QueryRow(ctx, `
									SELECT dv.effective_to, ld.document_number
									FROM document_versions dv
									JOIN legal_documents ld ON ld.id = dv.legal_document_id
									WHERE dv.content_hash = $1`, ev.ContentHash).Scan(&effectiveTo, &docTitle)
								if err == nil && effectiveTo != nil && effectiveTo.Before(now) {
									var exists bool
									err := db.QueryRow(ctx, `
										SELECT EXISTS(
											SELECT 1 FROM alerts
											WHERE workspace_id = $1::uuid AND type = 'EVIDENCE_STALE' AND payload->>'message' LIKE $2
										)`, comp.WorkspaceID, "%"+fieldKey+"%").Scan(&exists)
									if err == nil && !exists {
										slog.Info("crawler generating EVIDENCE_STALE alert", "workspace", comp.WorkspaceID, "field", fieldKey)
										alert := domain.Alert{
											ID:         uuid.NewString(),
											Type:       "EVIDENCE_STALE",
											Severity:   "critical",
											Title:      "Dữ kiện pháp lý hết hiệu lực: " + field.Label,
											Message:    fmt.Sprintf("Văn bản căn cứ pháp lý (%s) cho dữ kiện '%s' đã hết hiệu lực ngày %s. Vui lòng cập nhật tài liệu mới.", docTitle, field.Label, effectiveTo.Format("02/01/2006")),
											Read:       false,
											OccurredAt: now,
										}
										if err := store.SaveAlert(ctx, comp.WorkspaceID, alert); err != nil {
											slog.Error("failed to save crawler alert", "error", err)
										}
									}
								}
							}
						}
					}
				}
			}
		}

		// D. Check POLICY_CHANGED (Deadline Changes)
		if comp.Settings.DeadlineChanges {
			policyRows, err := db.Query(ctx, `
				SELECT p1.policy_key, p1.title, p1.agency, p1.deadline
				FROM policy_versions p1
				JOIN policy_versions p2 ON p2.policy_key = p1.policy_key AND p2.version < p1.version
				WHERE p1.lifecycle = 'ACTIVE' AND p1.deadline != p2.deadline AND p1.support_type = ANY($1)`, comp.SupportNeeds)
			if err == nil {
				for policyRows.Next() {
					var policyKey, title, agency string
					var deadline time.Time
					if err := policyRows.Scan(&policyKey, &title, &agency, &deadline); err == nil {
						var exists bool
						err := db.QueryRow(ctx, `
							SELECT EXISTS(
								SELECT 1 FROM alerts
								WHERE workspace_id = $1::uuid AND type = 'POLICY_CHANGED' AND payload->>'policy_id' = $2
							)`, comp.WorkspaceID, policyKey).Scan(&exists)
						if err == nil && !exists {
							slog.Info("crawler generating POLICY_CHANGED alert", "workspace", comp.WorkspaceID, "policy", policyKey)
							alert := domain.Alert{
								ID:         uuid.NewString(),
								Type:       "POLICY_CHANGED",
								Severity:   "warning",
								Title:      "Thay đổi thời hạn chính sách: " + title,
								Message:    fmt.Sprintf("Thời hạn nộp hồ sơ cho chính sách '%s' của %s đã được cập nhật thành ngày %s.", title, agency, deadline.Format("02/01/2006")),
								PolicyID:   policyKey,
								Read:       false,
								OccurredAt: now,
							}
							if err := store.SaveAlert(ctx, comp.WorkspaceID, alert); err != nil {
								slog.Error("failed to save crawler alert", "error", err)
							}
						}
					}
				}
				policyRows.Close()
			}
		}
	}
	slog.Info("Crawler pass completed successfully")
}
