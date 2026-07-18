package extraction

import (
	"fmt"
	"strings"

	passportservice "github.com/p2b/p2b/internal/passport"
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

var canonicalFields = passportservice.CanonicalFieldTypes()

func ValidateCandidates(markdown string, candidates []Candidate) ([]Candidate, []RejectedCandidate) {
	normalizedDocument := normalizeEvidence(markdown)
	valid := make([]Candidate, 0, len(candidates))
	rejected := make([]RejectedCandidate, 0)
	for _, candidate := range candidates {
		expectedType, known := canonicalFields[candidate.FieldKey]
		valueError := passportservice.ValidateFieldValue(candidate.FieldKey, candidate.Value)
		switch {
		case !known:
			rejected = append(rejected, RejectedCandidate{Candidate: candidate, Reason: "unknown field"})
		case candidate.DataType != expectedType:
			rejected = append(rejected, RejectedCandidate{Candidate: candidate, Reason: fmt.Sprintf("unexpected datatype %q", candidate.DataType)})
		case valueError != nil:
			rejected = append(rejected, RejectedCandidate{Candidate: candidate, Reason: valueError.Error()})
		case candidate.Confidence < 0 || candidate.Confidence > 1:
			rejected = append(rejected, RejectedCandidate{Candidate: candidate, Reason: "confidence outside 0..1"})
		case len([]rune(strings.TrimSpace(candidate.Quote))) < 4:
			rejected = append(rejected, RejectedCandidate{Candidate: candidate, Reason: "evidence quote is too short"})
		case !strings.Contains(normalizedDocument, normalizeEvidence(candidate.Quote)):
			rejected = append(rejected, RejectedCandidate{Candidate: candidate, Reason: "evidence quote not found in source"})
		case passportservice.ValidateEvidence(candidate.FieldKey, candidate.Quote) != nil:
			rejected = append(rejected, RejectedCandidate{Candidate: candidate, Reason: "evidence does not match the field concept"})
		default:
			candidate.Quote = strings.TrimSpace(candidate.Quote)
			valid = append(valid, candidate)
		}
	}
	return valid, rejected
}

func normalizeEvidence(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\u00ad", "")
	value = strings.ReplaceAll(value, "-\n", " ")
	return strings.Join(strings.Fields(value), " ")
}
