package domain

import "time"

type Workspace struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

type FieldStatus string

const (
	FieldMissing     FieldStatus = "MISSING"
	FieldExtracted   FieldStatus = "EXTRACTED"
	FieldNeedsReview FieldStatus = "NEEDS_REVIEW"
	FieldConfirmed   FieldStatus = "CONFIRMED"
	FieldConflicted  FieldStatus = "CONFLICTED"
	FieldStale       FieldStatus = "STALE"
)

type Evidence struct {
	SourceID    string    `json:"source_id"`
	SourceName  string    `json:"source_name"`
	URL         string    `json:"url,omitempty"`
	Page        int       `json:"page,omitempty"`
	Quote       string    `json:"quote"`
	ContentHash string    `json:"content_hash"`
	ObservedAt  time.Time `json:"observed_at"`
}

type PassportField struct {
	Key        string      `json:"key"`
	Label      string      `json:"label"`
	Value      any         `json:"value,omitempty"`
	DataType   string      `json:"data_type"`
	Status     FieldStatus `json:"status"`
	Confidence float64     `json:"confidence"`
	Evidence   []Evidence  `json:"evidence"`
}

type Passport struct {
	ID           string                   `json:"id"`
	WorkspaceID  string                   `json:"workspace_id"`
	CompanyName  string                   `json:"company_name"`
	Website      string                   `json:"website,omitempty"`
	SupportNeeds []string                 `json:"support_needs"`
	Version      int                      `json:"version"`
	Fields       map[string]PassportField `json:"fields"`
	UpdatedAt    time.Time                `json:"updated_at"`
}

type RuleOperator string

const (
	OpEQ         RuleOperator = "EQ"
	OpIN         RuleOperator = "IN"
	OpContains   RuleOperator = "CONTAINS"
	OpGT         RuleOperator = "GT"
	OpGTE        RuleOperator = "GTE"
	OpLT         RuleOperator = "LT"
	OpLTE        RuleOperator = "LTE"
	OpExists     RuleOperator = "EXISTS"
	OpDateBefore RuleOperator = "DATE_BEFORE"
	OpDateAfter  RuleOperator = "DATE_AFTER"
)

type Rule struct {
	ID          string       `json:"id"`
	FieldKey    string       `json:"field_key"`
	Operator    RuleOperator `json:"operator"`
	Expected    any          `json:"expected,omitempty"`
	Required    bool         `json:"required"`
	Description string       `json:"description"`
	Citation    Evidence     `json:"citation"`
}

type Policy struct {
	ID            string                  `json:"id"`
	Version       int                     `json:"version"`
	Title         string                  `json:"title"`
	Agency        string                  `json:"agency"`
	Benefit       string                  `json:"benefit"`
	BenefitAmount string                  `json:"benefit_amount,omitempty"`
	SupportType   string                  `json:"support_type"`
	Sectors       []string                `json:"sectors"`
	Geographies   []string                `json:"geographies"`
	Deadline      time.Time               `json:"deadline"`
	Rules         []Rule                  `json:"rules"`
	Checklist     []ChecklistTemplateItem `json:"checklist"`
	Lifecycle     string                  `json:"lifecycle"`
	VerifiedAt    time.Time               `json:"verified_at"`
	SourceURL     string                  `json:"source_url"`
	TemplateReady bool                    `json:"template_ready"`
}

type ChecklistTemplateItem struct {
	Key         string   `json:"key"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	FieldKeys   []string `json:"field_keys"`
}
