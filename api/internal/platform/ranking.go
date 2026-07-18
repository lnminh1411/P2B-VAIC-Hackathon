package platform

import (
	"strings"

	"github.com/p2b/p2b/internal/domain"
	"github.com/p2b/p2b/internal/eligibility"
)

func rank(policy domain.Policy, pass domain.Passport, evaluation eligibility.Result) (int, []string) {
	score := 45
	reasons := []string{"Chính sách đang hiệu lực và đã được duyệt"}
	for _, need := range pass.SupportNeeds {
		if containsFold(policy.SupportType+" "+strings.Join(policy.Sectors, " ")+" "+policy.Benefit, need) {
			score += 12
			reasons = append(reasons, "Phù hợp nhu cầu: "+need)
		}
	}
	switch evaluation.Status {
	case eligibility.StatusMet:
		score += 30
		reasons = append(reasons, "Đã đáp ứng các điều kiện bắt buộc")
	case eligibility.StatusMissingInfo:
		score += 14
		reasons = append(reasons, "Có tiềm năng nhưng cần bổ sung thông tin")
	case eligibility.StatusNotMet:
		score -= 12
		reasons = append(reasons, "Có điều kiện hiện chưa đáp ứng")
	}
	if policy.TemplateReady {
		score += 4
		reasons = append(reasons, "Có mẫu hồ sơ sẵn sàng")
	}
	if score > 99 {
		score = 99
	}
	if score < 0 {
		score = 0
	}
	return score, reasons
}

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(strings.TrimSpace(needle)))
}

func findPolicy(policies []domain.Policy, id string) (domain.Policy, bool) {
	for _, policy := range policies {
		if policy.ID == id {
			return policy, true
		}
	}
	return domain.Policy{}, false
}

func clonePassport(input domain.Passport) domain.Passport {
	result := input
	result.SupportNeeds = append([]string(nil), input.SupportNeeds...)
	result.Fields = make(map[string]domain.PassportField, len(input.Fields))
	for key, field := range input.Fields {
		field.Evidence = append([]domain.Evidence(nil), field.Evidence...)
		result.Fields[key] = field
	}
	return result
}
