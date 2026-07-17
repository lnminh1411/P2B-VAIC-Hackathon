# SQLite Database Repository Pattern

## Context and Decision

To minimize setup friction and avoid dependencies on cloud database credentials during the hackathon, we decided to implement a Repository Pattern in our Python backend, utilizing SQLite as the single database provider. SQLite will store all relational state (tenants, crawler configurations, drafts, and logs). Vector search will be handled using a local Python-based cosine similarity lookup over cached embeddings (using OpenAI or Gemini embeddings) stored directly in SQLite or loaded into memory. This eliminates the need for `pgvector` or Supabase, while keeping the database layer clean and swap-capable in the future.
