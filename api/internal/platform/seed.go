package platform

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/p2b/p2b/internal/domain"
	"github.com/p2b/p2b/internal/eligibility"
	"github.com/p2b/p2b/internal/passport"
)

func demoPolicies() []domain.Policy {
	now := time.Now().UTC()
	citation := func(name, quote, url string) domain.Evidence {
		return domain.Evidence{SourceID: name, SourceName: name, URL: url, Quote: quote, ContentHash: "sha256:demo-reviewed", ObservedAt: now}
	}
	return []domain.Policy{
		{ID: "innovation-fund", Version: 1, Title: "Quỹ đổi mới sáng tạo cho doanh nghiệp công nghệ", Agency: "Quỹ Phát triển doanh nghiệp nhỏ và vừa", Benefit: "Khoản vay ưu đãi cho dự án đổi mới sáng tạo, R&D và mô hình kinh doanh mới.", BenefitAmount: "Tối đa 2 tỷ đồng", SupportType: "Vốn ưu đãi", Sectors: []string{"AI", "đổi mới sáng tạo", "công nghệ"}, Geographies: []string{"Toàn quốc"}, Deadline: now.AddDate(0, 5, 10), Lifecycle: "ACTIVE", VerifiedAt: now, SourceURL: "https://www.smedf.gov.vn/", TemplateReady: true,
			Rules: []domain.Rule{
				{ID: "innovation-sme", FieldKey: "employee_count", Operator: domain.OpLTE, Expected: 200, Required: true, Description: "Doanh nghiệp có không quá 200 lao động", Citation: citation("Luật Hỗ trợ DNNVV", "Tiêu chí xác định doanh nghiệp nhỏ và vừa", "https://vbpl.vn/")},
				{ID: "innovation-tech", FieldKey: "technologies", Operator: domain.OpContains, Expected: "AI", Required: true, Description: "Có sản phẩm hoặc năng lực công nghệ", Citation: citation("Chương trình đổi mới", "Ưu tiên dự án đổi mới sáng tạo và công nghệ", "https://business.gov.vn/")},
			}, Checklist: []domain.ChecklistTemplateItem{{Key: "registration", Title: "Giấy chứng nhận đăng ký doanh nghiệp", Description: "Bản còn hiệu lực", Required: true, FieldKeys: []string{"tax_code", "legal_name"}}, {Key: "proposal", Title: "Thuyết minh dự án đổi mới", Description: "Mục tiêu, ngân sách và tác động", Required: true, FieldKeys: []string{"technologies", "support_plan"}}},
		},
		{ID: "green-hcm", Version: 1, Title: "Chương trình hỗ trợ chuyển đổi xanh TP.HCM", Agency: "Sở Khoa học và Công nghệ TP.HCM", Benefit: "Tư vấn, đánh giá công nghệ và hỗ trợ chi phí thử nghiệm giải pháp xanh.", BenefitAmount: "Hỗ trợ đến 50% chi phí", SupportType: "Công nghệ xanh", Sectors: []string{"công nghệ xanh", "năng lượng", "sản xuất"}, Geographies: []string{"Hồ Chí Minh"}, Deadline: now.AddDate(0, 2, 20), Lifecycle: "ACTIVE", VerifiedAt: now, SourceURL: "https://dost.hochiminhcity.gov.vn/", TemplateReady: true,
			Rules: []domain.Rule{
				{ID: "green-location", FieldKey: "province", Operator: domain.OpIN, Expected: []string{"Hồ Chí Minh", "TP. Hồ Chí Minh"}, Required: true, Description: "Hoạt động tại TP.HCM", Citation: citation("Chương trình địa phương", "Đối tượng hoạt động trên địa bàn TP.HCM", "https://dost.hochiminhcity.gov.vn/")},
				{ID: "green-project", FieldKey: "green_project", Operator: domain.OpExists, Required: true, Description: "Có dự án giảm phát thải hoặc công nghệ xanh", Citation: citation("Tiêu chí công nghệ xanh", "Dự án có tác động môi trường đo lường được", "https://dost.hochiminhcity.gov.vn/")},
			}, Checklist: []domain.ChecklistTemplateItem{{Key: "green-plan", Title: "Kế hoạch chuyển đổi xanh", Description: "Hiện trạng, mục tiêu và chỉ số giảm phát thải", Required: true, FieldKeys: []string{"green_project"}}, {Key: "registration", Title: "Giấy đăng ký doanh nghiệp", Required: true, FieldKeys: []string{"legal_name", "tax_code"}}},
		},
		{ID: "digital-sme", Version: 2, Title: "Hỗ trợ chuyển đổi số cho SME", Agency: "Cục Phát triển doanh nghiệp", Benefit: "Tư vấn lộ trình, đánh giá mức độ sẵn sàng và hỗ trợ triển khai giải pháp số.", BenefitAmount: "Tài trợ tư vấn chuyên sâu", SupportType: "Chuyển đổi số", Sectors: []string{"chuyển đổi số", "AI", "SME"}, Geographies: []string{"Toàn quốc"}, Deadline: now.AddDate(0, 8, 0), Lifecycle: "ACTIVE", VerifiedAt: now, SourceURL: "https://business.gov.vn/", TemplateReady: false,
			Rules:     []domain.Rule{{ID: "digital-sme-size", FieldKey: "employee_count", Operator: domain.OpLTE, Expected: 200, Required: true, Description: "Đáp ứng quy mô SME", Citation: citation("Nghị định 80/2021/NĐ-CP", "Tiêu chí lao động và quy mô doanh nghiệp", "https://vbpl.vn/")}},
			Checklist: []domain.ChecklistTemplateItem{{Key: "digital-assessment", Title: "Phiếu đánh giá mức độ chuyển đổi số", Required: true, FieldKeys: []string{"technologies"}}},
		},
		{ID: "pending-rnd", Version: 1, Title: "Ứng viên chính sách R&D chưa duyệt", Agency: "Nguồn crawl demo", Benefit: "Chưa được công bố", SupportType: "R&D", Deadline: now.AddDate(0, 9, 0), Lifecycle: "PENDING_REVIEW", VerifiedAt: now, SourceURL: "https://vbpl.vn/"},
	}
}

