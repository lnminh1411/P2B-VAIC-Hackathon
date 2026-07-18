package extraction

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	applicationdomain "github.com/p2b/p2b/internal/application"
)

func TestGeminiGeneratesStructuredApplicationSections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			SystemInstruction struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"systemInstruction"`
			Contents []struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"contents"`
			GenerationConfig struct {
				ResponseJSONSchema map[string]any `json:"responseJsonSchema"`
			} `json:"generationConfig"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(body.SystemInstruction.Parts[0].Text, "untrusted template") {
			t.Fatalf("system instruction must isolate template content: %#v", body.SystemInstruction)
		}
		if !strings.Contains(body.Contents[0].Parts[0].Text, "Công ty P2B") {
			t.Fatalf("prompt lacks grounded variables: %#v", body.Contents)
		}
		properties := body.GenerationConfig.ResponseJSONSchema["properties"].(map[string]any)
		if _, ok := properties["company_overview"]; !ok {
			t.Fatal("schema lacks company_overview")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"company_overview\":\"Công ty P2B\",\"support_need\":\"Đối chiếu điều kiện\",\"proposal\":\"Chuẩn bị hồ sơ\"}"}]}}]}`))
	}))
	defer server.Close()

	extractor, err := NewGeminiExtractor("test-key", GeminiStableModel, server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	sections, err := extractor.GenerateApplication(context.Background(), applicationdomain.GenerationRequest{
		TemplateText: "Biên bản của {{company_name}}",
		Variables:    map[string]string{"company_name": "Công ty P2B", "policy_title": "162/2024/NĐ-CP"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if sections["company_overview"] != "Công ty P2B" || sections["proposal"] != "Chuẩn bị hồ sơ" {
		t.Fatalf("sections = %#v", sections)
	}
}

func TestGeminiExtractorUsesStableModelAndParsesStructuredCandidates(t *testing.T) {
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if r.Header.Get("x-goog-api-key") != "test-key" {
			t.Fatalf("missing API key header")
		}
		var requestBody struct {
			SystemInstruction struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"systemInstruction"`
			GenerationConfig struct {
				Temperature    *float64 `json:"temperature"`
				ThinkingConfig struct {
					ThinkingLevel string `json:"thinkingLevel"`
				} `json:"thinkingConfig"`
				ResponseJSONSchema map[string]any `json:"responseJsonSchema"`
			} `json:"generationConfig"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if requestBody.GenerationConfig.Temperature != nil {
			t.Fatal("Gemini 3 request must use the model's default temperature")
		}
		if requestBody.GenerationConfig.ThinkingConfig.ThinkingLevel != "high" {
			t.Fatalf("thinking level = %q", requestBody.GenerationConfig.ThinkingConfig.ThinkingLevel)
		}
		if len(requestBody.SystemInstruction.Parts) != 1 || !strings.Contains(requestBody.SystemInstruction.Parts[0].Text, "tax_code | Mã số thuế | string") {
			t.Fatalf("system instruction lacks canonical catalog: %#v", requestBody.SystemInstruction)
		}
		properties := requestBody.GenerationConfig.ResponseJSONSchema["properties"].(map[string]any)
		candidates := properties["candidates"].(map[string]any)
		if _, exists := candidates["maxItems"]; exists {
			t.Fatal("Gemini 3.1 Flash-Lite rejects maxItems on the extraction schema")
		}
		items := candidates["items"].(map[string]any)
		candidateProperties := items["properties"].(map[string]any)
		fieldKey := candidateProperties["field_key"].(map[string]any)
		if !strings.Contains(fieldKey["description"].(string), "canonical") {
			t.Fatalf("field_key description = %#v", fieldKey["description"])
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

func TestGeminiExtractorTargetsOnlyRequestedFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody struct {
			SystemInstruction struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"systemInstruction"`
			GenerationConfig struct {
				ResponseJSONSchema map[string]any `json:"responseJsonSchema"`
			} `json:"generationConfig"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		properties := requestBody.GenerationConfig.ResponseJSONSchema["properties"].(map[string]any)
		items := properties["candidates"].(map[string]any)["items"].(map[string]any)
		fieldKey := items["properties"].(map[string]any)["field_key"].(map[string]any)
		enum := fieldKey["enum"].([]any)
		if len(enum) != 2 || enum[0] != "charter_capital" || enum[1] != "employee_count" {
			t.Fatalf("field enum = %#v", enum)
		}
		instruction := requestBody.SystemInstruction.Parts[0].Text
		if strings.Contains(instruction, "tax_code |") || !strings.Contains(instruction, "charter_capital | Vốn điều lệ | money") {
			t.Fatalf("targeted instruction = %q", instruction)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"candidates\":[]}"}]}}]}`))
	}))
	defer server.Close()

	extractor, err := NewGeminiExtractor("test-key", GeminiStableModel, server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := extractor.ExtractFields(context.Background(), "Vốn điều lệ: 10 tỷ đồng", []string{"charter_capital", "employee_count"}); err != nil {
		t.Fatal(err)
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
