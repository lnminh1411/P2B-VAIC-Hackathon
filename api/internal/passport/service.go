package passport

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/p2b/p2b/internal/domain"
)

type FieldDefinition struct {
	Label                  string
	DataType               string
	Description            string
	EvidenceTerms          []string
	ForbiddenEvidenceTerms []string
}

var fieldDefinitions = map[string]FieldDefinition{
	"legal_name":                {Label: "Tên pháp lý", DataType: "string", Description: "Tên đăng ký đầy đủ của pháp nhân, không dùng tên sản phẩm hoặc thương hiệu nếu không phải tên pháp lý."},
	"tax_code":                  {Label: "Mã số thuế", DataType: "string", Description: "Mã số thuế hoặc mã số doanh nghiệp của pháp nhân."},
	"legal_form":                {Label: "Loại hình doanh nghiệp", DataType: "string", Description: "Loại hình pháp lý như công ty cổ phần, công ty TNHH hoặc doanh nghiệp tư nhân."},
	"incorporation_date":        {Label: "Ngày thành lập", DataType: "date", Description: "Ngày thành lập hoặc ngày đăng ký lần đầu; value phải ở dạng YYYY-MM-DD."},
	"operating_status":          {Label: "Trạng thái hoạt động", DataType: "string", Description: "Trạng thái pháp lý hoặc hoạt động hiện tại của doanh nghiệp."},
	"charter_capital":           {Label: "Vốn điều lệ", DataType: "money", Description: "Chỉ vốn điều lệ. Nếu tài liệu nêu nhiều mốc lịch sử, trả từng giá trị có mốc thời gian thay vì tự chọn giá trị hiện tại.", EvidenceTerms: []string{"vốn điều lệ", "charter capital"}},
	"revenue":                   {Label: "Doanh thu", DataType: "money", Description: "Doanh thu của doanh nghiệp, giữ quote chứa kỳ báo cáo và đơn vị."},
	"assets":                    {Label: "Tổng tài sản", DataType: "money", Description: "Tổng tài sản của doanh nghiệp, không dùng vốn chủ sở hữu hoặc vốn hóa thị trường."},
	"employee_count":            {Label: "Số lao động", DataType: "integer", Description: "Tổng số người lao động hoặc tổng headcount toàn doanh nghiệp; không dùng nhân lực môi giới, cộng tác viên hoặc một bộ phận.", EvidenceTerms: []string{"số lao động", "số nhân viên", "tổng số lao động", "tổng số nhân viên", "employee count", "headcount"}, ForbiddenEvidenceTerms: []string{"môi giới", "broker", "cộng tác viên"}},
	"registered_address":        {Label: "Địa chỉ đăng ký", DataType: "string", Description: "Địa chỉ trụ sở chính hoặc địa chỉ đăng ký của pháp nhân."},
	"province":                  {Label: "Tỉnh/thành", DataType: "string", Description: "Tỉnh hoặc thành phố trực thuộc trung ương của địa chỉ đăng ký."},
	"industrial_zone":           {Label: "Khu công nghiệp", DataType: "string", Description: "Tên khu công nghiệp, khu chế xuất hoặc khu công nghệ nơi doanh nghiệp hoạt động."},
	"industry_codes":            {Label: "Ngành nghề", DataType: "string_array", Description: "Ngành nghề kinh doanh hoặc mã ngành được nêu rõ cho doanh nghiệp."},
	"products":                  {Label: "Sản phẩm", DataType: "string_array", Description: "Sản phẩm hoặc dịch vụ cụ thể doanh nghiệp cung cấp."},
	"technologies":              {Label: "Công nghệ", DataType: "string_array", Description: "Công nghệ, nền tảng kỹ thuật hoặc quy trình công nghệ doanh nghiệp đang sử dụng."},
	"markets":                   {Label: "Thị trường", DataType: "string_array", Description: "Thị trường địa lý hoặc phân khúc khách hàng mục tiêu của doanh nghiệp."},
	"fdi_status":                {Label: "Doanh nghiệp FDI", DataType: "boolean", Description: "Đúng khi tài liệu xác định doanh nghiệp là FDI hoặc có vốn đầu tư nước ngoài; không suy đoán."},
	"foreign_ownership_percent": {Label: "Tỷ lệ vốn nước ngoài", DataType: "number", Description: "Tỷ lệ phần trăm sở hữu hoặc vốn góp nước ngoài, từ 0 đến 100."},
	"women_owned":               {Label: "Doanh nghiệp nữ làm chủ", DataType: "boolean", Description: "Đúng khi tài liệu xác định doanh nghiệp do phụ nữ sở hữu hoặc làm chủ; không suy từ tên người đại diện."},
	"rd_capacity":               {Label: "Năng lực R&D", DataType: "string", Description: "Mô tả năng lực nghiên cứu và phát triển của doanh nghiệp."},
	"intellectual_property":     {Label: "Sở hữu trí tuệ", DataType: "string_array", Description: "Bằng sáng chế, nhãn hiệu, bản quyền, kiểu dáng hoặc tài sản sở hữu trí tuệ được nêu rõ."},
	"certifications":            {Label: "Chứng nhận", DataType: "string_array", Description: "Chứng nhận, tiêu chuẩn hoặc giấy chứng nhận doanh nghiệp đã đạt được."},
	"green_project":             {Label: "Dự án công nghệ xanh", DataType: "string_array", Description: "Dự án, hoạt động hoặc giải pháp xanh được mô tả rõ; không suy diễn từ tuyên bố chung."},
	"funding_need":              {Label: "Nhu cầu vốn", DataType: "money", Description: "Số tiền doanh nghiệp đang cần huy động hoặc đề nghị hỗ trợ; không dùng vốn điều lệ, doanh thu hoặc tài sản."},
	"support_plan":              {Label: "Kế hoạch sử dụng hỗ trợ", DataType: "string", Description: "Kế hoạch cụ thể sử dụng khoản hỗ trợ hoặc nguồn vốn đề nghị."},
}

