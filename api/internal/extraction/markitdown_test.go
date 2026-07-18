package extraction

import (
	"context"
	"os"
	"path/filepath"
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
