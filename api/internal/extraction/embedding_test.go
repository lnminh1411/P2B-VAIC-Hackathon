package extraction

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestONNXEmbedderRunsEmbedding(t *testing.T) {
	// Find local python executable
	pythonExec := "python"
	if _, err := os.Stat("/opt/markitdown/bin/python"); err == nil {
		pythonExec = "/opt/markitdown/bin/python"
	}

	// Get absolute path to calculate_embeddings.py relative to the api directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// E.g. we are in api/internal/extraction/
	scriptPath := filepath.Join(cwd, "..", "..", "..", "scripts", "calculate_embeddings.py")

	// If the script doesn't exist, we are probably running go tests from elsewhere; skip
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skip("calculate_embeddings.py script not found, skipping real test")
	}

	// Check if python and all required dependencies are available
	depCheck := exec.Command(pythonExec, "-c", "import onnxruntime, tokenizers, numpy")
	if err := depCheck.Run(); err != nil {
		t.Skip("Python dependencies (onnxruntime, tokenizers, numpy) are not installed, skipping real embedding test")
	}

	embedder := ONNXEmbedder{
		PythonExecutable: pythonExec,
		ScriptPath:       scriptPath,
		Timeout:          30 * time.Second,
	}

	ctx := context.Background()
	vector, err := embedder.Embed(ctx, "passage: test Vietnamese embedding")
	if err != nil {
		t.Fatalf("Failed to generate embedding: %v", err)
	}

	if len(vector) != 768 {
		t.Errorf("Expected 768-dimensional vector, got %d", len(vector))
	}
}

func TestONNXEmbedderHandlesErrorCleanly(t *testing.T) {
	embedder := ONNXEmbedder{
		PythonExecutable: "python",
		ScriptPath:       "non_existent_script.py",
		Timeout:          5 * time.Second,
	}

	_, err := embedder.Embed(context.Background(), "test")
	if err == nil {
		t.Error("Expected error when running non-existent script, got nil")
	}
}

func TestProductionImagePreloadsONNXModel(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dockerfile, err := os.ReadFile(filepath.Join(cwd, "..", "..", "..", "infra", "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range [][]byte{
		[]byte("P2B_MODEL_CACHE_DIR=/opt/p2b-embedding-model"),
		[]byte("query: production model preload"),
	} {
		if !bytes.Contains(dockerfile, required) {
			t.Fatalf("production Dockerfile does not preload ONNX model; missing %q", required)
		}
	}
}

func TestEmbeddingSlotHonorsContextCancellation(t *testing.T) {
	for range cap(embeddingSlots) {
		embeddingSlots <- struct{}{}
	}
	defer func() {
		for range cap(embeddingSlots) {
			<-embeddingSlots
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := acquireEmbeddingSlot(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("acquire error = %v, want context canceled", err)
	}
}
