# ADR-0010: VBPL Dataset Ingestion Pipeline Design

Status: accepted, 2026-07-19.

## Context

We are integrating the Hugging Face Vietnamese legal document dataset `tmquan/vbpl-vn` (derived from the official `vbpl.vn` database) to populate our PostgreSQL tables (`legal_documents`, `document_versions`, and `document_chunks`). The dataset contains a huge variety of decrees, some of which are irrelevant to business eligibility or lack body texts.

We need a structured, automated pipeline to ingest only relevant, valid documents, parse and chunk them semantically, and generate high-quality vector embeddings.

## Decisions

We decided on the following pipeline design:

1.  **Ingestion Whitelist Filter**: Ingest only documents issued by business-relevant ministries (e.g., Ministry of Finance, Ministry of Planning and Investment, Ministry of Science and Technology, State Bank of Vietnam) AND containing keywords in their title (e.g., "hỗ trợ", "doanh nghiệp", "thuế", "ưu đãi", "tín dụng", "công nghệ").
2.  **Validity Check**: Only ingest documents that are currently in effect. Ingest "Còn hiệu lực" (In effect) and "Hết hiệu lực một phần" (Partially expired). Skip expired ("Hết hiệu lực") documents. Fall back to validity dates if status is empty.
3.  **Missing Bodies API Fetching**: For matching metadata records that lack a body, use the dataset's `api_url` column to fetch the full XML document from the government's official edit gateway (`https://vbpl-bientap-gateway.moj.gov.vn/api/qtdc/public/doc/{document_id}`).
4.  **Article-level Chunking**: Parse the XML tags (such as `<Article>`/`<Điều>`) to split the content into individual Article/Clause chunks. This maintains semantic legal boundaries and ensures precise citations.
5.  **Local E5-Base Embeddings on RTX 3050**: Generate 768-dimensional embeddings using the local PyTorch `sentence-transformers` library with the `intfloat/multilingual-e5-base` model. Force PyTorch execution on the user's local **NVIDIA RTX 3050 GPU** (`device="cuda"`) for high speed and zero cost.

## Consequences

*   **Pros**:
    *   Saves database storage by discarding irrelevant criminal/civil decrees.
    *   Avoids Gemini cloud embedding API costs and rate limits.
    *   Ensures very high quality semantic search results by chunking text at precise article boundaries.
    *   Resolves empty body fields automatically via the moj.gov.vn XML gateway API.
*   **Cons**:
    *   Requires Python, PyTorch (CUDA), and `sentence-transformers` libraries installed in the local environment.
    *   Ingestion speed is bounded by the MoJ gateway API rate limits when fetching missing bodies.
