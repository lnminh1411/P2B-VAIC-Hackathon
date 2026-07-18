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
	for _, key := range []string{"tax_code", "employee_count", "province", "charter_capital", "technologies", "green_project"} {
		field, exists := passport.Fields[key]
		if !exists || field.Value != nil || field.Status != domain.FieldMissing {
			t.Fatalf("field %q = %#v, want empty MISSING placeholder", key, field)
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

func TestMatchReturnsEmptyArrayWhenNoPublishedPoliciesExist(t *testing.T) {
	service := NewService(nil)
	if _, err := service.BuildPassport("workspace", BuildPassportInput{CompanyName: "Công ty thật"}); err != nil {
		t.Fatal(err)
	}

	run := service.Match("workspace")
	if run.Results == nil || len(run.Results) != 0 {
		t.Fatalf("results = %#v, want non-nil empty array", run.Results)
	}
}
