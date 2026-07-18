package platform

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/p2b/p2b/internal/domain"
	"github.com/p2b/p2b/internal/eligibility"
	passportservice "github.com/p2b/p2b/internal/passport"
)

var (
	ErrNotFound = errors.New("resource not found")
	ErrConflict = errors.New("resource version conflict")
	ErrBlocked  = errors.New("operation blocked by missing evidence")
)

type Service struct {
	mu         sync.RWMutex
	workspaces map[string]*workspaceState
	policies   []domain.Policy
}

func NewDemoService() *Service {
	return &Service{workspaces: map[string]*workspaceState{}, policies: demoPolicies()}
}

func (s *Service) workspace(id string) *workspaceState {
	state, ok := s.workspaces[id]
	if ok {
		return state
	}
	now := time.Now().UTC()
	state = &workspaceState{
		Passport: domain.Passport{ID: uuid.NewString(), WorkspaceID: id, Version: 1, Fields: map[string]domain.PassportField{}, UpdatedAt: now},
		Jobs:     map[string]Job{}, Matches: map[string]MatchRun{}, Enrichment: map[string]EnrichmentRun{},
		Checklists: map[string]Checklist{}, Applications: map[string]Application{},
		Alerts: []Alert{{ID: uuid.NewString(), Type: "POLICY_NEW", Title: "Cơ hội mới cho doanh nghiệp công nghệ", Message: "Quỹ đổi mới sáng tạo đang nhận hồ sơ đến cuối năm.", PolicyID: "innovation-fund", Severity: "info", OccurredAt: now}},
	}
	s.workspaces[id] = state
	return state
}

func (s *Service) BuildPassport(workspaceID string, input BuildPassportInput) (Job, error) {
	input.CompanyName = strings.TrimSpace(input.CompanyName)
	if input.CompanyName == "" || len(input.CompanyName) > 200 {
		return Job{}, errors.New("company_name is required and limited to 200 characters")
	}
	if len(input.SourceNames) > 10 {
		return Job{}, errors.New("at most 10 PDF sources are allowed")
	}
	if input.Website != "" && !strings.HasPrefix(strings.ToLower(input.Website), "https://") {
		return Job{}, errors.New("website must use https")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	now := time.Now().UTC()
	userEvidence := domain.Evidence{SourceID: "user-input", SourceName: "Thông tin do người dùng cung cấp", Quote: input.CompanyName, ContentHash: "user:" + workspaceID, ObservedAt: now}
	state.Passport.CompanyName = input.CompanyName
	state.Passport.Website = input.Website
	state.Passport.SupportNeeds = cleanStrings(input.SupportNeeds)
	state.Passport.Fields = map[string]domain.PassportField{
		"legal_name": {Key: "legal_name", Label: "Tên pháp lý", Value: input.CompanyName, DataType: "string", Status: domain.FieldConfirmed, Confidence: 1, Evidence: []domain.Evidence{userEvidence}},
	}
	if input.Website != "" {
		state.Passport.Fields["website"] = domain.PassportField{Key: "website", Label: "Website", Value: input.Website, DataType: "url", Status: domain.FieldConfirmed, Confidence: 1, Evidence: []domain.Evidence{{SourceID: "user-input", SourceName: "Website do người dùng cung cấp", URL: input.Website, Quote: input.Website, ContentHash: "user:" + workspaceID, ObservedAt: now}}}
	}
	state.Passport.Version++
	state.Passport.UpdatedAt = now
	state.Candidates = demoCandidates(input, now)
	for _, candidate := range state.Candidates {
		updated, err := passportservice.MergeCandidate(state.Passport, candidate)
		if err == nil {
			state.Passport = updated
		}
	}
	job := Job{ID: uuid.NewString(), Type: "PASSPORT_BUILD", Status: "SUCCEEDED", Progress: 100, CreatedAt: now}
	state.Jobs[job.ID] = job
	return job, nil
}

func (s *Service) Passport(workspaceID string) domain.Passport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return clonePassport(s.workspace(workspaceID).Passport)
}

func (s *Service) Candidates(workspaceID string) []passportservice.Candidate {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.workspace(workspaceID).Candidates)
}

func (s *Service) ConfirmField(workspaceID, key string, value any, expectedVersion int) (domain.Passport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	updated, err := passportservice.ConfirmField(state.Passport, key, value, expectedVersion)
	if err != nil {
		if strings.Contains(err.Error(), "version conflict") {
			return domain.Passport{}, ErrConflict
		}
		return domain.Passport{}, err
	}
	state.Passport = updated
	for index := range state.Candidates {
		if state.Candidates[index].FieldKey == key {
			state.Candidates[index].Status = "ACCEPTED"
		}
	}
	return clonePassport(updated), nil
}

func (s *Service) Job(workspaceID, id string) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.workspace(workspaceID).Jobs[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	return job, nil
}

func (s *Service) Policies(activeOnly bool) []domain.Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Policy, 0, len(s.policies))
	for _, policy := range s.policies {
		if !activeOnly || policy.Lifecycle == "ACTIVE" {
			result = append(result, policy)
		}
	}
	return result
}

