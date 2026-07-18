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
	policy, ok := findPolicy(s.policies, policyID)
	if !ok || policy.Lifecycle != "ACTIVE" {
		return Checklist{}, ErrNotFound
	}
	checklist := Checklist{ID: uuid.NewString(), PolicyID: policy.ID, PolicyVersion: policy.Version, Version: 1, UpdatedAt: time.Now().UTC()}
	for _, template := range policy.Checklist {
		status := "AVAILABLE"
		if len(template.FieldKeys) == 0 {
			status = "MISSING"
		}
		for _, fieldKey := range template.FieldKeys {
			if field, exists := state.Passport.Fields[fieldKey]; !exists || field.Status != domain.FieldConfirmed {
				status = "MISSING"
				break
			}
		}
		checklist.Items = append(checklist.Items, ChecklistItem{ID: uuid.NewString(), TemplateKey: template.Key, Title: template.Title, Description: template.Description, Required: template.Required, Status: status, FieldKeys: template.FieldKeys})
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
	policy, ok := findPolicy(s.policies, checklist.PolicyID)
	if !ok {
		return Application{}, ErrNotFound
	}
	application := Application{ID: uuid.NewString(), ChecklistID: checklistID, PolicyID: policy.ID, PassportVersion: state.Passport.Version, PolicyVersion: policy.Version, TemplateVersion: 1, Version: 1, Status: "DRAFT_READY", UpdatedAt: time.Now().UTC(), Sections: map[string]string{
		"company_overview": fmt.Sprintf("%s đề nghị tham gia %s.", state.Passport.CompanyName, policy.Title),
		"support_need":     strings.Join(state.Passport.SupportNeeds, ", "),
		"proposal":         "Doanh nghiệp sẽ sử dụng nguồn hỗ trợ để hoàn thiện sản phẩm, đo lường tác động và mở rộng thị trường.",
	}}
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
	policy, _ := findPolicy(s.policies, application.PolicyID)
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
	policy, _ := findPolicy(s.policies, application.PolicyID)
	lines := []string{"P2B APPLICATION PACKAGE", "Company: " + state.Passport.CompanyName, "Policy: " + policy.Title, "Agency: " + policy.Agency, "Status: Human reviewed", ""}
	for _, key := range []string{"company_overview", "support_need", "proposal"} {
		lines = append(lines, key+": "+application.Sections[key])
	}
	return simplePDF(lines), "P2B-application-" + id + ".pdf", nil
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

func simplePDF(lines []string) []byte {
	content := "BT /F1 11 Tf 50 790 Td 14 TL "
	for index, line := range lines {
		if index > 0 {
			content += "T* "
		}
		content += "(" + pdfEscape(asciiFallback(line)) + ") Tj "
	}
	content += "ET"
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	result := "%PDF-1.4\n"
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = len(result)
		result += fmt.Sprintf("%d 0 obj\n%s\nendobj\n", index+1, object)
	}
	xref := len(result)
	result += fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for index := 1; index <= len(objects); index++ {
		result += fmt.Sprintf("%010d 00000 n \n", offsets[index])
	}
	result += fmt.Sprintf("trailer << /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objects)+1, xref)
	return []byte(result)
}

func pdfEscape(value string) string {
	return strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)").Replace(value)
}

func asciiFallback(value string) string {
	replacer := strings.NewReplacer("Đ", "D", "đ", "d", "á", "a", "à", "a", "ả", "a", "ã", "a", "ạ", "a", "ă", "a", "â", "a", "é", "e", "è", "e", "ê", "e", "í", "i", "ì", "i", "ó", "o", "ò", "o", "ô", "o", "ơ", "o", "ú", "u", "ù", "u", "ư", "u", "ý", "y")
	return replacer.Replace(value)
}
