package platform

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/p2b/p2b/internal/domain"
)

func (s *Service) CreateChecklist(workspaceID, policyID string) (Checklist, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	policy, ok := s.findWorkspacePolicy(state, policyID)
	if !ok || policy.Lifecycle != "ACTIVE" {
		return Checklist{}, ErrNotFound
	}
	checklist := Checklist{ID: uuid.NewString(), PolicyID: policy.ID, PolicyVersion: policy.Version, Version: 1, UpdatedAt: time.Now().UTC()}
	for _, template := range policy.Checklist {
		fieldKeys := append([]string{}, template.FieldKeys...)
		status := "AVAILABLE"
		if len(fieldKeys) == 0 {
			status = "MISSING"
		}
		for _, fieldKey := range fieldKeys {
			if field, exists := state.Passport.Fields[fieldKey]; !exists || field.Status != domain.FieldConfirmed {
				status = "MISSING"
				break
			}
		}
		checklist.Items = append(checklist.Items, ChecklistItem{ID: uuid.NewString(), TemplateKey: template.Key, Title: template.Title, Description: template.Description, Required: template.Required, Status: status, FieldKeys: fieldKeys})
	}
	state.Checklists[checklist.ID] = checklist
	return checklist, nil
}

func (s *Service) Checklist(workspaceID, id string) (Checklist, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	checklist, ok := s.workspace(workspaceID).Checklists[id]
	if !ok {
		return Checklist{}, ErrNotFound
	}
	return checklist, nil
}

func (s *Service) UpdateChecklistItem(workspaceID, checklistID, itemID, status, evidence string, expectedVersion int) (Checklist, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	checklist, ok := state.Checklists[checklistID]
	if !ok {
		return Checklist{}, ErrNotFound
	}
	if checklist.Version != expectedVersion {
		return Checklist{}, ErrConflict
	}
	if status != "AVAILABLE" && status != "MISSING" && status != "NEEDS_REVIEW" && status != "NOT_APPLICABLE" {
		return Checklist{}, errors.New("invalid checklist item status")
	}
	for index := range checklist.Items {
		if checklist.Items[index].ID != itemID {
			continue
		}
		if status == "AVAILABLE" && strings.TrimSpace(evidence) == "" {
			return Checklist{}, errors.New("AVAILABLE requires an evidence source")
		}
		checklist.Items[index].Status = status
		checklist.Items[index].EvidenceSource = strings.TrimSpace(evidence)
		checklist.Version++
		checklist.UpdatedAt = time.Now().UTC()
		state.Checklists[checklistID] = checklist
		return checklist, nil
	}
	return Checklist{}, ErrNotFound
}

func (s *Service) CreateApplication(workspaceID, checklistID string) (Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	checklist, ok := state.Checklists[checklistID]
	if !ok {
		return Application{}, ErrNotFound
	}
	policy, ok := s.findWorkspacePolicy(state, checklist.PolicyID)
	if !ok {
		return Application{}, ErrNotFound
	}
	sections := map[string]string{
		"company_overview": fmt.Sprintf("%s đề nghị tham gia %s.", state.Passport.CompanyName, policy.Title),
		"support_need":     strings.Join(state.Passport.SupportNeeds, ", "),
		"proposal":         "Doanh nghiệp sẽ sử dụng nguồn hỗ trợ để hoàn thiện sản phẩm, đo lường tác động và mở rộng thị trường.",
	}
	if isRetrievedWorkingPolicy(policy) {
		sections = map[string]string{
			"company_overview": companyOverview(state.Passport),
			"support_need":     fmt.Sprintf("Đối chiếu phạm vi áp dụng, điều kiện, ngoại lệ, hiệu lực và tài liệu bắt buộc của văn bản %s.", policy.Title),
			"proposal":         fmt.Sprintf("Doanh nghiệp và người phụ trách pháp chế thực hiện đối chiếu văn bản %s; kết quả là cơ sở rà soát nội bộ, không phải kết luận đủ điều kiện hoặc mẫu hồ sơ chính thức.", policy.Title),
		}
	}
	application := Application{ID: uuid.NewString(), ChecklistID: checklistID, PolicyID: policy.ID, PassportVersion: state.Passport.Version, PolicyVersion: policy.Version, TemplateVersion: 1, Version: 1, Status: "DRAFT_READY", UpdatedAt: time.Now().UTC(), Sections: sections}
	application.BlockingReasons = applicationBlockers(checklist, policy.TemplateReady)
	state.Applications[application.ID] = application
	return application, nil
}

