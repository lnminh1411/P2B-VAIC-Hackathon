package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/p2b/p2b/internal/authn"
	"github.com/p2b/p2b/internal/domain"
	passportdomain "github.com/p2b/p2b/internal/passport"
	"github.com/p2b/p2b/internal/pipeline"
	"github.com/p2b/p2b/internal/platform"
)

const maxBodyBytes = 1 << 20

type contextKey string

const workspaceKey contextKey = "workspace"
const principalKey contextKey = "principal"

type Workspace = domain.Workspace

type WorkspaceManager interface {
	Ensure(context.Context, authn.Principal) error
	Resolve(context.Context, authn.Principal, string) (string, error)
	List(context.Context, authn.Principal) ([]Workspace, error)
	Create(context.Context, authn.Principal, string) (Workspace, error)
}

type Config struct {
	DevAuth          bool
	WebOrigin        string
	Verifier         authn.Verifier
	WorkspaceManager WorkspaceManager
	UploadSigner     interface {
		CreateUploadURL(context.Context, string) (string, error)
	}
	ExtractionStore interface {
		RegisterSource(context.Context, pipeline.Source) error
		MarkUploaded(context.Context, string, string) error
		EnqueueBuild(context.Context, string, pipeline.BuildRequest) (pipeline.Job, error)
		EnqueueRefresh(context.Context, string, []string, string, string) (pipeline.Job, error)
		Job(context.Context, string, string) (pipeline.Job, error)
		Passport(context.Context, string) (domain.Passport, error)
		Candidates(context.Context, string) ([]passportdomain.Candidate, error)
		ConfirmField(context.Context, string, string, string, any, int) (domain.Passport, error)
	}
	PolicyStore interface {
		Policies(context.Context, bool) ([]domain.Policy, error)
	}
	ReadinessChecker interface {
		Ping(context.Context) error
	}
}

type Server struct {
	service       *platform.Service
	config        Config
	idempotency   *idempotencyStore
	devMu         sync.Mutex
	devWorkspaces map[string][]Workspace
}

func NewServer(service *platform.Service) http.Handler {
	return NewServerWithConfig(service, Config{DevAuth: true, WebOrigin: "http://localhost:5173"})
}

