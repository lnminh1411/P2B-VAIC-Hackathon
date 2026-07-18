package platform

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/p2b/p2b/internal/domain"
)

func TestApplicationBlockersReturnsEmptyJSONCollection(t *testing.T) {
	blockers := applicationBlockers(Checklist{}, true)
	if blockers == nil {
		t.Fatal("application blockers must be an empty slice, not nil")
	}
	if len(blockers) != 0 {
		t.Fatalf("expected no blockers, got %v", blockers)
	}
}

func TestCreateApplicationForRetrievedDocumentUsesReviewSpecificCopy(t *testing.T) {
	service := NewService([]domain.Policy{{
		ID: "decree-162", Version: 1, Title: "162/2024/NĐ-CP", Agency: "Chính phủ", Benefit: "Quy định điều kiện cấp giấy phép",
		Lifecycle: "ACTIVE", TemplateReady: true,
		Checklist: []domain.ChecklistTemplateItem{{Key: "applicability_review", Title: "Biên bản đối chiếu", Required: true}},
	}})
	workspaceID := "retrieved-application-copy"
	if _, err := service.BuildPassport(workspaceID, BuildPassportInput{CompanyName: "Công ty cổ phần SSI"}); err != nil {
		t.Fatal(err)
	}
	checklist, err := service.CreateChecklist(workspaceID, "decree-162")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.UpdateChecklistItem(workspaceID, checklist.ID, checklist.Items[0].ID, "AVAILABLE", "Đã đối chiếu", checklist.Version); err != nil {
		t.Fatal(err)
	}
	application, err := service.CreateApplication(workspaceID, checklist.ID)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(application.Sections["proposal"], "nguồn hỗ trợ") {
		t.Fatalf("retrieved legal document must not use support-program boilerplate: %q", application.Sections["proposal"])
	}
	if !strings.Contains(application.Sections["proposal"], "162/2024/NĐ-CP") {
		t.Fatalf("proposal must reference selected document, got %q", application.Sections["proposal"])
	}
}

func TestCreateApplicationFromTemplatePinsGeneratedDraftMetadata(t *testing.T) {
	service := NewService([]domain.Policy{{ID: "policy-template", Version: 3, Title: "Nghị định mẫu", Agency: "Chính phủ", Lifecycle: "ACTIVE", TemplateReady: true}})
	workspaceID := "application-template"
	if _, err := service.BuildPassport(workspaceID, BuildPassportInput{CompanyName: "Công ty P2B"}); err != nil {
		t.Fatal(err)
	}
	checklist, err := service.CreateChecklist(workspaceID, "policy-template")
	if err != nil {
		t.Fatal(err)
	}
	sections := map[string]string{"company_overview": "Bản sinh bởi Gemini", "support_need": "Đối chiếu", "proposal": "Chuẩn bị"}
	application, err := service.CreateApplicationFromTemplate(workspaceID, checklist.ID, "11111111-1111-1111-1111-111111111111", "Mẫu hồ sơ 2025", sections, "")
	if err != nil {
		t.Fatal(err)
	}
	if application.TemplateID == "" || application.TemplateName != "Mẫu hồ sơ 2025" || application.PolicyTitle != "Nghị định mẫu" {
		t.Fatalf("application metadata = %#v", application)
	}
	if application.Sections["company_overview"] != "Bản sinh bởi Gemini" {
		t.Fatalf("application sections = %#v", application.Sections)
	}
}

func TestRestoreApplicationMakesCachedDraftAvailable(t *testing.T) {
	service := NewService(nil)
	draft := Application{ID: "cached", Version: 7, Status: "DRAFT_READY", Sections: map[string]string{"proposal": "Đã lưu"}}
	service.RestoreApplication("workspace-cache", draft)
	got, err := service.Application("workspace-cache", draft.ID)
	if err != nil || got.Version != 7 || got.Sections["proposal"] != "Đã lưu" {
		t.Fatalf("restored application = %#v, err = %v", got, err)
	}
}

