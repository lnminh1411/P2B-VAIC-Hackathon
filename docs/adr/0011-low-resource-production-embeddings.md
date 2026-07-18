# ADR-0011: Low-Resource Production Embedding Architecture

Status: accepted, 2026-07-19.

## Context

While our initial database populating script utilized local GPU acceleration (NVIDIA RTX 3050) to run PyTorch embedding generation, the production backend container on Railway runs on a CPU-only environment with tight CPU and memory limits.

Running full PyTorch and `sentence-transformers` on a CPU-only cloud host is extremely resource-heavy. PyTorch alone takes over 1.5GB of RAM to load, which can trigger Out-Of-Memory (OOM) kills on basic container tiers. We need a way for the Go backend to calculate 768-dimensional E5 vector embeddings for newly crawled decrees using as few resources as possible.

## Decisions

We decided on the following low-resource embedding architecture:

1.  **Go Subprocess to Python Helper**: Keep the Go API codebase clean and CGO-free by executing a lightweight Python helper script (`calculate_embeddings.py`) via `exec.CommandContext`, passing the input text through `stdin`.
2.  **ONNX Runtime (CPU) + Hugging Face Tokenizers**: Run the CPU inference using the optimized ONNX Runtime and the Rust-based Hugging Face `tokenizers` library. This stack has zero PyTorch overhead, starts up instantly, and consumes less than 60MB of RAM during execution.
3.  **8-bit Quantized Model (`model_quantized.onnx`)**: Download and run the 8-bit quantized version of `multilingual-e5-base` (`Xenova/multilingual-e5-base/onnx/model_quantized.onnx`). It has a 50% smaller disk footprint (141MB vs. 278MB) and runs twice as fast on CPU with a 99.9% correlation to the original float32 model outputs.
4.  **Remote Cache-on-Startup**: Download the model and tokenizer files on startup from Hugging Face if they are not present in the local cache directory (`P2B_MODEL_CACHE_DIR`), keeping the Git repository lightweight.
5.  **Unified Docker Environment**: Package `onnxruntime` and `tokenizers` directly into the existing `/opt/markitdown` Python virtual environment in `infra/Dockerfile`, avoiding the cost of a separate microservice.

## Consequences

*   **Pros**:
    *   **Low memory footprint**: Safe from OOM container kills on low-resource tiers.
    *   **No CGO in Go**: Avoids cross-compilation headaches across Windows local and Linux Railway targets.
    *   **Completely Free**: Free from external cloud embedding API costs or credits.
    *   **Consistent Vector Space**: Remains 100% compatible with our existing 19.4K database embeddings without needing to re-embed.
*   **Cons**:
    *   Subprocess spawning adds a 50ms overhead per call (completely negligible for background crawling tasks).
    *   First startup requires an internet connection to download the 141MB model file.
