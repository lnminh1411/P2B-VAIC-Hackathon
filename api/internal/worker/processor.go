package worker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/p2b/p2b/internal/extraction"
	passportservice "github.com/p2b/p2b/internal/passport"
	"github.com/p2b/p2b/internal/pipeline"
)

var ErrInvalidPDF = errors.New("downloaded object is not a valid PDF")

const maxGeminiChunkBytes = 200 << 10

type Store interface {
	Sources(context.Context, string, []string) ([]pipeline.SourceRecord, error)
	StartSource(context.Context, string) error
	CompleteSource(context.Context, string, string, string, string) error
	FailSource(context.Context, string, string) error
	SaveCandidates(context.Context, string, string, string, string, []extraction.Candidate) error
	SetJobProgress(context.Context, string, int) error
	CompleteJob(context.Context, string) error
	FailJob(context.Context, pipeline.Job, error) error
}

type Downloader interface {
	Download(context.Context, string, int64) ([]byte, error)
}

type Converter interface {
	Convert(context.Context, string) (string, error)
}

type Extractor interface {
	Extract(context.Context, string) ([]extraction.Candidate, error)
}

type TargetedExtractor interface {
	ExtractFields(context.Context, string, []string) ([]extraction.Candidate, error)
}

type Processor struct {
	Store      Store
	Downloader Downloader
	Converter  Converter
	Extractor  Extractor
	Model      string
}

func (p Processor) Process(ctx context.Context, job pipeline.Job) error {
	if p.Store == nil || p.Downloader == nil || p.Converter == nil || p.Extractor == nil {
		return errors.New("extraction processor is not configured")
	}
	sources, err := p.Store.Sources(ctx, job.WorkspaceID, job.Payload.SourceIDs)
	if err != nil {
		return p.fail(ctx, job, err)
	}
	for index, source := range sources {
		if source.Status == "EXTRACTED" {
			continue
		}
		if err = p.Store.StartSource(ctx, source.ID); err != nil {
			return p.fail(ctx, job, err)
		}
		leaseProgress := 10 + (index * 85 / max(len(sources), 1))
		if err = p.processSource(ctx, job.ID, job.WorkspaceID, source, leaseProgress); err != nil {
			_ = p.Store.FailSource(ctx, source.ID, err.Error())
			return p.fail(ctx, job, err)
		}
		progress := 10 + ((index + 1) * 85 / max(len(sources), 1))
		if err = p.Store.SetJobProgress(ctx, job.ID, progress); err != nil {
			return p.fail(ctx, job, err)
		}
	}
	if err = p.Store.CompleteJob(ctx, job.ID); err != nil {
		return err
	}
	return nil
}

func (p Processor) processSource(ctx context.Context, jobID, workspaceID string, source pipeline.SourceRecord, leaseProgress int) error {
	content, err := p.Downloader.Download(ctx, source.ObjectKey, source.SizeBytes)
	if err != nil {
		return fmt.Errorf("download %s: %w", source.Filename, err)
	}
	path, contentHash, err := createIsolatedSourceFile(content)
	if err != nil {
		return err
	}
	defer os.Remove(path)

	markdown, err := p.Converter.Convert(ctx, path)
	if err != nil {
		return fmt.Errorf("convert %s: %w", source.Filename, err)
	}
	allCandidates, err := p.extractCandidates(ctx, jobID, source, markdown, leaseProgress)
	if err != nil {
		return err
	}
	if err = p.Store.SaveCandidates(ctx, workspaceID, source.ID, source.Filename, contentHash, allCandidates); err != nil {
		return err
	}
	return p.Store.CompleteSource(ctx, source.ID, contentHash, markdown, p.Model)
}

func createIsolatedSourceFile(content []byte) (string, string, error) {
	prefixLength := min(len(content), 1024)
	if prefixLength < 5 || !bytes.Contains(content[:prefixLength], []byte("%PDF-")) {
		return "", "", ErrInvalidPDF
	}
	hash := sha256.Sum256(content)
	contentHash := hex.EncodeToString(hash[:])
	file, err := os.CreateTemp("", "p2b-source-*.pdf")
	if err != nil {
		return "", "", fmt.Errorf("create isolated source file: %w", err)
	}
	path := file.Name()
	if err = file.Chmod(0o600); err == nil {
		_, err = file.Write(content)
	}
	closeErr := file.Close()
	if err != nil {
		_ = os.Remove(path)
		return "", "", fmt.Errorf("write isolated source file: %w", err)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", "", fmt.Errorf("close isolated source file: %w", closeErr)
	}
	return path, contentHash, nil
}

