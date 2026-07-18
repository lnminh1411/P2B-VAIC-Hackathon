package platform

import (
	"time"

	"github.com/p2b/p2b/internal/domain"
	"github.com/p2b/p2b/internal/eligibility"
	"github.com/p2b/p2b/internal/passport"
)

type Job struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Progress  int       `json:"progress"`
	CreatedAt time.Time `json:"created_at"`
}

type BuildPassportInput struct {
	CompanyName  string   `json:"company_name"`
	Website      string   `json:"website"`
	SupportNeeds []string `json:"support_needs"`
	SourceNames  []string `json:"source_names"`
	SourceIDs    []string `json:"source_ids"`
}

type MatchResult struct {
	PolicyID       string             `json:"policy_id"`
	PolicyVersion  int                `json:"policy_version"`
	Title          string             `json:"title"`
	Agency         string             `json:"agency"`
	Benefit        string             `json:"benefit"`
	BenefitAmount  string             `json:"benefit_amount"`
	Deadline       time.Time          `json:"deadline"`
	Score          int                `json:"score"`
	Eligibility    eligibility.Result `json:"eligibility"`
	RankingReasons []string           `json:"ranking_reasons"`
	TemplateReady  bool               `json:"template_ready"`
	RetrievalMode  string             `json:"retrieval_mode"`
	SourceURL      string             `json:"source_url,omitempty"`
}

type MatchRun struct {
	ID              string        `json:"id"`
	PassportVersion int           `json:"passport_version"`
	CreatedAt       time.Time     `json:"created_at"`
	Results         []MatchResult `json:"results"`
}

type EnrichmentCandidate struct {
	ID         string          `json:"id"`
	FieldKey   string          `json:"field_key"`
	Label      string          `json:"label"`
	Value      any             `json:"value"`
	Confidence float64         `json:"confidence"`
	Evidence   domain.Evidence `json:"evidence"`
	Status     string          `json:"status"`
	Warning    string          `json:"warning"`
}

type EnrichmentRun struct {
	ID         string                `json:"id"`
	PolicyID   string                `json:"policy_id"`
	Status     string                `json:"status"`
	Candidates []EnrichmentCandidate `json:"candidates"`
	CreatedAt  time.Time             `json:"created_at"`
}

type ChecklistItem struct {
	ID             string   `json:"id"`
	TemplateKey    string   `json:"template_key"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Required       bool     `json:"required"`
	Status         string   `json:"status"`
	FieldKeys      []string `json:"field_keys"`
	EvidenceSource string   `json:"evidence_source,omitempty"`
}

type Checklist struct {
	ID            string          `json:"id"`
	PolicyID      string          `json:"policy_id"`
	PolicyVersion int             `json:"policy_version"`
	Version       int             `json:"version"`
	Items         []ChecklistItem `json:"items"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type Application struct {
	ID              string            `json:"id"`
	ChecklistID     string            `json:"checklist_id"`
	PolicyID        string            `json:"policy_id"`
	PassportVersion int               `json:"passport_version"`
	PolicyVersion   int               `json:"policy_version"`
	TemplateVersion int               `json:"template_version"`
	Version         int               `json:"version"`
	Status          string            `json:"status"`
	Sections        map[string]string `json:"sections"`
	BlockingReasons []string          `json:"blocking_reasons"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type Alert struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	PolicyID   string    `json:"policy_id,omitempty"`
	Severity   string    `json:"severity"`
	Read       bool      `json:"read"`
	OccurredAt time.Time `json:"occurred_at"`
}

type workspaceState struct {
	Passport     domain.Passport
	Candidates   []passport.Candidate
	Jobs         map[string]Job
	Matches      map[string]MatchRun
	Enrichment   map[string]EnrichmentRun
	Checklists   map[string]Checklist
	Applications map[string]Application
	Alerts       []Alert
}
