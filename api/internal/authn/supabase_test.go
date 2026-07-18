package authn

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSupabaseVerifierReturnsVerifiedPrincipal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/v1/user" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer access-token" || r.Header.Get("apikey") != "publishable-key" {
			t.Fatalf("missing Supabase auth headers")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"0f34fe4f-37dc-43c0-9277-67e49b7b06b5","email":"founder@greentech.vn","app_metadata":{"roles":["admin"]},"user_metadata":{"full_name":"Nguyễn Minh Anh","avatar_url":"https://example.com/avatar.png","roles":["owner"]}}`))
	}))
	defer server.Close()

	verifier, err := NewSupabaseVerifier(server.URL, "publishable-key", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	principal, err := verifier.Verify(t.Context(), "access-token")
	if err != nil {
		t.Fatal(err)
	}
	if principal.Subject != "0f34fe4f-37dc-43c0-9277-67e49b7b06b5" || principal.Email != "founder@greentech.vn" || principal.Name != "Nguyễn Minh Anh" {
		t.Fatalf("principal = %#v", principal)
	}
	if !principal.HasRole("admin") || principal.HasRole("owner") {
		t.Fatalf("trusted roles = %#v; user_metadata role must be ignored", principal.Roles)
	}
}

func TestSupabaseVerifierRejectsInvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	verifier, err := NewSupabaseVerifier(server.URL, "publishable-key", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = verifier.Verify(t.Context(), "expired-token")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("error = %v, want ErrInvalidToken", err)
	}
}

func TestSupabaseVerifierRejectsInvalidConfiguration(t *testing.T) {
	for _, test := range []struct{ url, key string }{{"", "key"}, {"https://example.com", ""}, {"http://example.com", "key"}} {
		if _, err := NewSupabaseVerifier(test.url, test.key, nil); err == nil {
			t.Fatalf("NewSupabaseVerifier(%q, %q) succeeded", test.url, test.key)
		}
	}
}
