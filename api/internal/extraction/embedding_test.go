package extraction

import (
	"context"
	"os"
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
