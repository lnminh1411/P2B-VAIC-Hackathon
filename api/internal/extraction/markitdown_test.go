package extraction

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMarkItDownConverterUsesArgumentWithoutShellExpansion(t *testing.T) {
	directory := t.TempDir()
	program := filepath.Join(directory, "fake-markitdown")
	script := "#!/bin/sh\nprintf '# Extracted\\n%s' \"$1\"\n"
	if err := os.WriteFile(program, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(directory, "company;touch SHOULD_NOT_EXIST.pdf")
	if err := os.WriteFile(input, []byte("%PDF-1.7"), 0o600); err != nil {
		t.Fatal(err)
	}

	converter := MarkItDownConverter{Executable: program, Timeout: 5 * time.Second, MaxOutputBytes: 1024}
	markdown, err := converter.Convert(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if markdown == "" {
		t.Fatal("markdown is empty")
	}
	if _, err := os.Stat(filepath.Join(directory, "SHOULD_NOT_EXIST.pdf")); !os.IsNotExist(err) {
		t.Fatal("filename was interpreted by a shell")
	}
}

func TestMarkItDownConverterRejectsOversizedOutput(t *testing.T) {
	directory := t.TempDir()
	program := filepath.Join(directory, "fake-markitdown")
	if err := os.WriteFile(program, []byte("#!/bin/sh\nprintf '123456789'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(directory, "company.pdf")
	if err := os.WriteFile(input, []byte("%PDF-1.7"), 0o600); err != nil {
		t.Fatal(err)
	}

	converter := MarkItDownConverter{Executable: program, Timeout: 5 * time.Second, MaxOutputBytes: 8}
	if _, err := converter.Convert(context.Background(), input); err == nil {
		t.Fatal("expected oversized output error")
	}
}

func TestMarkItDownConverterRejectsLowQualityTextInsteadOfPublishingEmptyFacts(t *testing.T) {
	directory := t.TempDir()
	program := filepath.Join(directory, "fake-markitdown")
	if err := os.WriteFile(program, []byte("#!/bin/sh\nprintf '[Image]'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(directory, "scanned.pdf")
	if err := os.WriteFile(input, []byte("%PDF-1.7"), 0o600); err != nil {
		t.Fatal(err)
	}

	converter := MarkItDownConverter{Executable: program, Timeout: 5 * time.Second, MaxOutputBytes: 1024}
	if _, err := converter.Convert(context.Background(), input); err == nil {
		t.Fatal("expected low quality extraction error")
	}
}

func TestMarkItDownConverterSupplementsTableHeavyMarkdownWithLayoutText(t *testing.T) {
	directory := t.TempDir()
	markitdown := filepath.Join(directory, "fake-markitdown")
	markitdownScript := "#!/bin/sh\nprintf '| A | B | C | D | E | F | G | H | I | J | K | L |\\n| Vốn điều lệ | | | | | | | | | | | |'\n"
	if err := os.WriteFile(markitdown, []byte(markitdownScript), 0o700); err != nil {
		t.Fatal(err)
	}
	pdfText := filepath.Join(directory, "fake-pdftotext")
	if err := os.WriteFile(pdfText, []byte("#!/bin/sh\nprintf 'Vốn điều lệ hiện tại: 5.100,6 tỷ VNĐ'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(directory, "company.pdf")
	if err := os.WriteFile(input, []byte("%PDF-1.7"), 0o600); err != nil {
		t.Fatal(err)
	}

	converter := MarkItDownConverter{Executable: markitdown, PDFTextExecutable: pdfText, Timeout: 5 * time.Second, MaxOutputBytes: 4096}
	markdown, err := converter.Convert(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(markdown, "## Layout-preserving PDF text") || !strings.Contains(markdown, "5.100,6 tỷ VNĐ") {
		t.Fatalf("markdown = %q", markdown)
	}
}
