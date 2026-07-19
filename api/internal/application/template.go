package application

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/p2b/p2b/internal/domain"
	passportdomain "github.com/p2b/p2b/internal/passport"
)

var placeholderPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

var wellKnownVariableLabels = map[string]string{
	"company_name":  "Tên doanh nghiệp",
	"website":       "Website",
	"policy_title":  "Tên chính sách",
	"policy_agency": "Cơ quan ban hành",
	"policy_source": "Nguồn văn bản",
	"current_date":  "Ngày hiện tại",
}

// LabelForVariable resolves a template variable key to a human-readable Vietnamese
// label, falling back to the passport field label, then the raw key.
func LabelForVariable(key string) string {
	if label, ok := wellKnownVariableLabels[key]; ok {
		return label
	}
	if definition, ok := passportdomain.LookupField(key); ok {
		return definition.Label
	}
	return key
}

// MissingVariables returns which placeholders referenced by text have no
// non-blank value in variables, in template order.
func MissingVariables(text string, variables map[string]string) []string {
	missing := make([]string, 0)
	for _, key := range ExtractPlaceholders(text) {
		if strings.TrimSpace(variables[key]) == "" {
			missing = append(missing, key)
		}
	}
	return missing
}

type GenerationRequest struct {
	TemplateText string            `json:"template_text"`
	Variables    map[string]string `json:"variables"`
}

func ExtractPlaceholders(text string) []string {
	unique := map[string]struct{}{}
	for _, match := range placeholderPattern.FindAllStringSubmatch(text, -1) {
		unique[match[1]] = struct{}{}
	}
	result := make([]string, 0, len(unique))
	for key := range unique {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func RenderTemplate(text string, variables map[string]string) string {
	return placeholderPattern.ReplaceAllStringFunc(text, func(placeholder string) string {
		key := placeholderPattern.FindStringSubmatch(placeholder)[1]
		if value := strings.TrimSpace(variables[key]); value != "" {
			return value
		}
		return "[CẦN BỔ SUNG: " + LabelForVariable(key) + "]"
	})
}

func TemplateVariables(passport domain.Passport, policy domain.Policy) map[string]string {
	variables := map[string]string{
		"company_name": strings.TrimSpace(passport.CompanyName), "website": strings.TrimSpace(passport.Website),
		"policy_title": strings.TrimSpace(policy.Title), "policy_agency": strings.TrimSpace(policy.Agency),
		"policy_source": strings.TrimSpace(policy.SourceURL), "current_date": time.Now().Format("02/01/2006"),
	}
	for key, field := range passport.Fields {
		if field.Status == domain.FieldConfirmed && field.Value != nil {
			variables[key] = strings.TrimSpace(fmt.Sprint(field.Value))
		}
	}
	return variables
}
