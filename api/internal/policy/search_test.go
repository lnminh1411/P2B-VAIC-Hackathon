package policy

import (
	"math"
	"strings"
	"testing"

	"github.com/p2b/p2b/internal/domain"
)

func TestBuildPassportSearchTextUsesSupportNeedsAndConfirmedFacts(t *testing.T) {
	pass := domain.Passport{
		CompanyName:  "Công ty Xanh",
		SupportNeeds: []string{"đổi mới công nghệ", "vốn ưu đãi"},
		Fields: map[string]domain.PassportField{
			"province": {Label: "Tỉnh thành", Value: "Đà Nẵng", Status: domain.FieldConfirmed},
			"tax_code": {Label: "Mã số thuế", Value: "secret-draft", Status: domain.FieldNeedsReview},
		},
	}

	semantic, lexical := buildPassportSearchText(pass)

	for _, expected := range []string{"query:", "đổi mới công nghệ", "vốn ưu đãi", "Đà Nẵng"} {
		if !strings.Contains(semantic, expected) {
			t.Fatalf("semantic query %q missing %q", semantic, expected)
		}
	}
	if strings.Contains(semantic, "secret-draft") {
		t.Fatalf("semantic query included unconfirmed value: %q", semantic)
	}
	if !strings.Contains(lexical, " OR ") {
		t.Fatalf("lexical query = %q, want websearch OR terms", lexical)
	}
}

func TestFormatVectorRequiresFinite768Dimensions(t *testing.T) {
	vector := make([]float32, embeddingDimensions)
	vector[0] = 0.25
	formatted, err := formatVector(vector)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(formatted, "[0.25,") || !strings.HasSuffix(formatted, "]") {
		t.Fatalf("formatted vector = %q", formatted[:min(len(formatted), 40)])
	}
	if _, err = formatVector(vector[:embeddingDimensions-1]); err == nil {
		t.Fatal("expected wrong dimension error")
	}
	vector[3] = float32(math.Inf(1))
	if _, err = formatVector(vector); err == nil {
		t.Fatal("expected non-finite vector error")
	}
}

func TestSafeSourceURLRejectsNonHTTPSLinks(t *testing.T) {
	if got := safeSourceURL("javascript:alert(1)"); got != "" {
		t.Fatalf("unsafe URL survived: %q", got)
	}
	if got := safeSourceURL("https://vbpl.vn/van-ban/123"); got != "https://vbpl.vn/van-ban/123" {
		t.Fatalf("safe URL = %q", got)
	}
}
