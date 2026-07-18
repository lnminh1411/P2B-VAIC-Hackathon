package worker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/p2b/p2b/internal/extraction"
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
		if err = p.processSource(ctx, job.WorkspaceID, source); err != nil {
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

func (p Processor) processSource(ctx context.Context, workspaceID string, source pipeline.SourceRecord) error {
	content, err := p.Downloader.Download(ctx, source.ObjectKey, source.SizeBytes)
	if err != nil {
		return fmt.Errorf("download %s: %w", source.Filename, err)
	}
	prefixLength := min(len(content), 1024)
	if prefixLength < 5 || !bytes.Contains(content[:prefixLength], []byte("%PDF-")) {
		return ErrInvalidPDF
	}
	hash := sha256.Sum256(content)
	contentHash := hex.EncodeToString(hash[:])
	file, err := os.CreateTemp("", "p2b-source-*.pdf")
	if err != nil {
		return fmt.Errorf("create isolated source file: %w", err)
	}
	path := file.Name()
	defer os.Remove(path)
	if err = file.Chmod(0o600); err == nil {
		_, err = file.Write(content)
	}
	closeErr := file.Close()
	if err != nil {
		return fmt.Errorf("write isolated source file: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close isolated source file: %w", closeErr)
	}
	markdown, err := p.Converter.Convert(ctx, path)
	if err != nil {
		return fmt.Errorf("convert %s: %w", source.Filename, err)
	}
	allCandidates := make([]extraction.Candidate, 0)
	for _, chunk := range chunks(markdown, maxGeminiChunkBytes) {
		candidates, extractErr := p.Extractor.Extract(ctx, chunk)
		if extractErr != nil {
			return fmt.Errorf("extract %s with Gemini: %w", source.Filename, extractErr)
		}
		valid, _ := extraction.ValidateCandidates(chunk, candidates)
		allCandidates = append(allCandidates, valid...)
	}
	allCandidates = deduplicate(allCandidates)
	if err = p.Store.SaveCandidates(ctx, workspaceID, source.ID, source.Filename, contentHash, allCandidates); err != nil {
		return err
	}
	if err = p.Store.CompleteSource(ctx, source.ID, contentHash, markdown, p.Model); err != nil {
		return err
	}
	return nil
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
