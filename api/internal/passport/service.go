package passport

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/p2b/p2b/internal/domain"
)

type FieldDefinition struct {
	Label    string
	DataType string
}

var fieldDefinitions = map[string]FieldDefinition{
	"legal_name": {Label: "Tên pháp lý", DataType: "string"}, "tax_code": {Label: "Mã số thuế", DataType: "string"},
	"legal_form": {Label: "Loại hình doanh nghiệp", DataType: "string"}, "incorporation_date": {Label: "Ngày thành lập", DataType: "date"},
	"operating_status": {Label: "Trạng thái hoạt động", DataType: "string"}, "charter_capital": {Label: "Vốn điều lệ", DataType: "money"},
	"revenue": {Label: "Doanh thu", DataType: "money"}, "assets": {Label: "Tổng tài sản", DataType: "money"},
	"employee_count": {Label: "Số lao động", DataType: "integer"}, "registered_address": {Label: "Địa chỉ đăng ký", DataType: "string"},
	"province": {Label: "Tỉnh/thành", DataType: "string"}, "industrial_zone": {Label: "Khu công nghiệp", DataType: "string"},
	"industry_codes": {Label: "Ngành nghề", DataType: "string_array"}, "products": {Label: "Sản phẩm", DataType: "string_array"},
	"technologies": {Label: "Công nghệ", DataType: "string_array"}, "markets": {Label: "Thị trường", DataType: "string_array"},
	"fdi_status": {Label: "Doanh nghiệp FDI", DataType: "boolean"}, "foreign_ownership_percent": {Label: "Tỷ lệ vốn nước ngoài", DataType: "number"},
	"women_owned": {Label: "Doanh nghiệp nữ làm chủ", DataType: "boolean"}, "rd_capacity": {Label: "Năng lực R&D", DataType: "string"},
	"intellectual_property": {Label: "Sở hữu trí tuệ", DataType: "string_array"}, "certifications": {Label: "Chứng nhận", DataType: "string_array"},
	"green_project": {Label: "Dự án công nghệ xanh", DataType: "string_array"}, "funding_need": {Label: "Nhu cầu vốn", DataType: "money"},
	"support_plan": {Label: "Kế hoạch sử dụng hỗ trợ", DataType: "string"},
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

func ValidateFieldValue(key string, value any) error {
	definition, exists := LookupField(key)
	if !exists {
		return fmt.Errorf("unknown passport field %q", key)
	}
	switch definition.DataType {
	case "string":
		text, ok := value.(string)
		maxCharacters := 2000
		if key == "legal_name" {
			maxCharacters = 200
		}
		if !ok || strings.TrimSpace(text) == "" || len([]rune(text)) > maxCharacters {
			return fmt.Errorf("field value must be a non-empty string up to %d characters", maxCharacters)
		}
	case "date":
		text, ok := value.(string)
		if !ok {
			return errors.New("field value must be a date in YYYY-MM-DD format")
		}
		if _, err := time.Parse("2006-01-02", text); err != nil {
			return errors.New("field value must be a date in YYYY-MM-DD format")
		}
	case "money", "number", "integer":
		number, ok := numericValue(value)
		if !ok || number < 0 || (definition.DataType == "integer" && math.Trunc(number) != number) {
			return errors.New("field value must be a non-negative number with the expected precision")
		}
		if key == "foreign_ownership_percent" && number > 100 {
			return errors.New("ownership percentage must be between 0 and 100")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("field value must be boolean")
		}
	case "string_array":
		items, ok := stringItems(value)
		if !ok || len(items) == 0 || len(items) > 100 {
			return errors.New("field value must contain between 1 and 100 text items")
		}
		for _, item := range items {
			if strings.TrimSpace(item) == "" || len([]rune(item)) > 500 {
				return errors.New("each field item must contain up to 500 characters")
			}
		}
	default:
		return errors.New("unsupported passport field datatype")
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
