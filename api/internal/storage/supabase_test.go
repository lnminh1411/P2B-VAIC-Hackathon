package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSupabaseSignerCreatesPrivateUploadURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/object/upload/sign/p2b-private/workspace/source.pdf") {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret-key" || r.Header.Get("apikey") != "secret-key" {
			t.Fatal("missing server credentials")
		}
		_, _ = w.Write([]byte(`{"url":"/object/upload/sign/p2b-private/workspace/source.pdf?token=signed-token"}`))
	}))
	defer server.Close()

	signer, err := NewSupabaseSigner(server.URL, "secret-key", "p2b-private", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	url, err := signer.CreateUploadURL(t.Context(), "workspace/source.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if url != server.URL+"/storage/v1/object/upload/sign/p2b-private/workspace/source.pdf?token=signed-token" {
		t.Fatalf("url = %q", url)
	}
}

func TestSupabaseSignerDownloadsPrivateObjectWithLimit(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/storage/v1/object/authenticated/private/workspace/sources/source.pdf" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatal("missing service authorization")
		}
		_, _ = w.Write([]byte("%PDF-1.7\ncontent"))
	}))
	defer server.Close()

	signer, err := NewSupabaseSigner(server.URL, "secret", "private", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	content, err := signer.Download(context.Background(), "workspace/sources/source.pdf", 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(content[:5]) != "%PDF-" {
		t.Fatalf("content = %q", content)
	}
}

func TestSupabaseSignerRejectsUnsafeObjectKey(t *testing.T) {
	signer, err := NewSupabaseSigner("https://example.supabase.co", "secret-key", "p2b-private", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"", "../secret.pdf", "/absolute.pdf", "folder/not pdf.exe"} {
		if _, err := signer.CreateUploadURL(t.Context(), key); err == nil {
			t.Fatalf("key %q accepted", key)
		}
	}
}