func (s *Service) Policy(id string, version int) (domain.Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, policy := range s.policies {
		if policy.ID == id && (version == 0 || policy.Version == version) {
			return policy, nil
		}
	}
	return domain.Policy{}, ErrNotFound
}

func (s *Service) Match(workspaceID string) MatchRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	run := MatchRun{ID: uuid.NewString(), PassportVersion: state.Passport.Version, CreatedAt: time.Now().UTC()}
	for _, policy := range s.policies {
		if policy.Lifecycle != "ACTIVE" {
			continue
		}
		evaluation := eligibility.Evaluate(state.Passport, policy.Rules)
		score, reasons := rank(policy, state.Passport, evaluation)
		run.Results = append(run.Results, MatchResult{PolicyID: policy.ID, PolicyVersion: policy.Version, Title: policy.Title, Agency: policy.Agency, Benefit: policy.Benefit, BenefitAmount: policy.BenefitAmount, Deadline: policy.Deadline, Score: score, Eligibility: evaluation, RankingReasons: reasons, TemplateReady: policy.TemplateReady, RetrievalMode: "HYBRID_DEMO"})
	}
	sort.SliceStable(run.Results, func(i, j int) bool { return run.Results[i].Score > run.Results[j].Score })
	state.Matches[run.ID] = run
	return run
}

func (s *Service) MatchRun(workspaceID, id string) (MatchRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.workspace(workspaceID).Matches[id]
	if !ok {
		return MatchRun{}, ErrNotFound
	}
	return run, nil
}

func (s *Service) StartEnrichment(workspaceID, policyID string) (EnrichmentRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	policy, ok := findPolicy(s.policies, policyID)
	if !ok || policy.Lifecycle != "ACTIVE" {
		return EnrichmentRun{}, ErrNotFound
	}
	evaluation := eligibility.Evaluate(state.Passport, policy.Rules)
	run := EnrichmentRun{ID: uuid.NewString(), PolicyID: policyID, Status: "NEEDS_REVIEW", CreatedAt: time.Now().UTC()}
	labels := passportservice.CanonicalFields()
	for _, criterion := range evaluation.Criteria {
		if criterion.Status != eligibility.StatusMissingInfo {
			continue
		}
		value := suggestedValue(criterion.FieldKey)
		if value == nil {
			continue
		}
		run.Candidates = append(run.Candidates, EnrichmentCandidate{ID: uuid.NewString(), FieldKey: criterion.FieldKey, Label: labels[criterion.FieldKey], Value: value, Confidence: .78, Status: "NEEDS_REVIEW", Warning: "Nguồn công khai cần người dùng xác nhận", Evidence: domain.Evidence{SourceID: "web-demo", SourceName: "Cổng thông tin doanh nghiệp", URL: "https://business.gov.vn/", Quote: fmt.Sprintf("Thông tin tham chiếu về %s", labels[criterion.FieldKey]), ContentHash: "sha256:demo-public-source", ObservedAt: time.Now().UTC()}})
	}
	if len(run.Candidates) == 0 {
		run.Status = "NO_RESULTS"
	}
	state.Enrichment[run.ID] = run
	return run, nil
}

func (s *Service) EnrichmentRun(workspaceID, id string) (EnrichmentRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.workspace(workspaceID).Enrichment[id]
	if !ok {
		return EnrichmentRun{}, ErrNotFound
	}
	return run, nil
}

func (s *Service) AcceptEnrichment(workspaceID, candidateID string, expectedVersion int) (domain.Passport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	for runID, run := range state.Enrichment {
		for index, candidate := range run.Candidates {
			if candidate.ID != candidateID {
				continue
			}
			if expectedVersion != state.Passport.Version {
				return domain.Passport{}, ErrConflict
			}
			merged, err := passportservice.MergeCandidate(state.Passport, passportservice.Candidate{ID: candidate.ID, FieldKey: candidate.FieldKey, Value: candidate.Value, DataType: inferType(candidate.Value), Confidence: candidate.Confidence, Evidence: candidate.Evidence})
			if err != nil {
				return domain.Passport{}, err
			}
			field := merged.Fields[candidate.FieldKey]
			field.Status = domain.FieldConfirmed
			field.Confidence = 1
			merged.Fields[candidate.FieldKey] = field
			merged.Version++
			state.Passport = merged
			run.Candidates[index].Status = "ACCEPTED"
			state.Enrichment[runID] = run
			return clonePassport(merged), nil
		}
	}
	return domain.Passport{}, ErrNotFound
}

func (s *Service) RejectEnrichment(workspaceID, candidateID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	for runID, run := range state.Enrichment {
		for index := range run.Candidates {
			if run.Candidates[index].ID == candidateID {
				run.Candidates[index].Status = "REJECTED"
				state.Enrichment[runID] = run
				return nil
			}
		}
	}
	return ErrNotFound
}

func cleanStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if cleaned := strings.TrimSpace(value); cleaned != "" && len(cleaned) <= 80 {
			result = append(result, cleaned)
		}
	}
	return slices.Compact(result)
}

func inferType(value any) string {
	switch value.(type) {
	case bool:
		return "boolean"
	case int, int64, float64:
		return "number"
	case []string:
		return "string_array"
	default:
		return "string"
	}
}