func (s *Service) Application(workspaceID, id string) (Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	application, ok := s.workspace(workspaceID).Applications[id]
	if !ok {
		return Application{}, ErrNotFound
	}
	return application, nil
}

func (s *Service) UpdateApplication(workspaceID, id string, sections map[string]string, expectedVersion int) (Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	application, ok := state.Applications[id]
	if !ok {
		return Application{}, ErrNotFound
	}
	if application.Version != expectedVersion {
		return Application{}, ErrConflict
	}
	if application.Status != "DRAFT_READY" && application.Status != "REJECTED" {
		return Application{}, ErrConflict
	}
	for key, value := range sections {
		if len(key) > 80 || len(value) > 10_000 {
			return Application{}, errors.New("application section exceeds allowed size")
		}
		application.Sections[key] = strings.TrimSpace(value)
	}
	application.Version++
	application.UpdatedAt = time.Now().UTC()
	state.Applications[id] = application
	return application, nil
}

func (s *Service) TransitionApplication(workspaceID, id, action string) (Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	application, ok := state.Applications[id]
	if !ok {
		return Application{}, ErrNotFound
	}
	checklist := state.Checklists[application.ChecklistID]
	policy, ok := s.findWorkspacePolicy(state, application.PolicyID)
	if !ok {
		return Application{}, ErrNotFound
	}
	switch action {
	case "submit":
		if application.Status != "DRAFT_READY" {
			return Application{}, ErrConflict
		}
		application.Status = "PENDING_REVIEW"
	case "approve":
		if application.Status != "PENDING_REVIEW" {
			return Application{}, ErrConflict
		}
		application.BlockingReasons = applicationBlockers(checklist, policy.TemplateReady)
		if len(application.BlockingReasons) > 0 {
			state.Applications[id] = application
			return application, ErrBlocked
		}
		application.Status = "APPROVED"
	case "generate":
		if application.Status != "APPROVED" {
			return Application{}, ErrConflict
		}
		application.Status = "GENERATED"
	default:
		return Application{}, errors.New("unknown application action")
	}
	application.Version++
	application.UpdatedAt = time.Now().UTC()
	state.Applications[id] = application
	return application, nil
}

func (s *Service) ApplicationPDF(workspaceID, id string) ([]byte, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	application, ok := state.Applications[id]
	if !ok {
		return nil, "", ErrNotFound
	}
	if application.Status != "GENERATED" {
		return nil, "", ErrConflict
	}
	policy, ok := s.findWorkspacePolicy(state, application.PolicyID)
	if !ok {
		return nil, "", ErrNotFound
	}
	checklist, ok := state.Checklists[application.ChecklistID]
	if !ok {
		return nil, "", ErrNotFound
	}
	pdf, err := renderApplicationMinutes(newApplicationMinutes(state.Passport, policy, application, checklist))
	if err != nil {
		return nil, "", err
	}
	return pdf, "P2B-bien-ban-" + id + ".pdf", nil
}

func (s *Service) Alerts(workspaceID string) []Alert {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Alert(nil), s.workspace(workspaceID).Alerts...)
}

func (s *Service) ReadAlert(workspaceID, id string) (Alert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	for index := range state.Alerts {
		if state.Alerts[index].ID == id {
			state.Alerts[index].Read = true
			return state.Alerts[index], nil
		}
	}
	return Alert{}, ErrNotFound
}

func applicationBlockers(checklist Checklist, templateReady bool) []string {
	reasons := make([]string, 0)
	if !templateReady {
		reasons = append(reasons, "Chính sách chưa có mẫu hồ sơ được duyệt")
	}
	for _, item := range checklist.Items {
		if item.Required && item.Status != "AVAILABLE" {
			reasons = append(reasons, "Thiếu tài liệu: "+item.Title)
		}
	}
	return reasons
}

func isRetrievedWorkingPolicy(policy domain.Policy) bool {
	for _, item := range policy.Checklist {
		if item.Key == "applicability_review" || item.Key == "required_documents_review" {
			return true
		}
	}
	return false
}

func companyOverview(pass domain.Passport) string {
	parts := []string{pass.CompanyName}
	labels := map[string]string{"tax_code": "Mã số thuế", "legal_form": "Loại hình pháp lý", "registered_address": "Địa chỉ trụ sở", "province": "Tỉnh/thành"}
	for _, key := range []string{"tax_code", "legal_form", "registered_address", "province"} {
		if field, ok := pass.Fields[key]; ok && field.Value != nil && field.Status == domain.FieldConfirmed {
			parts = append(parts, firstNonEmpty(field.Label, labels[key])+": "+fmt.Sprint(field.Value))
		}
	}
	return strings.Join(parts, "; ") + "."
}
