package platform

import (
	"testing"

	"github.com/p2b/p2b/internal/domain"
)

func TestApplicationBlockersReturnsEmptyJSONCollection(t *testing.T) {
	blockers := applicationBlockers(Checklist{}, true)
	if blockers == nil {
		t.Fatal("application blockers must be an empty slice, not nil")
	}
	if len(blockers) != 0 {
		t.Fatalf("expected no blockers, got %v", blockers)
	}
}

func TestCreateChecklistRequiresEveryReferencedField(t *testing.T) {
	service := NewDemoService()
	workspaceID := "checklist-all-fields"
	_, err := service.BuildPassport(workspaceID, BuildPassportInput{CompanyName: "Công ty kiểm thử"})
	if err != nil {
		t.Fatalf("build passport: %v", err)
	}

	var policyID string
	for _, policy := range service.Policies(true) {
		for _, item := range policy.Checklist {
			if item.Key == "registration" {
				policyID = policy.ID
				break
			}
		}
	}
	if policyID == "" {
		t.Fatal("demo policy with registration checklist not found")
	}

	checklist, err := service.CreateChecklist(workspaceID, policyID)
	if err != nil {
		t.Fatalf("create checklist: %v", err)
	}
	for _, item := range checklist.Items {
		if item.TemplateKey == "registration" {
			if item.Status != "MISSING" {
				t.Fatalf("expected MISSING while tax_code is %s, got %s", domain.FieldExtracted, item.Status)
			}
			return
		}
	}
	t.Fatal("registration checklist item not found")
}
