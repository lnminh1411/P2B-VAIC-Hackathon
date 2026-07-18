package application

import (
	"reflect"
	"strings"
	"testing"

	"github.com/p2b/p2b/internal/domain"
)

func TestExtractPlaceholdersDeduplicatesAndSorts(t *testing.T) {
	got := ExtractPlaceholders("{{ policy_title }} / {{company_name}} / {{policy_title}}")
	want := []string{"company_name", "policy_title"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("placeholders = %#v, want %#v", got, want)
	}
}

func TestRenderTemplateGroundsKnownValuesAndMarksUnknowns(t *testing.T) {
	got := RenderTemplate("Doanh nghiệp {{company_name}} - {{unknown_field}}", map[string]string{"company_name": "Công ty P2B"})
	if !strings.Contains(got, "Công ty P2B") || !strings.Contains(got, "[CẦN BỔ SUNG: unknown_field]") {
		t.Fatalf("rendered template = %q", got)
	}
}

func TestTemplateVariablesUseConfirmedPassportAndPolicyFacts(t *testing.T) {
	passport := domain.Passport{CompanyName: "Công ty P2B", Website: "https://p2b.vn", Fields: map[string]domain.PassportField{
		"tax_code":   {Value: "0123456789", Status: domain.FieldConfirmed},
		"draft_fact": {Value: "không dùng", Status: domain.FieldExtracted},
	}}
	policy := domain.Policy{Title: "162/2024/NĐ-CP", Agency: "Chính phủ", SourceURL: "https://vbpl.vn/example"}
	variables := TemplateVariables(passport, policy)
	if variables["company_name"] != "Công ty P2B" || variables["tax_code"] != "0123456789" || variables["policy_title"] != policy.Title {
		t.Fatalf("variables = %#v", variables)
	}
	if _, exists := variables["draft_fact"]; exists {
		t.Fatal("unconfirmed passport fact must not be sent to application generation")
	}
}
