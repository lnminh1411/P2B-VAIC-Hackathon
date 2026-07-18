package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/p2b/p2b/internal/platform"
)

func TestGoldenPathBuildConfirmMatchAndGenerate(t *testing.T) {
	server := NewServer(platform.NewDemoService())

	build := request(t, server, http.MethodPost, "/v1/passports/build", map[string]any{
		"company_name":  "GreenTech Việt Nam",
		"website":       "https://greentech.example.vn",
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

	confirm := request(t, server, http.MethodPut, "/v1/passport/fields/employee_count", map[string]any{"value": 25, "expected_version": passport.Version})
	if confirm.Code != http.StatusOK {
		t.Fatalf("confirm status = %d: %s", confirm.Code, confirm.Body.String())
	}

	matches := request(t, server, http.MethodPost, "/v1/matches", map[string]any{})
	if matches.Code != http.StatusCreated {
		t.Fatalf("matches status = %d: %s", matches.Code, matches.Body.String())
	}
	var matchResponse struct {
		Results []struct {
			PolicyID string `json:"policy_id"`
		} `json:"results"`
	}
	decode(t, matches, &matchResponse)
	if len(matchResponse.Results) == 0 {
		t.Fatal("expected policy matches")
	}

	checklist := request(t, server, http.MethodPost, "/v1/checklists", map[string]any{"policy_id": matchResponse.Results[0].PolicyID})
	if checklist.Code != http.StatusCreated {
		t.Fatalf("checklist status = %d: %s", checklist.Code, checklist.Body.String())
	}
	var checklistResponse struct {
		ID string `json:"id"`
	}
	decode(t, checklist, &checklistResponse)

	application := request(t, server, http.MethodPost, "/v1/applications", map[string]any{"checklist_id": checklistResponse.ID})
	if application.Code != http.StatusCreated {
		t.Fatalf("application status = %d: %s", application.Code, application.Body.String())
	}
}

func TestRejectsOversizedBodyAndStaleVersion(t *testing.T) {
	server := NewServer(platform.NewDemoService())
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

func request(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
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
	req.Header.Set("Idempotency-Key", "test-"+method+path)
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