type CanonicalFieldDefinition struct {
	Key                    string
	Label                  string
	DataType               string
	Description            string
	EvidenceTerms          []string
	ForbiddenEvidenceTerms []string
}

func CanonicalFieldCatalog() []CanonicalFieldDefinition {
	keys := make([]string, 0, len(fieldDefinitions))
	for key := range fieldDefinitions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]CanonicalFieldDefinition, 0, len(keys))
	for _, key := range keys {
		definition := fieldDefinitions[key]
		result = append(result, CanonicalFieldDefinition{Key: key, Label: definition.Label, DataType: definition.DataType, Description: definition.Description, EvidenceTerms: definition.EvidenceTerms, ForbiddenEvidenceTerms: definition.ForbiddenEvidenceTerms})
	}
	return result
}

type Candidate struct {
	ID         string          `json:"id"`
	FieldKey   string          `json:"field_key"`
	Value      any             `json:"value"`
	DataType   string          `json:"data_type"`
	Confidence float64         `json:"confidence"`
	Evidence   domain.Evidence `json:"evidence"`
	Status     string          `json:"status"`
}

func MergeCandidate(pass domain.Passport, candidate Candidate) (domain.Passport, error) {
	definition, allowed := LookupField(candidate.FieldKey)
	if !allowed {
		return pass, fmt.Errorf("unknown passport field %q", candidate.FieldKey)
	}
	if candidate.DataType != definition.DataType {
		return pass, fmt.Errorf("unexpected datatype %q for passport field %q", candidate.DataType, candidate.FieldKey)
	}
	if err := ValidateFieldValue(candidate.FieldKey, candidate.Value); err != nil {
		return pass, err
	}
	if candidate.Value == nil || strings.TrimSpace(candidate.Evidence.Quote) == "" || candidate.Evidence.SourceID == "" || candidate.Evidence.ContentHash == "" {
		return pass, errors.New("candidate requires a value and grounded evidence")
	}
	if candidate.Confidence < 0 || candidate.Confidence > 1 {
		return pass, errors.New("confidence must be between 0 and 1")
	}
	if pass.Fields == nil {
		pass.Fields = map[string]domain.PassportField{}
	}
	status := domain.FieldExtracted
	if candidate.Confidence < .75 {
		status = domain.FieldNeedsReview
	}
	if current, exists := pass.Fields[candidate.FieldKey]; exists && current.Value != nil && !reflect.DeepEqual(current.Value, candidate.Value) {
		status = domain.FieldConflicted
	}
	pass.Fields[candidate.FieldKey] = domain.PassportField{Key: candidate.FieldKey, Label: definition.Label, Value: candidate.Value, DataType: candidate.DataType, Status: status, Confidence: candidate.Confidence, Evidence: []domain.Evidence{candidate.Evidence}}
	pass.UpdatedAt = time.Now().UTC()
	return pass, nil
}

func ConfirmField(pass domain.Passport, key string, value any, expectedVersion int) (domain.Passport, error) {
	if pass.Version != expectedVersion {
		return pass, errors.New("passport version conflict")
	}
	definition, allowed := LookupField(key)
	if !allowed {
		return pass, fmt.Errorf("unknown passport field %q", key)
	}
	field, exists := pass.Fields[key]
	if !exists {
		field = domain.PassportField{Key: key, Label: definition.Label, DataType: definition.DataType, Status: domain.FieldMissing, Evidence: []domain.Evidence{}}
	}
	if value == nil || strings.TrimSpace(fmt.Sprint(value)) == "" {
		return pass, errors.New("confirmed value is required")
	}
	if err := ValidateFieldValue(key, value); err != nil {
		return pass, err
	}
	field.Value = value
	field.Label = definition.Label
	field.DataType = definition.DataType
	field.Status = domain.FieldConfirmed
	field.Confidence = 1
	field.Evidence = append(field.Evidence, domain.Evidence{
		SourceID: "user-input", SourceName: "Người dùng xác nhận", Quote: fmt.Sprint(value),
		ContentHash: fmt.Sprintf("user-confirmation:v%d", expectedVersion+1), ObservedAt: time.Now().UTC(),
	})
	pass.Fields[key] = field
	pass.Version++
	pass.UpdatedAt = time.Now().UTC()
	return pass, nil
}

