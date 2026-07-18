package platform

import (
	"errors"
	"fmt"
	"math"
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

func NewService(policies []domain.Policy) *Service {
	return &Service{workspaces: map[string]*workspaceState{}, policies: slices.Clone(policies)}
}

func (s *Service) ReplacePolicies(policies []domain.Policy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies = slices.Clone(policies)
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
		RetrievedPolicies: map[string]domain.Policy{},
		Checklists:        map[string]Checklist{}, Applications: map[string]Application{}, Alerts: []Alert{},
	}
	s.workspaces[id] = state
	return state
}

func (s *Service) BuildPassport(workspaceID string, input BuildPassportInput) (Job, error) {
	if err := ValidateBuildPassportInput(&input); err != nil {
		return Job{}, err
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
	state.Candidates = nil
	job := Job{ID: uuid.NewString(), Type: "PASSPORT_BUILD", Status: "SUCCEEDED", Progress: 100, CreatedAt: now}
	state.Jobs[job.ID] = job
	return job, nil
}

func ValidateBuildPassportInput(input *BuildPassportInput) error {
	input.CompanyName = strings.TrimSpace(input.CompanyName)
	if input.CompanyName == "" || len(input.CompanyName) > 200 {
		return errors.New("company_name is required and limited to 200 characters")
	}
	if len(input.SourceNames) > 10 {
		return errors.New("at most 10 PDF sources are allowed")
	}
	if len(input.SourceIDs) > 10 {
		return errors.New("at most 10 PDF sources are allowed")
	}
	if input.Website != "" && !strings.HasPrefix(strings.ToLower(input.Website), "https://") {
		return errors.New("website must use https")
	}
	return nil
}

func (s *Service) Passport(workspaceID string) domain.Passport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return passportservice.EnsureCanonicalFields(clonePassport(s.workspace(workspaceID).Passport))
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
	return passportservice.EnsureCanonicalFields(clonePassport(updated)), nil
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
	return s.matchLocked(state, state.Passport, nil, "RULE_ENGINE_ONLY")
}

// MatchPassport matches using the persisted passport supplied by the API layer.
// Production passport writes live in PostgreSQL, while Service keeps only transient
// match/application state.
func (s *Service) MatchPassport(workspaceID string, pass domain.Passport) MatchRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	state.Passport = clonePassport(pass)
	return s.matchLocked(state, pass, nil, "RULE_ENGINE_ONLY")
}

// MatchPassportHybrid combines deterministic eligibility evaluation for reviewed
// policies with semantic retrieval from the legal-document corpus.
func (s *Service) MatchPassportHybrid(workspaceID string, pass domain.Passport, documents []domain.DocumentMatch, retrievalMode string) MatchRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	state.Passport = clonePassport(pass)
	if strings.TrimSpace(retrievalMode) == "" {
		retrievalMode = "HYBRID_RULE_VECTOR"
	}
	return s.matchLocked(state, pass, documents, retrievalMode)
}

func (s *Service) matchLocked(state *workspaceState, pass domain.Passport, documents []domain.DocumentMatch, retrievalMode string) MatchRun {
	run := MatchRun{ID: uuid.NewString(), PassportVersion: pass.Version, CreatedAt: time.Now().UTC(), Results: make([]MatchResult, 0)}
	knownSources := make(map[string]struct{}, len(s.policies))
	for _, policy := range s.policies {
		if policy.Lifecycle != "ACTIVE" {
			continue
		}
		evaluation := eligibility.Evaluate(pass, policy.Rules)
		score, reasons := rank(policy, pass, evaluation)
		mode := "RULE_ENGINE_ONLY"
		if len(documents) > 0 {
			mode = retrievalMode
		}
		run.Results = append(run.Results, MatchResult{PolicyID: policy.ID, PolicyVersion: policy.Version, Title: policy.Title, Agency: policy.Agency, Benefit: policy.Benefit, BenefitAmount: policy.BenefitAmount, Deadline: policy.Deadline, Score: score, Eligibility: evaluation, RankingReasons: reasons, TemplateReady: policy.TemplateReady, RetrievalMode: mode, SourceURL: policy.SourceURL})
		if policy.SourceURL != "" {
			knownSources[policy.SourceURL] = struct{}{}
		}
	}
	for _, document := range documents {
		if document.ID == "" {
			continue
		}
		if _, duplicate := knownSources[document.SourceURL]; document.SourceURL != "" && duplicate {
			continue
		}
		score := documentMatchScore(document)
		reasons := []string{"Văn bản được tìm thấy trong kho dữ liệu pháp luật"}
		if document.VectorScore > 0 {
			reasons = append(reasons, fmt.Sprintf("Độ tương đồng ngữ nghĩa %.0f%%", math.Min(document.VectorScore, 1)*100))
		}
		if document.LexicalScore > 0 {
			reasons = append(reasons, "Có từ khóa phù hợp với Company Passport")
		}
		benefit := compactLegalExcerpt(document.Excerpt)
		evaluation := retrievedDocumentCriteria(document, benefit)
		workingPolicy := retrievedDocumentPolicy(document, benefit)
		state.RetrievedPolicies[document.ID] = workingPolicy
		run.Results = append(run.Results, MatchResult{
			PolicyID: document.ID, PolicyVersion: document.Version, Title: document.Title,
			Agency: document.Agency, Benefit: benefit, Score: score,
			Eligibility: evaluation, RankingReasons: reasons, RetrievalMode: retrievalMode,
			SourceURL: document.SourceURL, TemplateReady: workingPolicy.TemplateReady,
		})
	}
	sort.SliceStable(run.Results, func(i, j int) bool { return run.Results[i].Score > run.Results[j].Score })
	state.Matches[run.ID] = run
	return run
}