func NewServerWithConfig(service *platform.Service, config Config) http.Handler {
	server := &Server{service: service, config: config, idempotency: newIdempotencyStore(), devWorkspaces: map[string][]Workspace{}}
	router := chi.NewRouter()
	router.Use(server.recoverer, server.securityHeaders, server.cors, server.limitBody)
	router.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "live"})
	})
	router.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		if config.ReadinessChecker == nil {
			if config.DevAuth {
				writeJSON(w, http.StatusOK, map[string]string{"status": "ready", "mode": "development"})
				return
			}
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := config.ReadinessChecker.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready", "mode": "production"})
	})
	router.Route("/v1", func(r chi.Router) {
		r.Use(server.authenticate)
		r.Use(server.idempotencyMiddleware)
		r.Get("/auth/me", server.authMe)
		r.Get("/workspaces", server.listWorkspaces)
		r.Post("/workspaces", server.createWorkspace)
		r.Post("/uploads/presign", server.presignUpload)
		r.Post("/uploads/{sourceID}/complete", server.completeUpload)
		r.Post("/passports/build", server.buildPassport)
		r.Post("/passports/refresh", server.refreshPassport)
		r.Get("/jobs/{jobID}", server.getJob)
		r.Get("/passport", server.getPassport)
		r.Get("/passport/versions", server.getPassportVersions)
		r.Get("/passport/candidates", server.getCandidates)
		r.Put("/passport/fields/{fieldKey}", server.confirmField)
		r.Post("/matches", server.createMatch)
		r.Get("/matches/{matchID}", server.getMatch)
		r.Get("/policies", server.listPolicies)
		r.Get("/policies/{policyID}/versions/{version}", server.getPolicy)
		r.Post("/enrichment-runs", server.startEnrichment)
		r.Get("/enrichment-runs/{runID}", server.getEnrichment)
		r.Post("/enrichment-candidates/{candidateID}/accept", server.acceptEnrichment)
		r.Post("/enrichment-candidates/{candidateID}/reject", server.rejectEnrichment)
		r.Post("/checklists", server.createChecklist)
		r.Get("/checklists/{checklistID}", server.getChecklist)
		r.Put("/checklists/{checklistID}/items/{itemID}", server.updateChecklistItem)
		r.Post("/applications", server.createApplication)
		r.Get("/applications/{applicationID}", server.getApplication)
		r.Put("/applications/{applicationID}", server.updateApplication)
		r.Post("/applications/{applicationID}/{action}", server.applicationAction)
		r.Get("/applications/{applicationID}/download", server.downloadApplication)
		r.Get("/alerts", server.listAlerts)
		r.Post("/alerts/{alertID}/read", server.readAlert)
		r.With(server.requireAdmin).Get("/admin/policies", server.adminPolicies)
	})
	return router
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workspaceID := ""
		principal := authn.Principal{}
		if s.config.DevAuth {
			workspaceID = strings.TrimSpace(r.Header.Get("X-Workspace-ID"))
			if workspaceID == "" {
				workspaceID = "local-development-workspace"
			}
			principal = authn.Principal{Subject: "local-development-account", Email: "founder@p2b.local", Name: "P2B Founder", Roles: []string{"admin"}}
		} else {
			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok || s.config.Verifier == nil {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Valid Supabase bearer token required")
				return
			}
			verified, err := s.config.Verifier.Verify(r.Context(), token)
			if err != nil {
				if errors.Is(err, authn.ErrVerifierUnavailable) {
					writeError(w, http.StatusServiceUnavailable, "IDENTITY_UNAVAILABLE", "Identity service is temporarily unavailable")
					return
				}
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Valid Supabase bearer token required")
				return
			}
			principal = verified
			workspaceID = verified.Subject
		}
		if workspaceID == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Valid Supabase bearer token required")
			return
		}
		if !s.config.DevAuth && s.config.WorkspaceManager != nil {
			if err := s.config.WorkspaceManager.Ensure(r.Context(), principal); err != nil {
				slog.ErrorContext(r.Context(), "workspace bootstrap failed", "error", err)
				writeError(w, http.StatusServiceUnavailable, "WORKSPACE_UNAVAILABLE", "Workspace is temporarily unavailable")
				return
			}
			selected, err := s.config.WorkspaceManager.Resolve(r.Context(), principal, r.Header.Get("X-Workspace-ID"))
			if err != nil {
				writeError(w, http.StatusForbidden, "WORKSPACE_FORBIDDEN", "You do not have access to this business workspace")
				return
			}
			workspaceID = selected
		}
		ctx := context.WithValue(r.Context(), workspaceKey, workspaceID)
		ctx = context.WithValue(ctx, principalKey, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	currentPrincipal := principal(r)
	if s.config.WorkspaceManager != nil && !s.config.DevAuth {
		workspaces, err := s.config.WorkspaceManager.List(r.Context(), currentPrincipal)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "WORKSPACES_UNAVAILABLE", "Business workspaces are temporarily unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"workspaces": workspaces, "active_workspace_id": workspace(r)})
		return
	}
	workspaces := s.devWorkspaceList(currentPrincipal.Subject, workspace(r))
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": workspaces, "active_workspace_id": workspace(r)})
}

