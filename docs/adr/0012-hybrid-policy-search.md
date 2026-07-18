# ADR-0012: Hybrid Policy and Legal-Document Search

Status: accepted, 2026-07-19.

## Context

Production contains 682 legal documents and 19,488 article-level chunks with 768-dimensional `multilingual-e5-base` embeddings. Matching previously queried only reviewed rows in `policy_versions`, so a populated `document_chunks` corpus could still produce zero results.

Query inference also depended on downloading the quantized ONNX model during the first request. A real cold-cache check exceeded the API's 30-second embedding timeout, while cached CPU inference produced a finite, L2-normalized 768-dimensional vector in under one second.

## Decision

1. Build an E5 query from support needs and confirmed Passport facts, using the required `query:` prefix. Existing corpus chunks retain their `passage:` prefix.
2. Retrieve current chunks with both PostgreSQL full-text rank and pgvector cosine distance, then combine the two ranked lists using Reciprocal Rank Fusion.
3. Deduplicate the best chunk per legal document and merge these results with deterministic eligibility results from reviewed `policy_versions` rules.
4. Keep retrieved, unstructured documents at `MISSING_INFO`; semantic relevance is not treated as proof that eligibility rules are met.
5. Fall back to full-text plus rule matching if query inference is temporarily unavailable.
6. Download and verify the quantized ONNX model during Docker image build. Production requests use the model cache already embedded in the image.
7. Limit the API process to two concurrent ONNX subprocesses; queued inference respects request cancellation and the embedding timeout.

## Consequences

- The 682-document corpus is searchable even before every document has been converted into a reviewed structured policy.
- Existing rule-engine decisions remain deterministic and distinguishable from semantic retrieval.
- Production image size grows by the quantized model and tokenizer size, but request latency and availability no longer depend on a first-request network download.
- Docker builds require access to the pinned Hugging Face model URLs and fail early if the model cannot load.
- This ADR supersedes ADR-0011's runtime/on-demand cache decision; its ONNX CPU, quantization, and CGO-free subprocess decisions remain active.
