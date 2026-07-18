package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	applicationdomain "github.com/p2b/p2b/internal/application"
	"github.com/p2b/p2b/internal/platform"
)

func (s *Server) createApplication(w http.ResponseWriter, r *http.Request) {
	if !requireIdempotency(w, r) {
		return
	}
	var input struct {
		ChecklistID string `json:"checklist_id"`
		TemplateID  string `json:"template_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	workspaceID := workspace(r)
	draftContext, err := s.service.ApplicationContext(workspaceID, input.ChecklistID)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	templateID, templateName := "", "Mẫu P2B mặc định"
	templateText := "Hồ sơ của {{company_name}} cho {{policy_title}}. Cơ quan ban hành: {{policy_agency}}. Mã số thuế: {{tax_code}}. Nguồn: {{policy_source}}."
	if strings.TrimSpace(input.TemplateID) != "" {
		if s.config.ApplicationStore == nil {
			writeError(w, http.StatusServiceUnavailable, "TEMPLATE_UNAVAILABLE", "Application template cache is not configured")
			return
		}
		template, templateErr := s.config.ApplicationStore.Template(r.Context(), workspaceID, input.TemplateID)
		if errors.Is(templateErr, applicationdomain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "TEMPLATE_NOT_FOUND", "Application template not found")
			return
		}
		if templateErr != nil {
			writeError(w, http.StatusServiceUnavailable, "TEMPLATE_UNAVAILABLE", "Application template could not be loaded")
			return
		}
		templateID, templateName, templateText = template.ID, template.Name, template.SourceText
	}
	variables := applicationdomain.TemplateVariables(draftContext.Passport, draftContext.Policy)
	sections := fallbackApplicationSections(templateText, variables)
	generationWarning := ""
	if s.config.ApplicationGenerator != nil {
		generationContext, cancelGeneration := context.WithTimeout(r.Context(), 25*time.Second)
		generated, generationErr := s.config.ApplicationGenerator.GenerateApplication(generationContext, applicationdomain.GenerationRequest{TemplateText: templateText, Variables: variables})
		cancelGeneration()
		if generationErr != nil {
			slog.WarnContext(r.Context(), "application generation fell back", "workspace_id", workspaceID, "template_id", templateID, "error", generationErr)
			generationWarning = "Gemini chưa thể hoàn tất bản nháp; hệ thống đã điền mẫu bằng dữ liệu đã xác nhận."
		} else {
			sections = generated
		}
	} else {
		generationWarning = "Gemini chưa được cấu hình; hệ thống đã điền mẫu bằng dữ liệu đã xác nhận."
	}
	application, err := s.service.CreateApplicationFromTemplate(workspaceID, input.ChecklistID, templateID, templateName, sections, generationWarning)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	if s.config.ApplicationStore != nil {
		if err = s.config.ApplicationStore.SaveDraft(r.Context(), workspaceID, application); err != nil {
			s.service.RemoveApplication(workspaceID, application.ID)
			writeError(w, http.StatusServiceUnavailable, "DRAFT_CACHE_UNAVAILABLE", "Application draft could not be cached")
			return
		}
	}
	writeJSON(w, http.StatusCreated, application)
}

func fallbackApplicationSections(templateText string, variables map[string]string) map[string]string {
	proposal := applicationdomain.RenderTemplate(templateText, variables)
	for len(proposal) > 10_000 {
		_, size := utf8.DecodeLastRuneInString(proposal)
		proposal = proposal[:len(proposal)-size]
	}
	return map[string]string{
		"company_overview": strings.TrimSpace(variables["company_name"] + " · Mã số thuế: " + valueOrMissing(variables["tax_code"])),
		"support_need":     strings.TrimSpace("Đối chiếu yêu cầu và tài liệu của " + variables["policy_title"] + "."),
		"proposal":         proposal,
	}
}

func valueOrMissing(value string) string {
	if strings.TrimSpace(value) == "" {
		return "[CẦN BỔ SUNG]"
	}
	return strings.TrimSpace(value)
}

func (s *Server) getApplication(w http.ResponseWriter, r *http.Request) {
	workspaceID, applicationID := workspace(r), chi.URLParam(r, "applicationID")
	application, err := s.service.Application(workspaceID, applicationID)
	if errors.Is(err, platform.ErrNotFound) && s.config.ApplicationStore != nil {
		application, err = s.config.ApplicationStore.Draft(r.Context(), workspaceID, applicationID)
		if err == nil {
			s.service.RestoreApplication(workspaceID, application)
		}
	}
	if err != nil {
		if errors.Is(err, applicationdomain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Application draft not found")
			return
		}
		respondServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, application)
}

func (s *Server) latestApplication(w http.ResponseWriter, r *http.Request) {
	if s.config.ApplicationStore == nil {
		writeJSON(w, http.StatusOK, map[string]any{"application": nil})
		return
	}
	application, err := s.config.ApplicationStore.LatestDraft(r.Context(), workspace(r))
	if errors.Is(err, applicationdomain.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"application": nil})
		return
	}
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "DRAFT_CACHE_UNAVAILABLE", "Application draft cache is unavailable")
		return
	}
	s.service.RestoreApplication(workspace(r), application)
	writeJSON(w, http.StatusOK, map[string]any{"application": application})
}

func (s *Server) updateApplication(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Sections        map[string]string `json:"sections"`
		ExpectedVersion int               `json:"expected_version"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	workspaceID, applicationID := workspace(r), chi.URLParam(r, "applicationID")
	previous, previousErr := s.service.Application(workspaceID, applicationID)
	if previousErr != nil {
		respondServiceError(w, previousErr)
		return
	}
	application, err := s.service.UpdateApplication(workspaceID, applicationID, input.Sections, input.ExpectedVersion)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	if s.config.ApplicationStore != nil {
		if err = s.config.ApplicationStore.SaveDraft(r.Context(), workspaceID, application); err != nil {
			s.service.RestoreApplication(workspaceID, previous)
			if errors.Is(err, applicationdomain.ErrConflict) {
				writeError(w, http.StatusConflict, "VERSION_CONFLICT", "A newer application draft already exists")
				return
			}
			writeError(w, http.StatusServiceUnavailable, "DRAFT_CACHE_UNAVAILABLE", "Application draft could not be cached")
			return
		}
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
	workspaceID, applicationID := workspace(r), chi.URLParam(r, "applicationID")
	previous, previousErr := s.service.Application(workspaceID, applicationID)
	if previousErr != nil {
		respondServiceError(w, previousErr)
		return
	}
	application, err := s.service.TransitionApplication(workspaceID, applicationID, action)
	if err != nil {
		if errors.Is(err, platform.ErrBlocked) {
			writeJSON(w, http.StatusConflict, map[string]any{"error": map[string]any{"code": "APPROVAL_BLOCKED", "message": "Required evidence is missing", "details": application.BlockingReasons}})
			return
		}
		respondServiceError(w, err)
		return
	}
	if s.config.ApplicationStore != nil {
		if err = s.config.ApplicationStore.SaveDraft(r.Context(), workspaceID, application); err != nil {
			s.service.RestoreApplication(workspaceID, previous)
			writeError(w, http.StatusServiceUnavailable, "DRAFT_CACHE_UNAVAILABLE", "Application status could not be cached")
			return
		}
	}
	writeJSON(w, http.StatusOK, application)
}

func (s *Server) listApplicationTemplates(w http.ResponseWriter, r *http.Request) {
	if s.config.ApplicationStore == nil {
		writeJSON(w, http.StatusOK, map[string]any{"templates": []applicationdomain.Template{}})
		return
	}
	templates, err := s.config.ApplicationStore.Templates(r.Context(), workspace(r))
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "TEMPLATE_UNAVAILABLE", "Application templates are unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates})
}

func (s *Server) uploadApplicationTemplate(w http.ResponseWriter, r *http.Request) {
	if !requireIdempotency(w, r) {
		return
	}
	if s.config.ApplicationStore == nil || s.config.TemplateConverter == nil {
		writeError(w, http.StatusServiceUnavailable, "TEMPLATE_UNAVAILABLE", "Application template upload is not configured")
		return
	}
	if err := r.ParseMultipartForm(maxTemplateBodyBytes); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_TEMPLATE", "Template upload is invalid")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_TEMPLATE", "Template file is required")
		return
	}
	defer file.Close()
	extension := strings.ToLower(filepath.Ext(header.Filename))
	contentTypes := map[string]string{".pdf": "application/pdf", ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document", ".txt": "text/plain"}
	contentType, ok := contentTypes[extension]
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_TEMPLATE", "Only PDF, DOCX and TXT templates are accepted")
		return
	}
	temporary, err := os.CreateTemp("", "p2b-application-template-*"+extension)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "TEMPLATE_UNAVAILABLE", "Template could not be processed")
		return
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	written, copyErr := io.Copy(temporary, io.LimitReader(file, maxTemplateFileBytes+1))
	closeErr := temporary.Close()
	if copyErr != nil || closeErr != nil || written < 1 || written > maxTemplateFileBytes {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_TEMPLATE", "Template must contain data and be at most 10 MB")
		return
	}
	if err = validateApplicationTemplateFile(temporaryPath, extension); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_TEMPLATE", "Template content does not match its file type")
		return
	}
	sourceText, err := s.config.TemplateConverter.Convert(r.Context(), temporaryPath)
	if err != nil || strings.TrimSpace(sourceText) == "" || len(sourceText) > 500_000 {
		writeError(w, http.StatusUnprocessableEntity, "TEMPLATE_CONVERSION_FAILED", "Template text could not be extracted")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(header.Filename), extension)
	}
	template, err := s.config.ApplicationStore.CreateTemplate(r.Context(), workspace(r), name, filepath.Base(header.Filename), contentType, sourceText)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "TEMPLATE_UNAVAILABLE", "Template could not be cached")
		return
	}
	writeJSON(w, http.StatusCreated, template)
}

func validateApplicationTemplateFile(path, extension string) error {
	switch extension {
	case ".pdf":
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		header := make([]byte, 5)
		if _, err = io.ReadFull(file, header); err != nil || !bytes.Equal(header, []byte("%PDF-")) {
			return errors.New("invalid PDF signature")
		}
		return nil
	case ".docx":
		archive, err := zip.OpenReader(path)
		if err != nil {
			return err
		}
		defer archive.Close()
		var total uint64
		hasContentTypes, hasDocument := false, false
		for _, file := range archive.File {
			total += file.UncompressedSize64
			if total > 50<<20 {
				return errors.New("DOCX expanded content exceeds limit")
			}
			hasContentTypes = hasContentTypes || file.Name == "[Content_Types].xml"
			hasDocument = hasDocument || file.Name == "word/document.xml"
		}
		if !hasContentTypes || !hasDocument {
			return errors.New("invalid DOCX structure")
		}
		return nil
	case ".txt":
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !utf8.Valid(content) || bytes.IndexByte(content, 0) >= 0 {
			return errors.New("TXT must be UTF-8 text")
		}
		return nil
	default:
		return errors.New("unsupported template type")
	}
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
