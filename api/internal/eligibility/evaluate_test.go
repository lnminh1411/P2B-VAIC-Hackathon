package eligibility

import (
	"testing"
	"time"

	"github.com/p2b/p2b/internal/domain"
)

func TestEvaluateReturnsMissingForUnconfirmedFact(t *testing.T) {
	pass := domain.Passport{Fields: map[string]domain.PassportField{
		"employee_count": {Key: "employee_count", Value: 18, Status: domain.FieldExtracted},
	}}
	rule := domain.Rule{ID: "r1", FieldKey: "employee_count", Operator: domain.OpLTE, Expected: 50, Required: true}

	got := Evaluate(pass, []domain.Rule{rule})
	if got.Status != StatusMissingInfo {
		t.Fatalf("status = %s, want %s", got.Status, StatusMissingInfo)
	}
}

func TestEvaluateDistinguishesMetAndNotMet(t *testing.T) {
	pass := domain.Passport{Fields: map[string]domain.PassportField{
		"province":       {Key: "province", Value: "Hồ Chí Minh", Status: domain.FieldConfirmed},
		"employee_count": {Key: "employee_count", Value: 75, Status: domain.FieldConfirmed},
	}}
	rules := []domain.Rule{
		{ID: "r1", FieldKey: "province", Operator: domain.OpIN, Expected: []string{"Hà Nội", "Hồ Chí Minh"}, Required: true},
		{ID: "r2", FieldKey: "employee_count", Operator: domain.OpLTE, Expected: 50, Required: true},
	}

	got := Evaluate(pass, rules)
	if got.Status != StatusNotMet {
		t.Fatalf("status = %s, want %s", got.Status, StatusNotMet)
	}
	if got.Criteria[0].Status != StatusMet || got.Criteria[1].Status != StatusNotMet {
		t.Fatalf("unexpected criteria: %#v", got.Criteria)
	}
}

func TestEvaluateSupportsDatesAndOptionalRules(t *testing.T) {
	pass := domain.Passport{Fields: map[string]domain.PassportField{
		"incorporation_date": {Key: "incorporation_date", Value: "2022-05-01", Status: domain.FieldConfirmed},
	}}
	rules := []domain.Rule{
		{ID: "r1", FieldKey: "incorporation_date", Operator: domain.OpDateAfter, Expected: "2020-01-01", Required: true},
		{ID: "r2", FieldKey: "revenue", Operator: domain.OpGTE, Expected: 1_000_000_000, Required: false},
	}

	got := Evaluate(pass, rules)
	if got.Status != StatusMet {
		t.Fatalf("status = %s, want %s at %s", got.Status, StatusMet, time.Now())
	}
}