func (p Processor) extractCandidates(ctx context.Context, jobID string, source pipeline.SourceRecord, markdown string, leaseProgress int) ([]extraction.Candidate, error) {
	markdownChunks := chunks(markdown, maxGeminiChunkBytes)
	slog.Info("source text extracted", "source_id", source.ID, "markdown_bytes", len(markdown), "chunks", len(markdownChunks))
	allCandidates := make([]extraction.Candidate, 0)
	for chunkIndex, chunk := range markdownChunks {
		candidates, extractErr := p.Extractor.Extract(ctx, chunk)
		if extractErr != nil {
			return nil, fmt.Errorf("extract %s with Gemini: %w", source.Filename, extractErr)
		}
		valid, rejected := extraction.ValidateCandidates(chunk, candidates)
		logExtractionDiagnostics(source.ID, chunkIndex, "initial", candidates, valid, rejected)
		allCandidates = append(allCandidates, valid...)
		if err := p.Store.SetJobProgress(ctx, jobID, leaseProgress); err != nil {
			return nil, fmt.Errorf("renew extraction job lease: %w", err)
		}

		targetedExtractor, supportsTargetedExtraction := p.Extractor.(TargetedExtractor)
		if !supportsTargetedExtraction {
			continue
		}
		missingFields := missingCanonicalFields(allCandidates)
		if len(missingFields) == 0 {
			continue
		}
		targetedCandidates, targetedErr := targetedExtractor.ExtractFields(ctx, chunk, missingFields)
		if targetedErr != nil {
			return nil, fmt.Errorf("complete %s with Gemini: %w", source.Filename, targetedErr)
		}
		targetedCandidates = candidatesForFields(targetedCandidates, missingFields)
		targetedValid, targetedRejected := extraction.ValidateCandidates(chunk, targetedCandidates)
		logExtractionDiagnostics(source.ID, chunkIndex, "targeted", targetedCandidates, targetedValid, targetedRejected)
		allCandidates = append(allCandidates, targetedValid...)
		if err := p.Store.SetJobProgress(ctx, jobID, leaseProgress); err != nil {
			return nil, fmt.Errorf("renew extraction job lease: %w", err)
		}
	}
	return deduplicate(allCandidates), nil
}

func missingCanonicalFields(candidates []extraction.Candidate) []string {
	present := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		present[candidate.FieldKey] = struct{}{}
	}
	missing := make([]string, 0)
	for _, definition := range passportservice.CanonicalFieldCatalog() {
		if _, exists := present[definition.Key]; !exists {
			missing = append(missing, definition.Key)
		}
	}
	return missing
}

func candidatesForFields(candidates []extraction.Candidate, fields []string) []extraction.Candidate {
	allowed := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		allowed[field] = struct{}{}
	}
	result := make([]extraction.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if _, exists := allowed[candidate.FieldKey]; exists {
			result = append(result, candidate)
		}
	}
	return result
}

func summarizeRejections(rejected []extraction.RejectedCandidate) map[string]int {
	result := make(map[string]int)
	for _, item := range rejected {
		result[item.Reason]++
	}
	return result
}

func logExtractionDiagnostics(sourceID string, chunkIndex int, pass string, raw, valid []extraction.Candidate, rejected []extraction.RejectedCandidate) {
	slog.Info("Gemini extraction evaluated",
		"source_id", sourceID,
		"chunk", chunkIndex+1,
		"pass", pass,
		"raw_candidates", len(raw),
		"valid_candidates", len(valid),
		"rejected_candidates", len(rejected),
		"rejection_reasons", summarizeRejections(rejected),
	)
}

func (p Processor) fail(ctx context.Context, job pipeline.Job, cause error) error {
	if err := p.Store.FailJob(ctx, job, cause); err != nil {
		return fmt.Errorf("%v; persist job failure: %w", cause, err)
	}
	return cause
}

func chunks(markdown string, limit int) []string {
	if len(markdown) <= limit {
		return []string{markdown}
	}
	result := make([]string, 0, len(markdown)/limit+1)
	for start := 0; start < len(markdown); {
		end := min(start+limit, len(markdown))
		if end < len(markdown) {
			if newline := strings.LastIndex(markdown[start:end], "\n"); newline > limit/2 {
				end = start + newline
			}
		}
		result = append(result, strings.TrimSpace(markdown[start:end]))
		start = end
		for start < len(markdown) && markdown[start] == '\n' {
			start++
		}
	}
	return result
}

func deduplicate(candidates []extraction.Candidate) []extraction.Candidate {
	seen := make(map[string]struct{}, len(candidates))
	result := make([]extraction.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		key := fmt.Sprintf("%s\x00%v\x00%s", candidate.FieldKey, candidate.Value, candidate.Quote)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, candidate)
	}
	return result
}
