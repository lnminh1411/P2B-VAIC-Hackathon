package eligibility

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/p2b/p2b/internal/domain"
)

type Status string

const (
	StatusMet         Status = "MET"
	StatusNotMet      Status = "NOT_MET"
	StatusMissingInfo Status = "MISSING_INFO"
)

type CriterionResult struct {
	RuleID      string              `json:"rule_id"`
	FieldKey    string              `json:"field_key"`
	Description string              `json:"description"`
	Status      Status              `json:"status"`
	Observed    any                 `json:"observed,omitempty"`
	Expected    any                 `json:"expected,omitempty"`
	Operator    domain.RuleOperator `json:"operator"`
	Evidence    []domain.Evidence   `json:"evidence"`
	Citation    domain.Evidence     `json:"citation"`
	Required    bool                `json:"required"`
}

type Result struct {
	Status   Status            `json:"status"`
	Criteria []CriterionResult `json:"criteria"`
}

// Evaluate is deterministic: no AI, I/O, clock or confidence-based decisions.
func Evaluate(pass domain.Passport, rules []domain.Rule) Result {
	result := Result{Status: StatusMet, Criteria: make([]CriterionResult, 0, len(rules))}
	for _, rule := range rules {
		criterion := evaluateRule(pass, rule)
		result.Criteria = append(result.Criteria, criterion)
		if !rule.Required {
			continue
		}
		if criterion.Status == StatusNotMet {
			result.Status = StatusNotMet
		} else if criterion.Status == StatusMissingInfo && result.Status != StatusNotMet {
			result.Status = StatusMissingInfo
		}
	}
	return result
}

func evaluateRule(pass domain.Passport, rule domain.Rule) CriterionResult {
	result := CriterionResult{RuleID: rule.ID, FieldKey: rule.FieldKey, Description: rule.Description, Expected: rule.Expected, Operator: rule.Operator, Citation: rule.Citation, Required: rule.Required}
	field, found := pass.Fields[rule.FieldKey]
	if !found || field.Status != domain.FieldConfirmed {
		result.Status = StatusMissingInfo
		return result
	}
	result.Observed = field.Value
	result.Evidence = field.Evidence
	matched, err := compare(field.Value, rule.Operator, rule.Expected)
	if err != nil {
		result.Status = StatusMissingInfo
		return result
	}
	if matched {
		result.Status = StatusMet
	} else {
		result.Status = StatusNotMet
	}
	return result
}

func compare(actual any, operator domain.RuleOperator, expected any) (bool, error) {
	switch operator {
	case domain.OpExists:
		return actual != nil && strings.TrimSpace(fmt.Sprint(actual)) != "", nil
	case domain.OpEQ:
		return strings.EqualFold(strings.TrimSpace(fmt.Sprint(actual)), strings.TrimSpace(fmt.Sprint(expected))), nil
	case domain.OpIN:
		values, ok := expected.([]string)
		if !ok {
			return false, fmt.Errorf("IN expects []string")
		}
		actualText := strings.TrimSpace(fmt.Sprint(actual))
		return slices.ContainsFunc(values, func(value string) bool {
			return strings.EqualFold(strings.TrimSpace(value), actualText)
		}), nil
	case domain.OpContains:
		return strings.Contains(strings.ToLower(fmt.Sprint(actual)), strings.ToLower(fmt.Sprint(expected))), nil
	case domain.OpGT, domain.OpGTE, domain.OpLT, domain.OpLTE:
		return compareNumbers(actual, expected, operator)
	case domain.OpDateBefore, domain.OpDateAfter:
		return compareDates(actual, expected, operator)
	default:
		return reflect.DeepEqual(actual, expected), fmt.Errorf("unsupported operator %q", operator)
	}
}

func compareNumbers(actual, expected any, operator domain.RuleOperator) (bool, error) {
	actualNumber, err := number(actual)
	if err != nil {
		return false, err
	}
	expectedNumber, err := number(expected)
	if err != nil {
		return false, err
	}

	switch operator {
	case domain.OpGT:
		return actualNumber > expectedNumber, nil
	case domain.OpGTE:
		return actualNumber >= expectedNumber, nil
	case domain.OpLT:
		return actualNumber < expectedNumber, nil
	default:
		return actualNumber <= expectedNumber, nil
	}
}

func compareDates(actual, expected any, operator domain.RuleOperator) (bool, error) {
	actualDate, err := time.Parse(time.DateOnly, fmt.Sprint(actual))
	if err != nil {
		return false, err
	}
	expectedDate, err := time.Parse(time.DateOnly, fmt.Sprint(expected))
	if err != nil {
		return false, err
	}
	if operator == domain.OpDateBefore {
		return actualDate.Before(expectedDate), nil
	}
	return actualDate.After(expectedDate), nil
}

func number(value any) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	default:
		return strconv.ParseFloat(strings.ReplaceAll(fmt.Sprint(value), ",", ""), 64)
	}
}