func (s *Server) createWorkspace(w http.ResponseWriter, r *http.Request) {
	if !requireIdempotency(w, r) {
		return
	}
	var input struct {
		DisplayName string `json:"display_name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.DisplayName == "" || len([]rune(input.DisplayName)) > 200 {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_INPUT", "display_name is required and limited to 200 characters")
		return
	}
	if s.config.WorkspaceManager != nil && !s.config.DevAuth {
		created, err := s.config.WorkspaceManager.Create(r.Context(), principal(r), input.DisplayName)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "WORKSPACE_UNAVAILABLE", "Business workspace could not be created")
			return
		}
		writeJSON(w, http.StatusCreated, created)
		return
	}
	created := Workspace{ID: uuid.NewString(), DisplayName: input.DisplayName, Role: "OWNER", CreatedAt: time.Now().UTC()}
	currentPrincipal := principal(r)
	s.devMu.Lock()
	s.devWorkspaces[currentPrincipal.Subject] = append(s.devWorkspaces[currentPrincipal.Subject], created)
	s.devMu.Unlock()
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) devWorkspaceList(principalID, activeID string) []Workspace {
	s.devMu.Lock()
	defer s.devMu.Unlock()
	workspaces := append([]Workspace(nil), s.devWorkspaces[principalID]...)
	if len(workspaces) == 0 {
		workspaces = []Workspace{{ID: activeID, DisplayName: activeID, Role: "OWNER", CreatedAt: time.Now().UTC()}}
		s.devWorkspaces[principalID] = append([]Workspace(nil), workspaces...)
	}
	return workspaces
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := r.Context().Value(principalKey).(authn.Principal)
		if !ok || !principal.HasRole("admin") {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Administrator role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	return token, ok && strings.EqualFold(scheme, "Bearer") && token != "" && !strings.ContainsAny(token, " \t\r\n") && len(token) <= 8192
}

func (s *Server) authMe(w http.ResponseWriter, r *http.Request) {
	principal, ok := r.Context().Value(principalKey).(authn.Principal)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Valid Supabase bearer token required")
		return
	}
	writeJSON(w, http.StatusOK, principal)
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origin == s.config.WebOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, If-Match, X-Workspace-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > maxBodyBytes {
			writeError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "Request body exceeds 1 MB")
			return
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL", "Unexpected server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) presignUpload(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		SizeBytes   int64  `json:"size_bytes"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if !strings.HasSuffix(strings.ToLower(input.Filename), ".pdf") || input.ContentType != "application/pdf" || input.SizeBytes < 1 || input.SizeBytes > 20<<20 {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_PDF", "Only PDF files up to 20 MB are accepted")
		return
	}
	sourceID := uuid.NewString()
	if s.config.UploadSigner == nil {
		writeError(w, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "Private upload is not configured")
		return
	}
	objectKey := workspace(r) + "/sources/" + sourceID + ".pdf"
	uploadURL, err := s.config.UploadSigner.CreateUploadURL(r.Context(), objectKey)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "Private upload is temporarily unavailable")
		return
	}
	if s.config.ExtractionStore != nil {
		if err = s.config.ExtractionStore.RegisterSource(r.Context(), pipeline.Source{ID: sourceID, WorkspaceID: workspace(r), Filename: input.Filename, ContentType: input.ContentType, SizeBytes: input.SizeBytes, ObjectKey: objectKey}); err != nil {
			slog.ErrorContext(r.Context(), "register upload source failed", "error", err)
			writeError(w, http.StatusServiceUnavailable, "SOURCE_UNAVAILABLE", "Upload source could not be registered")
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"source_id": sourceID, "object_key": objectKey, "upload_url": uploadURL, "expires_in": 7200, "mode": "SUPABASE_SIGNED"})
}

