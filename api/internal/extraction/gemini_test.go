package extraction

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
		var requestBody struct {
			GenerationConfig struct {
				Temperature        *float64       `json:"temperature"`
				ResponseJSONSchema map[string]any `json:"responseJsonSchema"`
			} `json:"generationConfig"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if requestBody.GenerationConfig.Temperature != nil {
			t.Fatal("Gemini 3 request must use the model's default temperature")
		}
		properties := requestBody.GenerationConfig.ResponseJSONSchema["properties"].(map[string]any)
		candidates := properties["candidates"].(map[string]any)
		if _, exists := candidates["maxItems"]; exists {
			t.Fatal("Gemini 3.1 Flash-Lite rejects maxItems on the extraction schema")
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

func TestGeminiExtractorRejectsTooManyCandidates(t *testing.T) {
	candidates := make([]Candidate, maxGeminiCandidates+1)
	for index := range candidates {
		candidates[index] = Candidate{FieldKey: "tax_code", Value: "0301955155", DataType: "string", Confidence: 1, Quote: "Mã số doanh nghiệp: 0301955155"}
	}
	structured, err := json.Marshal(map[string]any{"candidates": candidates})
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := json.Marshal(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"text": string(structured)}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(envelope)
	}))
	defer server.Close()

	extractor, err := NewGeminiExtractor("test-key", GeminiStableModel, server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = extractor.Extract(context.Background(), "Mã số doanh nghiệp: 0301955155")
	if err == nil || !strings.Contains(err.Error(), "more than 100 candidates") {
		t.Fatalf("error = %v", err)
	}
}

func TestGeminiExtractorLiveSchema(t *testing.T) {
	if os.Getenv("GEMINI_LIVE_TEST") != "1" {
		t.Skip("set GEMINI_LIVE_TEST=1 to call the live Gemini API")
	}
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY is not set")
	}
	extractor, err := NewGeminiExtractor(apiKey, GeminiStableModel, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := extractor.Extract(context.Background(), "Mã số doanh nghiệp: 0301955155")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].FieldKey != "tax_code" {
		t.Fatalf("result = %#v", result)
	}
}
