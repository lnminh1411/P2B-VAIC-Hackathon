package passport

import (
	"testing"
	"time"

	"github.com/p2b/p2b/internal/domain"
)

func TestMergeCandidateRequiresEvidenceAndDetectsConflict(t *testing.T) {
	now := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	pass := domain.Passport{Version: 1, Fields: map[string]domain.PassportField{
		"tax_code": {Key: "tax_code", Value: "0312345678", Status: domain.FieldConfirmed},
	}}
	candidate := Candidate{FieldKey: "tax_code", Value: "0312345679", DataType: "string", Confidence: .94, Evidence: domain.Evidence{
		SourceID: "source-1", Quote: "Mã số doanh nghiệp: 0312345679", ContentHash: "sha256:test", ObservedAt: now,
	}}

	updated, err := MergeCandidate(pass, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Fields["tax_code"].Status != domain.FieldConflicted {
		t.Fatalf("status = %s, want CONFLICTED", updated.Fields["tax_code"].Status)
	}
}

func TestMergeCandidateRejectsUnknownFieldAndMissingQuote(t *testing.T) {
	pass := domain.Passport{Version: 1, Fields: map[string]domain.PassportField{}}
	_, err := MergeCandidate(pass, Candidate{FieldKey: "invented", Value: "x"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestConfirmFieldCreatesNewVersion(t *testing.T) {
	pass := domain.Passport{Version: 3, Fields: map[string]domain.PassportField{
		"legal_name": {Key: "legal_name", Value: "Công ty P2B", Status: domain.FieldExtracted},
	}}
	updated, err := ConfirmField(pass, "legal_name", "Công ty Cổ phần P2B", 3)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 4 || updated.Fields["legal_name"].Status != domain.FieldConfirmed {
		t.Fatalf("unexpected passport: %#v", updated)
	}
}
