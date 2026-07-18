package platform

import (
	"testing"
	"time"

	"github.com/p2b/p2b/internal/domain"
)

func TestBuildPassportDoesNotInventCompanyFacts(t *testing.T) {
	service := NewService(nil)
	workspaceID := "real-company-only"

	_, err := service.BuildPassport(workspaceID, BuildPassportInput{
		CompanyName:  "Công ty TNHH Dữ liệu Thật",
		Website:      "https://dulieuthat.vn",
		SupportNeeds: []string{"Vốn ưu đãi"},
		SourceNames:  []string{"dang-ky-doanh-nghiep.pdf"},
	})
	if err != nil {
		t.Fatalf("build passport: %v", err)
	}

	passport := service.Passport(workspaceID)
	if len(passport.Fields) != 2 {
		t.Fatalf("fields = %#v, want only user-provided legal_name and website", passport.Fields)
	}
	for _, key := range []string{"tax_code", "employee_count", "province", "charter_capital", "technologies", "green_project"} {
		if _, exists := passport.Fields[key]; exists {
			t.Fatalf("invented field %q must not exist", key)
		}
	}
	if candidates := service.Candidates(workspaceID); len(candidates) != 0 {
		t.Fatalf("invented candidates = %#v", candidates)
	}
	if alerts := service.Alerts(workspaceID); len(alerts) != 0 {
		t.Fatalf("invented alerts = %#v", alerts)
	}
	if policies := service.Policies(false); len(policies) != 0 {
		t.Fatalf("hardcoded policies = %#v", policies)
	}
}

func TestEnrichmentReturnsNoResultsWithoutARealCollector(t *testing.T) {
	policy := domain.Policy{
		ID: "test-policy", Version: 1, Title: "Test policy", Lifecycle: "ACTIVE", VerifiedAt: time.Now().UTC(),
		Rules: []domain.Rule{{ID: "tax-code", FieldKey: "tax_code", Operator: domain.OpExists, Required: true}},
	}
	service := NewService([]domain.Policy{policy})
	if _, err := service.BuildPassport("workspace", BuildPassportInput{CompanyName: "Công ty thật"}); err != nil {
		t.Fatalf("build passport: %v", err)
	}

	run, err := service.StartEnrichment("workspace", policy.ID)
	if err != nil {
		t.Fatalf("start enrichment: %v", err)
	}
	if run.Status != "NO_RESULTS" || len(run.Candidates) != 0 {
		t.Fatalf("run = %#v, want NO_RESULTS without invented evidence", run)
	}
}