func CanonicalFields() map[string]string {
	copy := make(map[string]string, len(fieldDefinitions))
	for key, definition := range fieldDefinitions {
		copy[key] = definition.Label
	}
	return copy
}

func CanonicalFieldTypes() map[string]string {
	copy := make(map[string]string, len(fieldDefinitions))
	for key, definition := range fieldDefinitions {
		copy[key] = definition.DataType
	}
	return copy
}

func LookupField(key string) (FieldDefinition, bool) {
	definition, exists := fieldDefinitions[key]
	return definition, exists
}

func ValidateEvidence(fieldKey, quote string) error {
	definition, exists := LookupField(fieldKey)
	if !exists {
		return fmt.Errorf("unknown passport field %q", fieldKey)
	}
	text := strings.ToLower(strings.Join(strings.Fields(quote), " "))
	for _, forbidden := range definition.ForbiddenEvidenceTerms {
		if strings.Contains(text, forbidden) {
			return fmt.Errorf("evidence uses a disallowed concept for passport field %q", fieldKey)
		}
	}
	if len(definition.EvidenceTerms) == 0 {
		return nil
	}
	for _, term := range definition.EvidenceTerms {
		if strings.Contains(text, term) {
			return nil
		}
	}
	return fmt.Errorf("evidence does not identify passport field %q", fieldKey)
}

func ValidateFieldValue(key string, value any) error {
	definition, exists := LookupField(key)
	if !exists {
		return fmt.Errorf("unknown passport field %q", key)
	}
	switch definition.DataType {
	case "string":
		return validateStringValue(key, value)
	case "date":
		return validateDateValue(value)
	case "money", "number", "integer":
		return validateNumericValue(key, definition.DataType, value)
	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("field value must be boolean")
		}
	case "string_array":
		return validateStringArrayValue(value)
	default:
		return errors.New("unsupported passport field datatype")
	}
	return nil
}

func validateStringValue(key string, value any) error {
	text, ok := value.(string)
	maxCharacters := 2000
	if key == "legal_name" {
		maxCharacters = 200
	}
	if !ok || strings.TrimSpace(text) == "" || len([]rune(text)) > maxCharacters {
		return fmt.Errorf("field value must be a non-empty string up to %d characters", maxCharacters)
	}
	return nil
}

func validateDateValue(value any) error {
	text, ok := value.(string)
	if !ok {
		return errors.New("field value must be a date in YYYY-MM-DD format")
	}
	if _, err := time.Parse(time.DateOnly, text); err != nil {
		return errors.New("field value must be a date in YYYY-MM-DD format")
	}
	return nil
}

func validateNumericValue(key, dataType string, value any) error {
	number, ok := numericValue(value)
	if !ok || number < 0 || (dataType == "integer" && math.Trunc(number) != number) {
		return errors.New("field value must be a non-negative number with the expected precision")
	}
	if key == "foreign_ownership_percent" && number > 100 {
		return errors.New("ownership percentage must be between 0 and 100")
	}
	return nil
}

func validateStringArrayValue(value any) error {
	items, ok := stringItems(value)
	if !ok || len(items) == 0 || len(items) > 100 {
		return errors.New("field value must contain between 1 and 100 text items")
	}
	for _, item := range items {
		if strings.TrimSpace(item) == "" || len([]rune(item)) > 500 {
			return errors.New("each field item must contain up to 500 characters")
		}
	}
	return nil
}

func numericValue(value any) (float64, bool) {
	var number float64
	switch typed := value.(type) {
	case int:
		number = float64(typed)
	case int32:
		number = float64(typed)
	case int64:
		number = float64(typed)
	case float32:
		number = float64(typed)
	case float64:
		number = typed
	default:
		return 0, false
	}
	return number, !math.IsNaN(number) && !math.IsInf(number, 0)
}

func stringItems(value any) ([]string, bool) {
	if items, ok := value.([]string); ok {
		return items, true
	}
	values, ok := value.([]any)
	if !ok {
		return nil, false
	}
	items := make([]string, 0, len(values))
	for _, value := range values {
		item, ok := value.(string)
		if !ok {
			return nil, false
		}
		items = append(items, item)
	}
	return items, true
}

func EnsureCanonicalFields(pass domain.Passport) domain.Passport {
	if pass.Fields == nil {
		pass.Fields = map[string]domain.PassportField{}
	}
	for key, definition := range fieldDefinitions {
		if _, exists := pass.Fields[key]; !exists {
			pass.Fields[key] = domain.PassportField{Key: key, Label: definition.Label, DataType: definition.DataType, Status: domain.FieldMissing, Evidence: []domain.Evidence{}}
		}
	}
	return pass
}