func TestApplicationPDFIsUnicodeAdministrativeMinutes(t *testing.T) {
	service := NewService([]domain.Policy{{
		ID: "decree-162", Version: 1, Title: "162/2024/NĐ-CP", Agency: "Chính phủ", Benefit: "Quy định điều kiện cấp giấy phép",
		Lifecycle: "ACTIVE", TemplateReady: true, SourceURL: "https://vbpl.vn/van-ban/162-2024-nd-cp",
		Checklist: []domain.ChecklistTemplateItem{
			{Key: "company_profile", Title: "Thông tin doanh nghiệp", Required: true, FieldKeys: []string{"legal_name"}},
			{Key: "applicability_review", Title: "Biên bản đối chiếu phạm vi áp dụng", Required: true},
		},
	}})
	workspaceID := "unicode-administrative-pdf"
	if _, err := service.BuildPassport(workspaceID, BuildPassportInput{CompanyName: "Công ty cổ phần SSI", Website: "https://www.ssi.com.vn/"}); err != nil {
		t.Fatal(err)
	}
	state := service.workspace(workspaceID)
	state.Passport.Fields["legal_name"] = domain.PassportField{Key: "legal_name", Value: "Công ty cổ phần SSI", Status: domain.FieldConfirmed}
	state.Passport.Fields["tax_code"] = domain.PassportField{Key: "tax_code", Value: "0301955155", Status: domain.FieldConfirmed}
	state.Passport.Fields["registered_address"] = domain.PassportField{Key: "registered_address", Value: "72 Nguyễn Huệ, Thành phố Hồ Chí Minh", Status: domain.FieldConfirmed}
	state.Passport.Fields["province"] = domain.PassportField{Key: "province", Value: "Thành phố Hồ Chí Minh", Status: domain.FieldConfirmed}

	checklist, err := service.CreateChecklist(workspaceID, "decree-162")
	if err != nil {
		t.Fatal(err)
	}
	application, err := service.CreateApplication(workspaceID, checklist.ID)
	if err != nil {
		t.Fatal(err)
	}
	application.Status = "GENERATED"
	state.Applications[application.ID] = application

	pdf, _, err := service.ApplicationPDF(workspaceID, application.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}
	if len(pdf) < 20_000 {
		t.Fatalf("PDF is only %d bytes; expected embedded Unicode font and structured layout", len(pdf))
	}
	if bytes.Contains(pdf, []byte("P2B APPLICATION PACKAGE")) {
		t.Fatal("legacy English text dump must not remain")
	}
	if previewPath := os.Getenv("P2B_PDF_PREVIEW_PATH"); previewPath != "" {
		if err = os.WriteFile(previewPath, pdf, 0o600); err != nil {
			t.Fatalf("write PDF preview: %v", err)
		}
	}
}

func TestCreateChecklistRequiresEveryReferencedField(t *testing.T) {
	service := NewService([]domain.Policy{{
		ID: "registration-policy", Version: 1, Title: "Registration policy", Lifecycle: "ACTIVE",
		Checklist: []domain.ChecklistTemplateItem{{Key: "registration", Title: "Registration", Required: true, FieldKeys: []string{"legal_name", "tax_code"}}},
	}})
	workspaceID := "checklist-all-fields"
	_, err := service.BuildPassport(workspaceID, BuildPassportInput{CompanyName: "Công ty kiểm thử"})
	if err != nil {
		t.Fatalf("build passport: %v", err)
	}

	var policyID string
	for _, policy := range service.Policies(true) {
		for _, item := range policy.Checklist {
			if item.Key == "registration" {
				policyID = policy.ID
				break
			}
		}
	}
	if policyID == "" {
		t.Fatal("test policy with registration checklist not found")
	}

	checklist, err := service.CreateChecklist(workspaceID, policyID)
	if err != nil {
		t.Fatalf("create checklist: %v", err)
	}
	for _, item := range checklist.Items {
		if item.TemplateKey == "registration" {
			if item.Status != "MISSING" {
				t.Fatalf("expected MISSING while tax_code is %s, got %s", domain.FieldExtracted, item.Status)
			}
			return
		}
	}
	t.Fatal("registration checklist item not found")
}

func TestCreateChecklistReturnsEmptyFieldKeysCollection(t *testing.T) {
	service := NewService([]domain.Policy{{
		ID: "document-policy", Version: 1, Title: "Document policy", Lifecycle: "ACTIVE",
		Checklist: []domain.ChecklistTemplateItem{{Key: "manual-review", Title: "Manual review", Required: true}},
	}})
	workspaceID := "checklist-empty-fields"
	if _, err := service.BuildPassport(workspaceID, BuildPassportInput{CompanyName: "Công ty kiểm thử"}); err != nil {
		t.Fatalf("build passport: %v", err)
	}

	checklist, err := service.CreateChecklist(workspaceID, "document-policy")
	if err != nil {
		t.Fatalf("create checklist: %v", err)
	}
	if len(checklist.Items) != 1 {
		t.Fatalf("expected one checklist item, got %d", len(checklist.Items))
	}
	if checklist.Items[0].FieldKeys == nil {
		t.Fatal("field_keys must be an empty slice, not null")
	}
}
