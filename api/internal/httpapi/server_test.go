package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/p2b/p2b/internal/authn"
	"github.com/p2b/p2b/internal/domain"
	"github.com/p2b/p2b/internal/platform"
)

type verifierStub struct {
	principal authn.Principal
	err       error
}

type readinessStub struct {
	err error
}

func (r readinessStub) Ping(_ context.Context) error {
	return r.err
}

func (v verifierStub) Verify(_ context.Context, _ string) (authn.Principal, error) {
	return v.principal, v.err
}

func TestProductionAuthDerivesWorkspaceFromVerifiedPrincipal(t *testing.T) {
	principal := authn.Principal{Subject: "0f34fe4f-37dc-43c0-9277-67e49b7b06b5", Email: "founder@example.vn", Name: "Nguyễn Minh Anh"}
	server := NewServerWithConfig(newTestService(), Config{
		WebOrigin: "https://app.p2b.vn",
		Verifier:  verifierStub{principal: principal},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer verified-token")
	req.Header.Set("X-Workspace-ID", "attacker-controlled-workspace")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("auth me status = %d: %s", response.Code, response.Body.String())
	}
	var me authn.Principal
	decode(t, response, &me)
	if me.Subject != principal.Subject || me.Email != principal.Email {
		t.Fatalf("principal = %#v, want %#v", me, principal)
	}

	passportReq := httptest.NewRequest(http.MethodGet, "/v1/passport", nil)
	passportReq.Header.Set("Authorization", "Bearer verified-token")
	passportReq.Header.Set("X-Workspace-ID", "attacker-controlled-workspace")
	passportResponse := httptest.NewRecorder()
	server.ServeHTTP(passportResponse, passportReq)
	var passport struct {
		WorkspaceID string `json:"workspace_id"`
	}
	decode(t, passportResponse, &passport)
	if passport.WorkspaceID != principal.Subject {
		t.Fatalf("workspace = %q, want verified subject %q", passport.WorkspaceID, principal.Subject)
	}
}

func TestAdminPoliciesRequiresTrustedAdminRole(t *testing.T) {
	for _, test := range []struct {
		name       string
		roles      []string
		wantStatus int
	}{
		{name: "member denied", wantStatus: http.StatusForbidden},
		{name: "admin allowed", roles: []string{"admin"}, wantStatus: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			principal := authn.Principal{Subject: "0f34fe4f-37dc-43c0-9277-67e49b7b06b5", Email: "founder@example.vn", Roles: test.roles}
			server := NewServerWithConfig(newTestService(), Config{Verifier: verifierStub{principal: principal}})
			request := httptest.NewRequest(http.MethodGet, "/v1/admin/policies", nil)
			request.Header.Set("Authorization", "Bearer verified-token")
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestReadinessReflectsDatabaseHealth(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{name: "database ready", wantStatus: http.StatusOK, wantBody: `"status":"ready"`},
		{name: "database unavailable", err: errors.New("connection refused"), wantStatus: http.StatusServiceUnavailable, wantBody: `"status":"not_ready"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := NewServerWithConfig(newTestService(), Config{ReadinessChecker: readinessStub{err: test.err}})
			request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.wantStatus, response.Body.String())
			}
			if !bytes.Contains(response.Body.Bytes(), []byte(test.wantBody)) {
				t.Fatalf("body = %s, want to contain %s", response.Body.String(), test.wantBody)
			}
		})
	}
}

func TestGoldenPathBuildConfirmMatchAndGenerate(t *testing.T) {
	server := NewServer(newTestService())

	build := request(t, server, http.MethodPost, "/v1/passports/build", map[string]any{
		"company_name":  "Công ty kiểm thử P2B",
		"website":       "https://company.example.vn",
		"support_needs": []string{"công nghệ xanh", "vốn"},
		"source_names":  []string{"dang-ky-doanh-nghiep.pdf", "pitch-deck.pdf"},
	})
	if build.Code != http.StatusAccepted {
		t.Fatalf("build status = %d: %s", build.Code, build.Body.String())
	}

	passportResponse := request(t, server, http.MethodGet, "/v1/passport", nil)
	var passport struct {
		Version int `json:"version"`
	}
	decode(t, passportResponse, &passport)
	if passport.Version < 1 {
		t.Fatalf("passport version = %d", passport.Version)
	}

	candidatesResponse := request(t, server, http.MethodGet, "/v1/passport/candidates", nil)
	var candidates struct {
		Candidates []struct {
			FieldKey string `json:"field_key"`
			Value    any    `json:"value"`
		} `json:"candidates"`
	}
	decode(t, candidatesResponse, &candidates)
	for _, candidate := range candidates.Candidates {
		confirm := request(t, server, http.MethodPut, "/v1/passport/fields/"+candidate.FieldKey, map[string]any{"value": candidate.Value, "expected_version": passport.Version})
		if confirm.Code != http.StatusOK {
			t.Fatalf("confirm %s status = %d: %s", candidate.FieldKey, confirm.Code, confirm.Body.String())
		}
		decode(t, confirm, &passport)
	}

	matches := request(t, server, http.MethodPost, "/v1/matches", map[string]any{})
	if matches.Code != http.StatusCreated {
		t.Fatalf("matches status = %d: %s", matches.Code, matches.Body.String())
	}
	var matchResponse struct {
		Results []struct {
			PolicyID      string `json:"policy_id"`
			TemplateReady bool   `json:"template_ready"`
		} `json:"results"`
	}
	decode(t, matches, &matchResponse)
	if len(matchResponse.Results) == 0 {
		t.Fatal("expected policy matches")
	}

	policyID := ""
	for _, result := range matchResponse.Results {
		if result.PolicyID == "test-ready" && result.TemplateReady {
			policyID = result.PolicyID
		}
	}
	if policyID == "" {
		t.Fatal("expected reviewed green policy with an active template")
	}

	checklist := request(t, server, http.MethodPost, "/v1/checklists", map[string]any{"policy_id": policyID})
	if checklist.Code != http.StatusCreated {
		t.Fatalf("checklist status = %d: %s", checklist.Code, checklist.Body.String())
	}
	var checklistResponse struct {
		ID    string `json:"id"`
		Items []struct {
			Status string `json:"status"`
		} `json:"items"`
	}
	decode(t, checklist, &checklistResponse)
	for _, item := range checklistResponse.Items {
		if item.Status != "AVAILABLE" {
			t.Fatalf("golden-path checklist item status = %s", item.Status)
		}
	}

	application := request(t, server, http.MethodPost, "/v1/applications", map[string]any{"checklist_id": checklistResponse.ID})
	if application.Code != http.StatusCreated {
		t.Fatalf("application status = %d: %s", application.Code, application.Body.String())
	}
	var applicationResponse struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	decode(t, application, &applicationResponse)
	for _, action := range []string{"submit", "approve", "generate"} {
		transition := request(t, server, http.MethodPost, "/v1/applications/"+applicationResponse.ID+"/"+action, map[string]any{})
		if transition.Code != http.StatusOK {
			t.Fatalf("%s status = %d: %s", action, transition.Code, transition.Body.String())
		}
		decode(t, transition, &applicationResponse)
	}
	if applicationResponse.Status != "GENERATED" {
		t.Fatalf("application status = %s, want GENERATED", applicationResponse.Status)
	}

	pdf := request(t, server, http.MethodGet, "/v1/applications/"+applicationResponse.ID+"/download", nil)
	if pdf.Code != http.StatusOK || pdf.Header().Get("Content-Type") != "application/pdf" || !bytes.HasPrefix(pdf.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("invalid generated PDF: status=%d content-type=%s body=%q", pdf.Code, pdf.Header().Get("Content-Type"), pdf.Body.Bytes())
	}
}

func TestRejectsOversizedBodyAndStaleVersion(t *testing.T) {
	server := NewServer(newTestService())
	request(t, server, http.MethodPost, "/v1/passports/build", map[string]any{"company_name": "P2B", "support_needs": []string{"vốn"}})

	response := request(t, server, http.MethodPut, "/v1/passport/fields/legal_name", map[string]any{"value": "Tên mới", "expected_version": 999})
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", response.Code)
	}

	large := bytes.NewBufferString(`{"company_name":"` + string(bytes.Repeat([]byte("a"), maxBodyBytes+1)) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/passports/build", large)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", res.Code)
	}
}

func TestPresignFailsClosedWithoutStorageSigner(t *testing.T) {
	server := NewServer(platform.NewService(nil))
	response := request(t, server, http.MethodPost, "/v1/uploads/presign", map[string]any{
		"filename": "company.pdf", "content_type": "application/pdf", "size_bytes": 128,
	})
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", response.Code, response.Body.String())
	}
}

func TestIdempotencyReplaysResponseAndRejectsKeyReuse(t *testing.T) {
	server := NewServer(newTestService())
	body := map[string]any{"company_name": "P2B Idempotent"}

	first := requestWithKey(t, server, http.MethodPost, "/v1/passports/build", body, "build-once")
	second := requestWithKey(t, server, http.MethodPost, "/v1/passports/build", body, "build-once")
	if first.Code != http.StatusAccepted || second.Code != http.StatusAccepted {
		t.Fatalf("statuses = %d, %d", first.Code, second.Code)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("idempotent response changed:\nfirst: %s\nsecond: %s", first.Body.String(), second.Body.String())
	}

	reused := requestWithKey(t, server, http.MethodPost, "/v1/passports/build", map[string]any{"company_name": "Different request"}, "build-once")
	if reused.Code != http.StatusConflict {
		t.Fatalf("reused key status = %d, want 409: %s", reused.Code, reused.Body.String())
	}
}

func request(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	return requestWithKey(t, handler, method, path, body, "test-"+method+path)
}

func requestWithKey(t *testing.T, handler http.Handler, method, path string, body any, idempotencyKey string) *httptest.ResponseRecorder {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &payload)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-ID", "workspace-test")
	req.Header.Set("Idempotency-Key", idempotencyKey)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func decode(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode: %v: %s", err, response.Body.String())
	}
}

func newTestService() *platform.Service {
	now := time.Now().UTC()
	policy := domain.Policy{
		ID: "test-ready", Version: 1, Title: "Chính sách kiểm thử", Agency: "Cơ quan kiểm thử",
		Benefit: "Hỗ trợ kiểm thử", SupportType: "Công nghệ xanh", Lifecycle: "ACTIVE", VerifiedAt: now, Deadline: now.AddDate(0, 1, 0), TemplateReady: true,
		Rules: []domain.Rule{
			{ID: "legal-name", FieldKey: "legal_name", Operator: domain.OpExists, Required: true},
			{ID: "website", FieldKey: "website", Operator: domain.OpExists, Required: true},
		},
		Checklist: []domain.ChecklistTemplateItem{{Key: "registration", Title: "Hồ sơ doanh nghiệp", Required: true, FieldKeys: []string{"legal_name", "website"}}},
	}
	return platform.NewService([]domain.Policy{policy})
}