func demoCandidates(input BuildPassportInput, now time.Time) []passport.Candidate {
	sourceName := "Hồ sơ doanh nghiệp"
	if len(input.SourceNames) > 0 {
		sourceName = input.SourceNames[0]
	}
	evidence := func(field, quote string) domain.Evidence {
		return domain.Evidence{SourceID: "source-" + uuid.NewString(), SourceName: sourceName, Page: 1, Quote: quote, ContentHash: "sha256:demo-" + field, ObservedAt: now}
	}
	province := "Hồ Chí Minh"
	if strings.Contains(strings.ToLower(input.CompanyName), "hà nội") {
		province = "Hà Nội"
	}
	return []passport.Candidate{
		{ID: uuid.NewString(), FieldKey: "tax_code", Value: "0312345678", DataType: "string", Confidence: .96, Status: "NEEDS_REVIEW", Evidence: evidence("tax-code", "Mã số doanh nghiệp: 0312345678")},
		{ID: uuid.NewString(), FieldKey: "employee_count", Value: 25, DataType: "number", Confidence: .88, Status: "NEEDS_REVIEW", Evidence: evidence("employees", "Đội ngũ hiện có 25 nhân sự toàn thời gian")},
		{ID: uuid.NewString(), FieldKey: "province", Value: province, DataType: "string", Confidence: .93, Status: "NEEDS_REVIEW", Evidence: evidence("province", "Trụ sở chính tại "+province)},
		{ID: uuid.NewString(), FieldKey: "charter_capital", Value: 5_000_000_000, DataType: "number", Confidence: .91, Status: "NEEDS_REVIEW", Evidence: evidence("capital", "Vốn điều lệ: 5.000.000.000 đồng")},
		{ID: uuid.NewString(), FieldKey: "technologies", Value: []string{"AI", "GreenTech"}, DataType: "string_array", Confidence: .84, Status: "NEEDS_REVIEW", Evidence: evidence("technologies", "Nền tảng sử dụng AI để tối ưu năng lượng và giảm phát thải")},
		{ID: uuid.NewString(), FieldKey: "green_project", Value: "Nền tảng đo lường và tối ưu tiêu thụ năng lượng bằng AI", DataType: "string", Confidence: .79, Status: "NEEDS_REVIEW", Evidence: evidence("green-project", "Giải pháp giúp doanh nghiệp theo dõi và giảm phát thải vận hành")},
	}
}

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

func suggestedValue(key string) any {
	switch key {
	case "employee_count":
		return 25
	case "province":
		return "Hồ Chí Minh"
	case "technologies":
		return []string{"AI", "GreenTech"}
	case "green_project":
		return "Giải pháp giám sát và tối ưu năng lượng bằng AI"
	case "tax_code":
		return "0312345678"
	case "support_plan":
		return "Thử nghiệm sản phẩm, đo lường tác động và mở rộng thị trường"
	default:
		return nil
	}
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

func candidateQuote(value any) string { return fmt.Sprint(value) }
