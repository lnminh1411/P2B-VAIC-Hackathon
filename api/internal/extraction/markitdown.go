package extraction

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	defaultMarkItDownTimeout = 3 * time.Minute
	defaultMaxMarkdownBytes  = 4 << 20
)

var errOutputLimit = errors.New("MarkItDown output exceeds limit")

type MarkItDownConverter struct {
	Executable     string
	OCRExecutable  string
	PDFToImage     string
	OCRLanguages   string
	Timeout        time.Duration
	MaxOutputBytes int
}

func (c MarkItDownConverter) Convert(ctx context.Context, inputPath string) (string, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = defaultMarkItDownTimeout
	}
	limit := c.MaxOutputBytes
	if limit <= 0 {
		limit = defaultMaxMarkdownBytes
	}
	conversionContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	markdown, conversionErr := c.runMarkItDown(conversionContext, inputPath, limit)
	if conversionErr == nil && markdownQualityError(markdown) == nil {
		return strings.TrimSpace(markdown), nil
	}
	qualityErr := markdownQualityError(markdown)
	if conversionErr != nil && qualityErr == nil {
		qualityErr = conversionErr
	}
	if ocrMarkdown, ocrErr := c.convertWithOCR(conversionContext, inputPath, limit); ocrErr == nil {
		return ocrMarkdown, nil
	}
	if conversionErr != nil {
		return "", conversionErr
	}
	return "", qualityErr
}

func (c MarkItDownConverter) runMarkItDown(ctx context.Context, inputPath string, limit int) (string, error) {
	output := &limitedBuffer{remaining: limit}
	var stderr bytes.Buffer
	executable := strings.TrimSpace(c.Executable)
	if executable == "" {
		executable = "markitdown"
	}
	command := exec.CommandContext(ctx, executable, inputPath)
	command.Stdout = output
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if errors.Is(output.err, errOutputLimit) {
			return "", errOutputLimit
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", errors.New("MarkItDown conversion timed out")
		}
		return "", fmt.Errorf("MarkItDown conversion failed: %s", boundedError(stderr.String()))
	}
	return strings.TrimSpace(output.String()), nil
}

func (c MarkItDownConverter) convertWithOCR(ctx context.Context, inputPath string, limit int) (string, error) {
	ocrExecutable := strings.TrimSpace(c.OCRExecutable)
	if ocrExecutable == "" {
		return "", errors.New("OCR fallback is not configured")
	}
	pdfToImage := strings.TrimSpace(c.PDFToImage)
	if pdfToImage == "" {
		pdfToImage = "pdftoppm"
	}
	languages := strings.TrimSpace(c.OCRLanguages)
	if languages == "" {
		languages = "vie+eng"
	}
	directory, err := os.MkdirTemp("", "p2b-ocr-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(directory)
	prefix := filepath.Join(directory, "page")
	if output, runErr := exec.CommandContext(ctx, pdfToImage, "-png", "-r", "200", inputPath, prefix).CombinedOutput(); runErr != nil {
		return "", fmt.Errorf("PDF page rendering failed: %s", boundedError(string(output)))
	}
	pages, err := filepath.Glob(prefix + "-*.png")
	if err != nil || len(pages) == 0 {
		return "", errors.New("OCR fallback produced no pages")
	}
	sort.SliceStable(pages, func(left, right int) bool { return ocrPageNumber(pages[left]) < ocrPageNumber(pages[right]) })
	if len(pages) > 200 {
		return "", errors.New("OCR fallback exceeded 200 pages")
	}
	var markdown strings.Builder
	for index, page := range pages {
		remaining := limit - markdown.Len()
		if remaining <= 0 {
			return "", errOutputLimit
		}
		output := &limitedBuffer{remaining: remaining}
		command := exec.CommandContext(ctx, ocrExecutable, page, "stdout", "-l", languages, "--psm", "6")
		command.Stdout = output
		if err := command.Run(); err != nil {
			return "", fmt.Errorf("OCR failed on page %d: %w", index+1, err)
		}
		markdown.WriteString(fmt.Sprintf("## Page %d\n%s\n", index+1, strings.TrimSpace(output.String())))
	}
	result := strings.TrimSpace(markdown.String())
	if err := markdownQualityError(result); err != nil {
		return "", err
	}
	return result, nil
}

func ocrPageNumber(path string) int {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	_, number, found := strings.Cut(name, "-")
	if !found {
		return 0
	}
	value, err := strconv.Atoi(number)
	if err != nil {
		return 0
	}
	return value
}

func markdownQualityError(markdown string) error {
	fields := strings.Fields(markdown)
	meaningful := 0
	for _, character := range markdown {
		if unicode.IsLetter(character) || unicode.IsNumber(character) {
			meaningful++
		}
	}
	if len(fields) < 2 || meaningful < 12 {
		return errors.New("extracted text quality is too low; PDF may be scanned or damaged")
	}
	return nil
}

type limitedBuffer struct {
	buffer    bytes.Buffer
	remaining int
	err       error
}

func (b *limitedBuffer) Write(value []byte) (int, error) {
	if len(value) > b.remaining {
		b.err = errOutputLimit
		return 0, b.err
	}
	b.remaining -= len(value)
	return b.buffer.Write(value)
}

func (b *limitedBuffer) String() string { return b.buffer.String() }

func boundedError(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown converter error"
	}
	const limit = 500
	if len(value) > limit {
		return value[:limit]
	}
	return value
}
