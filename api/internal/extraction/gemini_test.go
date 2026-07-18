package extraction

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiExtractorUsesStableModelAndParsesStructuredCandidates(t *testing.T) {
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if r.Header.Get("x-goog-api-key") != "test-key" {
			t.Fatalf("missing API key header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"candidates\":[{\"field_key\":\"tax_code\",\"value\":\"0123456789\",\"data_type\":\"string\",\"confidence\":0.97,\"quote\":\"Mã số doanh nghiệp: 0123456789\"}]}"}]}}]}`))
	}))
	defer server.Close()

	extractor, err := NewGeminiExtractor("test-key", GeminiStableModel, server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	result, err := extractor.Extract(context.Background(), "Mã số doanh nghiệp: 0123456789")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(requestPath, "/models/"+GeminiStableModel+":generateContent") {
		t.Fatalf("request path = %q", requestPath)
	}
	if len(result) != 1 || result[0].FieldKey != "tax_code" {
		t.Fatalf("result = %#v", result)
	}
}

func TestGeminiExtractorRejectsMissingKey(t *testing.T) {
	if _, err := NewGeminiExtractor("", GeminiStableModel, "https://generativelanguage.googleapis.com", nil); err == nil {
		t.Fatal("expected missing key error")
	}
}
