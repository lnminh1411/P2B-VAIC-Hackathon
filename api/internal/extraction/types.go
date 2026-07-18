package extraction

import (
	"fmt"
	"strings"
)

type Candidate struct {
	FieldKey   string  `json:"field_key"`
	Value      any     `json:"value"`
	DataType   string  `json:"data_type"`
	Confidence float64 `json:"confidence"`
	Quote      string  `json:"quote"`
}

type RejectedCandidate struct {
	Candidate Candidate
	Reason    string
}

var canonicalFields = map[string]string{
	"legal_name":             "string",
	"tax_code":               "string",
	"company_type":           "string",
	"founded_date":           "date",
	"operating_status":       "string",
	"charter_capital":        "money",
	"revenue":                "money",
	"assets":                 "money",
	"employee_count":         "integer",
	"address":                "string",
	"province":               "string",
	"industrial_zone":        "string",
	"business_sectors":       "string_array",
	"products":               "string_array",
	"technologies":           "string_array",
	"markets":                "string_array",
	"fdi_status":             "boolean",
	"foreign_ownership_rate": "number",
	"women_owned":            "boolean",
	"rd_capability":          "string",
	"intellectual_property":  "string_array",
	"certifications":         "string_array",
	"innovation_projects":    "string_array",
	"green_projects":         "string_array",
	"funding_need":           "money",
	"funding_use_plan":       "string",
}

func ValidateCandidates(markdown string, candidates []Candidate) ([]Candidate, []RejectedCandidate) {
	normalizedDocument := normalizeEvidence(markdown)
	valid := make([]Candidate, 0, len(candidates))
	rejected := make([]RejectedCandidate, 0)
	for _, candidate := range candidates {
		expectedType, known := canonicalFields[candidate.FieldKey]
		switch {
		case !known:
			rejected = append(rejected, RejectedCandidate{Candidate: candidate, Reason: "unknown field"})
		case candidate.DataType != expectedType:
			rejected = append(rejected, RejectedCandidate{Candidate: candidate, Reason: fmt.Sprintf("unexpected datatype %q", candidate.DataType)})
		case candidate.Confidence < 0 || candidate.Confidence > 1:
			rejected = append(rejected, RejectedCandidate{Candidate: candidate, Reason: "confidence outside 0..1"})
		case len([]rune(strings.TrimSpace(candidate.Quote))) < 4:
			rejected = append(rejected, RejectedCandidate{Candidate: candidate, Reason: "evidence quote is too short"})
		case !strings.Contains(normalizedDocument, normalizeEvidence(candidate.Quote)):
			rejected = append(rejected, RejectedCandidate{Candidate: candidate, Reason: "evidence quote not found in source"})
		default:
			candidate.Quote = strings.TrimSpace(candidate.Quote)
			valid = append(valid, candidate)
		}
	}
	return valid, rejected
}

func normalizeEvidence(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
