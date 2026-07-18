package platform

import (
	"testing"
	"time"

	"github.com/p2b/p2b/internal/domain"
	"github.com/p2b/p2b/internal/eligibility"
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

func TestMatchPassportUsesPersistedPassportFacts(t *testing.T) {
	policy := domain.Policy{
		ID: "green-support", Version: 1, Title: "Hỗ trợ công nghệ xanh", Lifecycle: "ACTIVE",
		SupportType: "công nghệ xanh",
		Rules:       []domain.Rule{{ID: "province", FieldKey: "province", Operator: domain.OpEQ, Expected: "Đà Nẵng", Required: true}},
	}
	service := NewService([]domain.Policy{policy})
	pass := domain.Passport{
		Version: 7, SupportNeeds: []string{"công nghệ xanh"},
		Fields: map[string]domain.PassportField{
			"province": {Key: "province", Value: "Đà Nẵng", Status: domain.FieldConfirmed},
		},
	}

	run := service.MatchPassport("workspace", pass)

	if run.PassportVersion != 7 {
		t.Fatalf("passport version = %d, want persisted version 7", run.PassportVersion)
	}
	if len(run.Results) != 1 || run.Results[0].Eligibility.Status != "MET" {
		t.Fatalf("results = %#v, want policy matched from persisted passport", run.Results)
	}
}

func TestMatchPassportHybridCombinesRulePoliciesAndRetrievedDocuments(t *testing.T) {
	policy := domain.Policy{
		ID: "green-support", Version: 1, Title: "Hỗ trợ công nghệ xanh", Lifecycle: "ACTIVE",
		SupportType: "công nghệ xanh",
		Rules:       []domain.Rule{{ID: "province", FieldKey: "province", Operator: domain.OpEQ, Expected: "Đà Nẵng", Required: true}},
	}
	service := NewService([]domain.Policy{policy})
	pass := domain.Passport{
		Version: 8, SupportNeeds: []string{"công nghệ xanh"},
		Fields: map[string]domain.PassportField{
			"province": {Key: "province", Value: "Đà Nẵng", Status: domain.FieldConfirmed},
		},
	}
	documents := []domain.DocumentMatch{{
		ID: "legal-document", Version: 2, Title: "Nghị định hỗ trợ doanh nghiệp xanh",
		Agency: "Bộ Khoa học và Công nghệ", Excerpt: "Doanh nghiệp đổi mới công nghệ được hỗ trợ.",
		SourceURL: "https://vbpl.vn/legal-document", VectorScore: 0.91, LexicalScore: 0.24, HybridScore: 0.032,
	}}

	run := service.MatchPassportHybrid("workspace", pass, documents, "HYBRID_RULE_VECTOR")

	if len(run.Results) != 2 {
		t.Fatalf("results = %#v, want one rule result and one document result", run.Results)
	}
	var documentResult *MatchResult
	for index := range run.Results {
		if run.Results[index].PolicyID == "legal-document" {
			documentResult = &run.Results[index]
		}
	}
	if documentResult == nil {
		t.Fatal("retrieved legal document missing from match results")
	}
	if documentResult.Eligibility.Status != eligibility.StatusMissingInfo {
		t.Fatalf("document eligibility = %s, want MISSING_INFO until rules are structured", documentResult.Eligibility.Status)
	}
	if documentResult.RetrievalMode != "HYBRID_RULE_VECTOR" || documentResult.SourceURL != documents[0].SourceURL {
		t.Fatalf("document result = %#v", documentResult)
	}
}
