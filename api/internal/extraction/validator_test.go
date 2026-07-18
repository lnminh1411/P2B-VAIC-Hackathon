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

func TestValidateCandidatesHandlesPDFWordHyphenationInEvidence(t *testing.T) {
	markdown := "Tên doanh nghiệp: Công ty Cổ phần Sao Việt\nVốn điều-\nlệ: 5.000.000.000 đồng"
	candidates := []Candidate{{FieldKey: "charter_capital", Value: float64(5000000000), DataType: "money", Confidence: 0.9, Quote: "Vốn điều lệ: 5.000.000.000 đồng"}}

	valid, rejected := ValidateCandidates(markdown, candidates)
	if len(valid) != 1 || len(rejected) != 0 {
		t.Fatalf("valid = %#v, rejected = %#v", valid, rejected)
	}
}

func TestValidateCandidatesUsesPassportCanonicalFieldKeys(t *testing.T) {
	markdown := "Loại hình doanh nghiệp: Công ty cổ phần"
	candidates := []Candidate{
		{FieldKey: "legal_form", Value: "Công ty cổ phần", DataType: "string", Confidence: .9, Quote: markdown},
		{FieldKey: "company_type", Value: "Công ty cổ phần", DataType: "string", Confidence: .9, Quote: markdown},
	}

	valid, rejected := ValidateCandidates(markdown, candidates)

	if len(valid) != 1 || valid[0].FieldKey != "legal_form" {
		t.Fatalf("valid = %#v, want canonical legal_form", valid)
	}
	if len(rejected) != 1 || rejected[0].Candidate.FieldKey != "company_type" {
		t.Fatalf("rejected = %#v, want legacy company_type rejected", rejected)
	}
}

func TestValidateCandidatesRejectsValueThatDoesNotMatchDeclaredType(t *testing.T) {
	markdown := "Số lao động: 25"
	candidates := []Candidate{{FieldKey: "employee_count", Value: "25", DataType: "integer", Confidence: .9, Quote: markdown}}

	valid, rejected := ValidateCandidates(markdown, candidates)

	if len(valid) != 0 || len(rejected) != 1 {
		t.Fatalf("valid = %#v, rejected = %#v", valid, rejected)
	}
}

func TestValidateCandidatesKeepsCharterCapitalWhenLabelAndValueAreGrounded(t *testing.T) {
	markdown := "Thông tin đăng ký doanh nghiệp\nVốn điều lệ: 5.000.000.000 đồng"
	candidates := []Candidate{{FieldKey: "charter_capital", Value: float64(5000000000), DataType: "money", Confidence: .96, Quote: "Vốn điều lệ: 5.000.000.000 đồng"}}

	valid, rejected := ValidateCandidates(markdown, candidates)
	if len(valid) != 1 || valid[0].FieldKey != "charter_capital" || len(rejected) != 0 {
		t.Fatalf("valid = %#v, rejected = %#v", valid, rejected)
	}
}

func TestValidateCandidatesRejectsBrokerHeadcountAsEmployeeCount(t *testing.T) {
	markdown := "Nhân lực môi giới: 25 người"
	candidates := []Candidate{{FieldKey: "employee_count", Value: float64(25), DataType: "integer", Confidence: .96, Quote: markdown}}

	valid, rejected := ValidateCandidates(markdown, candidates)
	if len(valid) != 0 || len(rejected) != 1 {
		t.Fatalf("valid = %#v, rejected = %#v", valid, rejected)
	}
}
