package passport

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/p2b/p2b/internal/domain"
)

var fieldLabels = map[string]string{
	"legal_name": "Tên pháp lý", "tax_code": "Mã số thuế", "legal_form": "Loại hình doanh nghiệp",
	"incorporation_date": "Ngày thành lập", "operating_status": "Trạng thái hoạt động",
	"charter_capital": "Vốn điều lệ", "revenue": "Doanh thu", "assets": "Tổng tài sản",
	"employee_count": "Số lao động", "registered_address": "Địa chỉ đăng ký", "province": "Tỉnh/thành",
	"industrial_zone": "Khu công nghiệp", "industry_codes": "Ngành nghề", "products": "Sản phẩm",
	"technologies": "Công nghệ", "markets": "Thị trường", "fdi_status": "Doanh nghiệp FDI",
	"foreign_ownership_percent": "Tỷ lệ vốn nước ngoài", "women_owned": "Doanh nghiệp nữ làm chủ",
	"rd_capacity": "Năng lực R&D", "intellectual_property": "Sở hữu trí tuệ", "certifications": "Chứng nhận",
	"green_project": "Dự án công nghệ xanh", "funding_need": "Nhu cầu vốn", "support_plan": "Kế hoạch sử dụng hỗ trợ",
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
	label, allowed := fieldLabels[candidate.FieldKey]
	if !allowed {
		return pass, fmt.Errorf("unknown passport field %q", candidate.FieldKey)
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
	pass.Fields[candidate.FieldKey] = domain.PassportField{Key: candidate.FieldKey, Label: label, Value: candidate.Value, DataType: candidate.DataType, Status: status, Confidence: candidate.Confidence, Evidence: []domain.Evidence{candidate.Evidence}}
	pass.UpdatedAt = time.Now().UTC()
	return pass, nil
}

func ConfirmField(pass domain.Passport, key string, value any, expectedVersion int) (domain.Passport, error) {
	if pass.Version != expectedVersion {
		return pass, errors.New("passport version conflict")
	}
	field, exists := pass.Fields[key]
	if !exists {
		return pass, fmt.Errorf("unknown passport field %q", key)
	}
	if value == nil || strings.TrimSpace(fmt.Sprint(value)) == "" {
		return pass, errors.New("confirmed value is required")
	}
	field.Value = value
	field.Status = domain.FieldConfirmed
	field.Confidence = 1
	pass.Fields[key] = field
	pass.Version++
	pass.UpdatedAt = time.Now().UTC()
	return pass, nil
}

func CanonicalFields() map[string]string {
	copy := make(map[string]string, len(fieldLabels))
	for key, label := range fieldLabels {
		copy[key] = label
	}
	return copy
}