func (s *Server) completeUpload(w http.ResponseWriter, r *http.Request) {
	if !requireIdempotency(w, r) {
		return
	}
	if s.config.ExtractionStore == nil {
		writeError(w, http.StatusServiceUnavailable, "EXTRACTION_UNAVAILABLE", "Extraction pipeline is not configured")
		return
	}
	sourceID := chi.URLParam(r, "sourceID")
	if _, err := uuid.Parse(sourceID); err != nil {
		writeError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", "Upload source not found")
		return
	}
	if err := s.config.ExtractionStore.MarkUploaded(r.Context(), workspace(r), sourceID); err != nil {
		if errors.Is(err, pipeline.ErrNotFound) {
			writeError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", "Upload source not found")
			return
		}
		writeError(w, http.StatusServiceUnavailable, "SOURCE_UNAVAILABLE", "Upload source could not be finalized")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) buildPassport(w http.ResponseWriter, r *http.Request) {
	if !requireIdempotency(w, r) {
		return
	}
	var input platform.BuildPassportInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := platform.ValidateBuildPassportInput(&input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_INPUT", err.Error())
		return
	}
	if s.config.ExtractionStore != nil {
		principal := principal(r)
		job, err := s.config.ExtractionStore.EnqueueBuild(r.Context(), workspace(r), pipeline.BuildRequest{
			CompanyName: input.CompanyName, Website: input.Website, SupportNeeds: input.SupportNeeds, SourceIDs: input.SourceIDs,
			IdempotencyKey: r.Header.Get("Idempotency-Key"), ActorSubject: principal.Subject,
		})
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "INVALID_INPUT", err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, job)
		return
	}
	job, err := s.service.BuildPassport(workspace(r), input)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_INPUT", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (s *Server) refreshPassport(w http.ResponseWriter, r *http.Request) {
	if !requireIdempotency(w, r) {
		return
	}
	var input struct {
		SourceIDs []string `json:"source_ids"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	request := pipeline.BuildRequest{SourceIDs: input.SourceIDs}
	if err := pipeline.ValidateRefreshRequest(request); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_INPUT", err.Error())
		return
	}
	if s.config.ExtractionStore == nil {
		writeError(w, http.StatusServiceUnavailable, "EXTRACTION_UNAVAILABLE", "Extraction pipeline is not configured")
		return
	}
	job, err := s.config.ExtractionStore.EnqueueRefresh(r.Context(), workspace(r), input.SourceIDs, r.Header.Get("Idempotency-Key"), principal(r).Subject)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_INPUT", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	if s.config.ExtractionStore != nil {
		job, err := s.config.ExtractionStore.Job(r.Context(), workspace(r), chi.URLParam(r, "jobID"))
		if err != nil {
			if errors.Is(err, pipeline.ErrNotFound) {
				writeError(w, http.StatusNotFound, "NOT_FOUND", "Job not found")
				return
			}
			writeError(w, http.StatusServiceUnavailable, "PIPELINE_UNAVAILABLE", "Extraction status is temporarily unavailable")
			return
		}
		writeJSON(w, http.StatusOK, job)
		return
	}
	job, err := s.service.Job(workspace(r), chi.URLParam(r, "jobID"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) getPassport(w http.ResponseWriter, r *http.Request) {
	if s.config.ExtractionStore != nil {
		passport, err := s.config.ExtractionStore.Passport(r.Context(), workspace(r))
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "PASSPORT_UNAVAILABLE", "Company Passport is temporarily unavailable")
			return
		}
		writeJSON(w, http.StatusOK, passport)
		return
	}
	writeJSON(w, http.StatusOK, s.service.Passport(workspace(r)))
}
func (s *Server) getPassportVersions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []any{s.service.Passport(workspace(r))})
}
func (s *Server) getCandidates(w http.ResponseWriter, r *http.Request) {
	if s.config.ExtractionStore != nil {
		candidates, err := s.config.ExtractionStore.Candidates(r.Context(), workspace(r))
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "CANDIDATES_UNAVAILABLE", "Extracted candidates are temporarily unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"candidates": candidates})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"candidates": s.service.Candidates(workspace(r))})
}

func (s *Server) confirmField(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Value           any `json:"value"`
		ExpectedVersion int `json:"expected_version"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if s.config.ExtractionStore != nil {
		pass, err := s.config.ExtractionStore.ConfirmField(r.Context(), workspace(r), principal(r).Subject, chi.URLParam(r, "fieldKey"), input.Value, input.ExpectedVersion)
		if err != nil {
			if errors.Is(err, pipeline.ErrVersionConflict) {
				writeError(w, http.StatusConflict, "VERSION_CONFLICT", "Passport changed; reload before confirming")
				return
			}
			writeError(w, http.StatusUnprocessableEntity, "INVALID_FIELD", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, pass)
		return
	}
	pass, err := s.service.ConfirmField(workspace(r), chi.URLParam(r, "fieldKey"), input.Value, input.ExpectedVersion)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pass)
}

func (s *Server) createMatch(w http.ResponseWriter, r *http.Request) {
	if !requireIdempotency(w, r) {
		return
	}
	var input map[string]any
	if !decodeJSONAllowEmpty(w, r, &input) {
		return
	}
	if err := s.refreshPolicies(r.Context(), true); err != nil {
		slog.Error("refresh policies for matching", "error", err)
		writeError(w, http.StatusServiceUnavailable, "POLICY_STORE_UNAVAILABLE", "Policy corpus is temporarily unavailable")
		return
	}
	if s.config.ExtractionStore != nil {
		pass, err := s.config.ExtractionStore.Passport(r.Context(), workspace(r))
		if err != nil {
			slog.Error("load passport for matching", "error", err)
			writeError(w, http.StatusServiceUnavailable, "PASSPORT_STORE_UNAVAILABLE", "Company passport is temporarily unavailable")
			return
		}
		writeJSON(w, http.StatusCreated, s.service.MatchPassport(workspace(r), pass))
		return
	}
	writeJSON(w, http.StatusCreated, s.service.Match(workspace(r)))
}

func (s *Server) getMatch(w http.ResponseWriter, r *http.Request) {
	match, err := s.service.MatchRun(workspace(r), chi.URLParam(r, "matchID"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, match)
}

func (s *Server) listPolicies(w http.ResponseWriter, r *http.Request) {
	if err := s.refreshPolicies(r.Context(), true); err != nil {
		writeError(w, http.StatusServiceUnavailable, "POLICY_STORE_UNAVAILABLE", "Policy corpus is temporarily unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"policies": s.service.Policies(true)})
}

func (s *Server) getPolicy(w http.ResponseWriter, r *http.Request) {
	if err := s.refreshPolicies(r.Context(), true); err != nil {
		writeError(w, http.StatusServiceUnavailable, "POLICY_STORE_UNAVAILABLE", "Policy corpus is temporarily unavailable")
		return
	}
	version, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_VERSION", "Policy version must be an integer")
		return
	}
	policy, err := s.service.Policy(chi.URLParam(r, "policyID"), version)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

func (s *Server) startEnrichment(w http.ResponseWriter, r *http.Request) {
	if !requireIdempotency(w, r) {
		return
	}
	var input struct {
		PolicyID string `json:"policy_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	run, err := s.service.StartEnrichment(workspace(r), input.PolicyID)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}
func (s *Server) getEnrichment(w http.ResponseWriter, r *http.Request) {
	run, err := s.service.EnrichmentRun(workspace(r), chi.URLParam(r, "runID"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) acceptEnrichment(w http.ResponseWriter, r *http.Request) {
	if !requireIdempotency(w, r) {
		return
	}
	var input struct {
		ExpectedVersion int `json:"expected_version"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	pass, err := s.service.AcceptEnrichment(workspace(r), chi.URLParam(r, "candidateID"), input.ExpectedVersion)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pass)
}
func (s *Server) rejectEnrichment(w http.ResponseWriter, r *http.Request) {
	if !requireIdempotency(w, r) {
		return
	}
	if err := s.service.RejectEnrichment(workspace(r), chi.URLParam(r, "candidateID")); err != nil {
		respondServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) createChecklist(w http.ResponseWriter, r *http.Request) {
	if !requireIdempotency(w, r) {
		return
	}
	var input struct {
		PolicyID string `json:"policy_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	checklist, err := s.service.CreateChecklist(workspace(r), input.PolicyID)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, checklist)
}
func (s *Server) getChecklist(w http.ResponseWriter, r *http.Request) {
	checklist, err := s.service.Checklist(workspace(r), chi.URLParam(r, "checklistID"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, checklist)
}

func (s *Server) updateChecklistItem(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status          string `json:"status"`
		EvidenceSource  string `json:"evidence_source"`
		ExpectedVersion int    `json:"expected_version"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	checklist, err := s.service.UpdateChecklistItem(workspace(r), chi.URLParam(r, "checklistID"), chi.URLParam(r, "itemID"), input.Status, input.EvidenceSource, input.ExpectedVersion)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, checklist)
}

func (s *Server) createApplication(w http.ResponseWriter, r *http.Request) {
	if !requireIdempotency(w, r) {
		return
	}
	var input struct {
		ChecklistID string `json:"checklist_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	application, err := s.service.CreateApplication(workspace(r), input.ChecklistID)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, application)
}
func (s *Server) getApplication(w http.ResponseWriter, r *http.Request) {
	application, err := s.service.Application(workspace(r), chi.URLParam(r, "applicationID"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, application)
}

func (s *Server) updateApplication(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Sections        map[string]string `json:"sections"`
		ExpectedVersion int               `json:"expected_version"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	application, err := s.service.UpdateApplication(workspace(r), chi.URLParam(r, "applicationID"), input.Sections, input.ExpectedVersion)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, application)
}

func (s *Server) applicationAction(w http.ResponseWriter, r *http.Request) {
	if !requireIdempotency(w, r) {
		return
	}
	action := chi.URLParam(r, "action")
	if action != "submit" && action != "approve" && action != "generate" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Action not found")
		return
	}
	application, err := s.service.TransitionApplication(workspace(r), chi.URLParam(r, "applicationID"), action)
	if err != nil {
		if errors.Is(err, platform.ErrBlocked) {
			writeJSON(w, http.StatusConflict, map[string]any{"error": map[string]any{"code": "APPROVAL_BLOCKED", "message": "Required evidence is missing", "details": application.BlockingReasons}})
			return
		}
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, application)
}

func (s *Server) downloadApplication(w http.ResponseWriter, r *http.Request) {
	data, filename, err := s.service.ApplicationPDF(workspace(r), chi.URLParam(r, "applicationID"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) listAlerts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"alerts": s.service.Alerts(workspace(r))})
}
func (s *Server) readAlert(w http.ResponseWriter, r *http.Request) {
	alert, err := s.service.ReadAlert(workspace(r), chi.URLParam(r, "alertID"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, alert)
}
func (s *Server) adminPolicies(w http.ResponseWriter, r *http.Request) {
	if err := s.refreshPolicies(r.Context(), false); err != nil {
		writeError(w, http.StatusServiceUnavailable, "POLICY_STORE_UNAVAILABLE", "Policy corpus is temporarily unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"policies": s.service.Policies(false)})
}

func (s *Server) refreshPolicies(ctx context.Context, activeOnly bool) error {
	if s.config.PolicyStore == nil {
		return nil
	}
	policies, err := s.config.PolicyStore.Policies(ctx, activeOnly)
	if err != nil {
		return err
	}
	s.service.ReplacePolicies(policies)
	return nil
}

func workspace(r *http.Request) string {
	value, _ := r.Context().Value(workspaceKey).(string)
	return value
}

func principal(r *http.Request) authn.Principal {
	value, _ := r.Context().Value(principalKey).(authn.Principal)
	return value
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) || strings.Contains(err.Error(), "request body too large") {
			writeError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "Request body exceeds 1 MB")
		} else {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body is invalid")
		}
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request must contain one JSON object")
		return false
	}
	return true
}

func decodeJSONAllowEmpty(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Body == nil {
		return true
	}
	err := json.NewDecoder(r.Body).Decode(target)
	if err == io.EOF {
		return true
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body is invalid")
		return false
	}
	return true
}

func requireIdempotency(w http.ResponseWriter, r *http.Request) bool {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 200 {
		writeError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "A valid Idempotency-Key header is required")
		return false
	}
	return true
}

func respondServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, platform.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Resource not found")
	case errors.Is(err, platform.ErrConflict):
		writeError(w, http.StatusConflict, "VERSION_CONFLICT", "Resource changed; reload and try again")
	default:
		writeError(w, http.StatusUnprocessableEntity, "INVALID_OPERATION", err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func EnvConfig() (Config, error) {
	config := Config{DevAuth: strings.EqualFold(os.Getenv("DEV_AUTH"), "true"), WebOrigin: env("WEB_ORIGIN", "http://localhost:5173")}
	if config.DevAuth {
		return config, nil
	}
	verifier, err := authn.NewSupabaseVerifier(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_PUBLISHABLE_KEY"), nil)
	if err != nil {
		return Config{}, err
	}
	config.Verifier = verifier
	return config, nil
}
func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