func (s *Service) LoadMatchRun(workspaceID string, run MatchRun) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.workspace(workspaceID)
	state.Matches[run.ID] = run
	// Also register retrieved policies so checklist/document loading can find them
	for _, result := range run.Results {
		if result.RetrievalMode != "RULE_ENGINE_ONLY" && result.RetrievalMode != "" {
			state.RetrievedPolicies[result.PolicyID] = domain.Policy{
				ID:            result.PolicyID,
				Version:       result.PolicyVersion,
				Title:         result.Title,
				Agency:        result.Agency,
				Benefit:       result.Benefit,
				TemplateReady: result.TemplateReady,
				Lifecycle:     "ACTIVE",
			}
		}
	}
}

func compactLegalExcerpt(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	for strings.HasPrefix(value, "Giới thiệu Giới thiệu ") {
		value = strings.TrimPrefix(value, "Giới thiệu ")
	}
	const maxRunes = 320
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	cut := maxRunes - 1
	for cut > 220 && runes[cut] != ' ' {
		cut--
	}
	return strings.TrimSpace(string(runes[:cut])) + "…"
}

func retrievedDocumentCriteria(document domain.DocumentMatch, excerpt string) eligibility.Result {
	citation := domain.Evidence{
		SourceID: "legal-document:" + document.ID, SourceName: document.Title,
		URL: document.SourceURL, Quote: excerpt, ContentHash: document.ID, ObservedAt: time.Now().UTC(),
	}
	descriptions := []string{
		"Xác nhận doanh nghiệp thuộc đối tượng và phạm vi áp dụng của văn bản",
		"Đối chiếu điều kiện, ngoại lệ và thời hạn tại văn bản nguồn",
		"Xác nhận bộ hồ sơ và tài liệu bắt buộc với cơ quan ban hành",
	}
	criteria := make([]eligibility.CriterionResult, 0, len(descriptions))
	for index, description := range descriptions {
		criteria = append(criteria, eligibility.CriterionResult{
			RuleID: fmt.Sprintf("document-review-%d", index+1), Description: description,
			Status: eligibility.StatusMissingInfo, Expected: "Được người phụ trách xác nhận",
			Operator: domain.OpExists, Citation: citation, Required: true,
		})
	}
	return eligibility.Result{Status: eligibility.StatusMissingInfo, Criteria: criteria}
}

func retrievedDocumentPolicy(document domain.DocumentMatch, benefit string) domain.Policy {
	return domain.Policy{
		ID: document.ID, Version: document.Version, Title: document.Title, Agency: document.Agency,
		Benefit: benefit, Lifecycle: "ACTIVE", VerifiedAt: time.Now().UTC(), SourceURL: document.SourceURL,
		// This is P2B's working PDF package, not an official agency form.
		TemplateReady: true,
		Checklist: []domain.ChecklistTemplateItem{
			{Key: "company_profile", Title: "Thông tin pháp lý doanh nghiệp", Description: "Dữ liệu pháp lý đã xác nhận trong Company Passport", Required: true, FieldKeys: []string{"legal_name"}},
			{Key: "applicability_review", Title: "Biên bản đối chiếu phạm vi áp dụng", Description: "Người phụ trách xác nhận đối tượng, điều kiện, ngoại lệ và thời hạn theo văn bản nguồn", Required: true},
			{Key: "required_documents_review", Title: "Danh mục tài liệu theo văn bản nguồn", Description: "Đối chiếu bộ hồ sơ bắt buộc và mẫu chính thức với cơ quan ban hành", Required: true},
		},
	}
}

func documentMatchScore(document domain.DocumentMatch) int {
	vector := math.Max(0, math.Min(document.VectorScore, 1))
	lexical := math.Max(0, math.Min(document.LexicalScore, 1))
	score := 25 + int(math.Round(vector*60)) + int(math.Round(lexical*14))
	if score > 99 {
		return 99
	}
	return score
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
	policy, ok := s.findWorkspacePolicy(state, policyID)
	if !ok || policy.Lifecycle != "ACTIVE" {
		return EnrichmentRun{}, ErrNotFound
	}
	run := EnrichmentRun{ID: uuid.NewString(), PolicyID: policyID, Status: "NO_RESULTS", Candidates: []EnrichmentCandidate{}, CreatedAt: time.Now().UTC()}
	state.Enrichment[run.ID] = run
	return run, nil
}

func (s *Service) findWorkspacePolicy(state *workspaceState, policyID string) (domain.Policy, bool) {
	if policy, ok := findPolicy(s.policies, policyID); ok {
		return policy, true
	}
	policy, ok := state.RetrievedPolicies[policyID]
	return policy, ok
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
