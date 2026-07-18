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
			"legal_name": {Key: "legal_name", Value: "Công ty Xanh", Status: domain.FieldConfirmed},
			"province":   {Key: "province", Value: "Đà Nẵng", Status: domain.FieldConfirmed},
		},
	}
	documents := []domain.DocumentMatch{{
		ID: "legal-document", Version: 2, Title: "Nghị định hỗ trợ doanh nghiệp xanh",
		Agency: "Bộ Khoa học và Công nghệ", Excerpt: "Giới thiệu Giới thiệu Doanh nghiệp đổi mới công nghệ được hỗ trợ. Điều kiện áp dụng cần được đối chiếu theo phạm vi, đối tượng, ngoại lệ và thời hạn quy định trong văn bản nguồn. Hồ sơ phải được người phụ trách rà soát trước khi nộp. Nội dung lặp lại này chỉ dùng để kiểm tra phần trích dẫn được rút gọn, dễ đọc và không chiếm toàn bộ thẻ kết quả. Nội dung lặp lại này chỉ dùng để kiểm tra phần trích dẫn được rút gọn, dễ đọc và không chiếm toàn bộ thẻ kết quả.",
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
	if len([]rune(documentResult.Benefit)) > 360 {
		t.Fatalf("document excerpt has %d runes, want a compact legal summary", len([]rune(documentResult.Benefit)))
	}
	if len(documentResult.Eligibility.Criteria) < 3 {
		t.Fatalf("document criteria = %#v, want explicit human-review criteria", documentResult.Eligibility.Criteria)
	}
	for _, criterion := range documentResult.Eligibility.Criteria {
		if criterion.Status != eligibility.StatusMissingInfo || criterion.Citation.URL != documents[0].SourceURL {
			t.Fatalf("criterion = %#v, want missing-info criterion citing source document", criterion)
		}
	}
	if !documentResult.TemplateReady {
		t.Fatal("retrieved document should expose the generic P2B working template")
	}

	checklist, err := service.CreateChecklist("workspace", documentResult.PolicyID)
	if err != nil {
		t.Fatalf("create checklist from retrieved document: %v", err)
	}
	if len(checklist.Items) < 2 {
		t.Fatalf("retrieved document checklist = %#v, want actionable review items", checklist.Items)
	}
	for _, item := range checklist.Items {
		if item.Required && item.Status != "AVAILABLE" {
			checklist, err = service.UpdateChecklistItem("workspace", checklist.ID, item.ID, "AVAILABLE", "Người phụ trách đã đối chiếu văn bản nguồn", checklist.Version)
			if err != nil {
				t.Fatalf("confirm retrieved document checklist item: %v", err)
			}
		}
	}
	application, err := service.CreateApplication("workspace", checklist.ID)
	if err != nil {
		t.Fatalf("create application from retrieved document: %v", err)
	}
	if len(application.BlockingReasons) != 0 {
		t.Fatalf("application blockers = %v, want no missing-template dead end", application.BlockingReasons)
	}
}

func TestLoadMatchRunRestoresRetrievedPolicyChecklistAndSource(t *testing.T) {
	service := NewService(nil)
	run := MatchRun{ID: "cached-run", Results: []MatchResult{{
		PolicyID: "document-162", PolicyVersion: 2, Title: "162/2024/NĐ-CP", Agency: "Chính phủ",
		Benefit: "Quy định điều kiện cấp giấy phép", RetrievalMode: "HYBRID_RULE_VECTOR",
		SourceURL: "https://vbpl.vn/van-ban/162", TemplateReady: true,
	}}}
	service.LoadMatchRun("cached-workspace", run)

	checklist, err := service.CreateChecklist("cached-workspace", "document-162")
	if err != nil {
		t.Fatalf("create checklist from cached match: %v", err)
	}
	if len(checklist.Items) != 3 {
		t.Fatalf("cached retrieved policy checklist has %d items, want 3", len(checklist.Items))
	}
	policy, ok := service.workspace("cached-workspace").RetrievedPolicies["document-162"]
	if !ok || policy.SourceURL != "https://vbpl.vn/van-ban/162" {
		t.Fatalf("cached retrieved policy source = %q", policy.SourceURL)
	}
}
