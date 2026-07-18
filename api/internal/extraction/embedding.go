package extraction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultEmbeddingTimeout = 30 * time.Second
	embeddingDimensions     = 768
	maxConcurrentEmbeddings = 2
)

var embeddingSlots = make(chan struct{}, maxConcurrentEmbeddings)

type ONNXEmbedder struct {
	PythonExecutable string
	ScriptPath       string
	Timeout          time.Duration
}

func (e ONNXEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = defaultEmbeddingTimeout
	}

	embedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	release, err := acquireEmbeddingSlot(embedCtx)
	if err != nil {
		if embedCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("embedding calculation timed out after %s", timeout)
		}
		return nil, fmt.Errorf("embedding calculation canceled: %w", err)
	}
	defer release()

	pyExec := strings.TrimSpace(e.PythonExecutable)
	if pyExec == "" {
		pyExec = "/opt/markitdown/bin/python"
	}

	script := strings.TrimSpace(e.ScriptPath)
	if script == "" {
		script = "/usr/local/bin/calculate_embeddings.py"
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(embedCtx, pyExec, script)

	// Pass the input text via stdin to avoid argument length limits
	cmd.Stdin = strings.NewReader(text)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if embedCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("embedding calculation timed out after %s", timeout)
		}
		// If python script printed a JSON error, try to decode it
		var errData map[string]string
		if decodeErr := json.Unmarshal(stdout.Bytes(), &errData); decodeErr == nil {
			if errMsg, ok := errData["error"]; ok {
				return nil, fmt.Errorf("embedding helper error: %s", errMsg)
			}
		}
		return nil, fmt.Errorf("failed to run embedding helper: %w (stderr: %s)", err, stderr.String())
	}

	var result []float32
	// Handle parsing from stdout JSON array
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		// Check if stdout contains a JSON error instead
		var errData map[string]string
		if decodeErr := json.Unmarshal(stdout.Bytes(), &errData); decodeErr == nil {
			if errMsg, ok := errData["error"]; ok {
				return nil, fmt.Errorf("embedding helper error: %s", errMsg)
			}
		}
		return nil, fmt.Errorf("failed to parse embedding output: %w (stdout: %s)", err, stdout.String())
	}
	if len(result) != embeddingDimensions {
		return nil, fmt.Errorf("embedding helper returned %d dimensions, want %d", len(result), embeddingDimensions)
	}

	return result, nil
}

func acquireEmbeddingSlot(ctx context.Context) (func(), error) {
	select {
	case embeddingSlots <- struct{}{}:
		return func() { <-embeddingSlots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
