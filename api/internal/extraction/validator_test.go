package extraction

import "testing"

func TestValidateCandidatesRequiresExactEvidenceAndKnownField(t *testing.T) {
	markdown := "# Giấy chứng nhận\nMã số doanh nghiệp: 0123456789\nVốn điều lệ: 5.000.000.000 đồng"
	candidates := []Candidate{
		{FieldKey: "tax_code", Value: "0123456789", DataType: "string", Confidence: 0.98, Quote: "Mã số doanh nghiệp: 0123456789"},
		{FieldKey: "employee_count", Value: float64(25), DataType: "integer", Confidence: 0.8, Quote: "Số lao động: 25"},
		{FieldKey: "invented_field", Value: "x", DataType: "string", Confidence: 1, Quote: "Giấy chứng nhận"},
	}

	valid, rejected := ValidateCandidates(markdown, candidates)

	if len(valid) != 1 || valid[0].FieldKey != "tax_code" {
		t.Fatalf("valid = %#v, want only evidence-backed tax_code", valid)
	}
	if len(rejected) != 2 {
		t.Fatalf("rejected = %d, want 2", len(rejected))
	}
}

func TestValidateCandidatesNormalizesWhitespaceWithoutInventingText(t *testing.T) {
	markdown := "Tên doanh nghiệp:\n  Công ty Cổ phần Sao Việt"
	candidates := []Candidate{{FieldKey: "legal_name", Value: "Công ty Cổ phần Sao Việt", DataType: "string", Confidence: 0.9, Quote: "Tên doanh nghiệp: Công ty Cổ phần Sao Việt"}}

	valid, _ := ValidateCandidates(markdown, candidates)
	if len(valid) != 1 {
		t.Fatalf("valid = %#v, want whitespace-normalized evidence match", valid)
	}
}
