package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/p2b/p2b/internal/extraction"
	"github.com/p2b/p2b/internal/pipeline"
)

type fakeStore struct {
	sources    []pipeline.SourceRecord
	saved      []extraction.Candidate
	completed  bool
	failedWith error
}

func (f *fakeStore) Sources(context.Context, string, []string) ([]pipeline.SourceRecord, error) {
	return f.sources, nil
}
func (f *fakeStore) StartSource(context.Context, string) error                            { return nil }
func (f *fakeStore) CompleteSource(context.Context, string, string, string, string) error { return nil }
func (f *fakeStore) FailSource(context.Context, string, string) error                     { return nil }
func (f *fakeStore) SaveCandidates(_ context.Context, _, _, _, _ string, candidates []extraction.Candidate) error {
	f.saved = append(f.saved, candidates...)
	return nil
}
func (f *fakeStore) SetJobProgress(context.Context, string, int) error { return nil }
func (f *fakeStore) CompleteJob(context.Context, string) error         { f.completed = true; return nil }
func (f *fakeStore) FailJob(_ context.Context, _ pipeline.Job, cause error) error {
	f.failedWith = cause
	return nil
}

type fakeDownloader struct{ content []byte }

func (f fakeDownloader) Download(context.Context, string, int64) ([]byte, error) {
	return f.content, nil
}

type fakeConverter struct{ markdown string }

func (f fakeConverter) Convert(context.Context, string) (string, error) { return f.markdown, nil }

type fakeExtractor struct{ candidates []extraction.Candidate }

func (f fakeExtractor) Extract(context.Context, string) ([]extraction.Candidate, error) {
	return f.candidates, nil
}

func TestProcessorPersistsOnlyEvidenceBackedCandidates(t *testing.T) {
	store := &fakeStore{sources: []pipeline.SourceRecord{{Source: pipeline.Source{ID: "source-1", Filename: "company.pdf", SizeBytes: 100, ObjectKey: "workspace/sources/source.pdf"}, Status: "UPLOADED"}}}
	processor := Processor{
		Store:      store,
		Downloader: fakeDownloader{content: []byte("%PDF-1.7\ncontent")},
		Converter:  fakeConverter{markdown: "Mã số doanh nghiệp: 0123456789"},
		Extractor: fakeExtractor{candidates: []extraction.Candidate{
			{FieldKey: "tax_code", Value: "0123456789", DataType: "string", Confidence: .98, Quote: "Mã số doanh nghiệp: 0123456789"},
			{FieldKey: "employee_count", Value: float64(25), DataType: "integer", Confidence: .8, Quote: "Số lao động: 25"},
		}},
		Model: "gemini-3.1-flash-lite",
	}
	job := pipeline.Job{ID: "job-1", WorkspaceID: "workspace-1", Attempts: 1, MaxAttempts: 5, Payload: pipeline.JobPayload{SourceIDs: []string{"source-1"}}}

	if err := processor.Process(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if !store.completed || store.failedWith != nil {
		t.Fatalf("completed=%v failed=%v", store.completed, store.failedWith)
	}
	if len(store.saved) != 1 || store.saved[0].FieldKey != "tax_code" {
		t.Fatalf("saved = %#v", store.saved)
	}
}

func TestProcessorRejectsNonPDFStorageObject(t *testing.T) {
	store := &fakeStore{sources: []pipeline.SourceRecord{{Source: pipeline.Source{ID: "source-1", Filename: "company.pdf", SizeBytes: 100, ObjectKey: "workspace/sources/source.pdf"}, Status: "UPLOADED"}}}
	processor := Processor{Store: store, Downloader: fakeDownloader{content: []byte("not a PDF")}, Converter: fakeConverter{}, Extractor: fakeExtractor{}}
	job := pipeline.Job{ID: "job-1", WorkspaceID: "workspace-1", Attempts: 1, MaxAttempts: 5, Payload: pipeline.JobPayload{SourceIDs: []string{"source-1"}}}

	err := processor.Process(context.Background(), job)
	if err == nil || !errors.Is(err, ErrInvalidPDF) {
		t.Fatalf("err = %v, want ErrInvalidPDF", err)
	}
	if store.failedWith == nil {
		t.Fatal("job failure was not persisted")
	}
}
