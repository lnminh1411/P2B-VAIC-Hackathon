package extraction

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultMarkItDownTimeout = 3 * time.Minute
	defaultMaxMarkdownBytes  = 4 << 20
)

var errOutputLimit = errors.New("MarkItDown output exceeds limit")

type MarkItDownConverter struct {
	Executable     string
	Timeout        time.Duration
	MaxOutputBytes int
}

func (c MarkItDownConverter) Convert(ctx context.Context, inputPath string) (string, error) {
	executable := strings.TrimSpace(c.Executable)
	if executable == "" {
		executable = "markitdown"
	}
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

	output := &limitedBuffer{remaining: limit}
	var stderr bytes.Buffer
	command := exec.CommandContext(conversionContext, executable, inputPath)
	command.Stdout = output
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if errors.Is(output.err, errOutputLimit) {
			return "", errOutputLimit
		}
		if errors.Is(conversionContext.Err(), context.DeadlineExceeded) {
			return "", errors.New("MarkItDown conversion timed out")
		}
		return "", fmt.Errorf("MarkItDown conversion failed: %s", boundedError(stderr.String()))
	}
	markdown := strings.TrimSpace(output.String())
	if markdown == "" {
		return "", errors.New("MarkItDown produced empty output; PDF may be scanned or damaged")
	}
	return markdown, nil
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
