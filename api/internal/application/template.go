package application

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/p2b/p2b/internal/domain"
)

var placeholderPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

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
		return "[CẦN BỔ SUNG: " + key + "]"
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
