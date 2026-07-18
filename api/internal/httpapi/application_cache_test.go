package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	applicationdomain "github.com/p2b/p2b/internal/application"
	"github.com/p2b/p2b/internal/domain"
	"github.com/p2b/p2b/internal/platform"
)

type applicationStoreFake struct {
	template applicationdomain.Template
	saved    platform.Application
}

func TestValidateApplicationTemplateRejectsExtensionSpoofing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spoofed.pdf")
	if err := os.WriteFile(path, []byte("not a pdf"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateApplicationTemplateFile(path, ".pdf"); err == nil {
		t.Fatal("spoofed PDF extension must be rejected")
	}
}

func TestValidateApplicationTemplateAcceptsUTF8Text(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mau.txt")
	if err := os.WriteFile(path, []byte("Hồ sơ {{company_name}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateApplicationTemplateFile(path, ".txt"); err != nil {
		t.Fatalf("valid UTF-8 template rejected: %v", err)
	}
}

func (f *applicationStoreFake) CreateTemplate(context.Context, string, string, string, string, string) (applicationdomain.Template, error) {
	return applicationdomain.Template{}, nil
}
func (f *applicationStoreFake) Templates(context.Context, string) ([]applicationdomain.Template, error) {
	return []applicationdomain.Template{f.template}, nil
}
func (f *applicationStoreFake) Template(_ context.Context, _, _ string) (applicationdomain.Template, error) {
	return f.template, nil
}
func (f *applicationStoreFake) SaveDraft(_ context.Context, _ string, draft platform.Application) error {
	f.saved = draft
	return nil
}
func (f *applicationStoreFake) Draft(context.Context, string, string) (platform.Application, error) {
	return f.saved, nil
}
func (f *applicationStoreFake) LatestDraft(context.Context, string) (platform.Application, error) {
	return f.saved, nil
}

type applicationGeneratorFake struct {
	request applicationdomain.GenerationRequest
}

func (f *applicationGeneratorFake) GenerateApplication(_ context.Context, request applicationdomain.GenerationRequest) (map[string]string, error) {
	f.request = request
	return map[string]string{"company_overview": "Công ty P2B", "support_need": "Đối chiếu", "proposal": "Hồ sơ từ mẫu"}, nil
}

func TestCreateApplicationUsesSelectedTemplateGeneratorAndCache(t *testing.T) {
	service := platform.NewService([]domain.Policy{{ID: "policy-1", Version: 2, Title: "162/2024/NĐ-CP", Agency: "Chính phủ", Lifecycle: "ACTIVE", TemplateReady: true}})
	workspaceID := "workspace-template-test"
	if _, err := service.BuildPassport(workspaceID, platform.BuildPassportInput{CompanyName: "Công ty P2B"}); err != nil {
		t.Fatal(err)
	}
	checklist, err := service.CreateChecklist(workspaceID, "policy-1")
	if err != nil {
		t.Fatal(err)
	}
	store := &applicationStoreFake{template: applicationdomain.Template{ID: "11111111-1111-1111-1111-111111111111", Name: "Mẫu đã làm", SourceText: "{{company_name}} / {{policy_title}}"}}
	generator := &applicationGeneratorFake{}
	handler := NewServerWithConfig(service, Config{DevAuth: true, WebOrigin: "http://localhost:5173", ApplicationStore: store, ApplicationGenerator: generator})
	body, _ := json.Marshal(map[string]string{"checklist_id": checklist.ID, "template_id": store.template.ID})
	request := httptest.NewRequest(http.MethodPost, "/v1/applications", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "create-application-template")
	request.Header.Set("X-Workspace-ID", workspaceID)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if generator.request.Variables["company_name"] != "Công ty P2B" || generator.request.Variables["policy_title"] != "162/2024/NĐ-CP" {
		t.Fatalf("generation request = %#v", generator.request)
	}
	if store.saved.TemplateID != store.template.ID || store.saved.Sections["proposal"] != "Hồ sơ từ mẫu" {
		t.Fatalf("cached application = %#v", store.saved)
	}
}
