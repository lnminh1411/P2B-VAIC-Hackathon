# P2B MVP

P2B là workspace AI-native giúp doanh nghiệp tạo Company Passport có nguồn dẫn, đối chiếu chính sách bằng rule engine, chuẩn bị checklist, duyệt nội dung và xuất PDF.

## Chạy local

Yêu cầu: Go 1.24+, Node.js 22+, npm. Extraction worker cần Python 3.10+ và `markitdown[pdf]==0.1.6`.

```bash
cp .env.example .env
make test
make build
```

Chạy API và web ở hai terminal:

```bash
make api
make web
```

Mở `http://localhost:5173`. `DEV_AUTH=true` bật workspace development tách biệt theo header `X-Workspace-ID`.

## Trạng thái runtime hiện tại

1. Supabase Auth xác thực Google; Railway PostgreSQL bootstrap workspace.
2. Frontend upload PDF trực tiếp vào private Supabase Storage qua signed URL.
3. API ghi nguồn upload vào Railway PostgreSQL, xác nhận upload hoàn tất và enqueue job bền vững.
4. Worker tải PDF bằng server credential, kiểm kích thước/PDF signature/SHA-256, dùng MarkItDown chuyển PDF thành Markdown rồi gọi `gemini-3.1-flash-lite` với structured output.
5. Chỉ candidate có quote khớp nguồn mới được lưu `NEEDS_REVIEW`; người dùng xác nhận mới tạo Passport version mới.
6. Job queue dùng lease, `FOR UPDATE SKIP LOCKED`, retry/backoff và dead-letter. Worker xử lý tuần tự để giới hạn RAM.
7. Production policy corpus, website crawler, enrichment candidates và alerts mặc định trống cho đến khi adapter thật được kết nối.

## Kiến trúc

- `web/`: React, TypeScript, Vite, TanStack Query, Radix UI, Motion.
- `api/`: Go modular monolith, `chi`, domain rules, HTTP API, worker/scheduler entrypoints.
- `api/migrations/`: Railway PostgreSQL schema và migration runner có checksum/advisory lock.
- `supabase/migrations/`: lịch sử schema Supabase và cấu hình private Storage; business data mới thuộc Railway.
- `infra/`: Railway container có Poppler, Tesseract `vie+eng`, LibreOffice và ClamAV.
- `vercel.json`: Vercel SPA hosting, immutable asset cache, security headers và route fallback.
- `docs/`: threat model và quyết định vận hành.

API contract nằm tại [`api/openapi.yaml`](api/openapi.yaml). Railway migration đầu tiên là [`api/migrations/sql/000001_p2b_core.sql`](api/migrations/sql/000001_p2b_core.sql).

## Adapter còn thiếu

Company profiling, source metadata, jobs, Markdown và field candidates đã dùng Railway PostgreSQL. Policy matching, enrichment, checklist và application runtime cũ vẫn nằm trong memory cho đến khi repository tương ứng được chuyển đổi. Chưa có website collector, policy repository/crawler thật hoặc DOCX/LibreOffice export orchestration. `DEV_AUTH` phải là `false` trên production.

## Deploy

- Railway: build bằng [`infra/Dockerfile`](infra/Dockerfile) và [`railway.json`](railway.json); pre-deploy tự chạy migration. Cấu hình service riêng: API chạy `/usr/local/bin/p2b-service` với healthcheck `/health/ready`; worker chạy `/usr/local/bin/p2b-worker` không mở HTTP port.
- Vercel: import repo, dùng cấu hình [`vercel.json`](vercel.json), đặt bốn biến `VITE_*` theo `.env.example`.
- Secrets chỉ đặt ở Railway/Vercel environment; không đưa Supabase secret key hoặc `DATABASE_URL` xuống browser.
- Extraction worker bắt buộc có `GEMINI_API_KEY`; model mặc định `GEMINI_MODEL=gemini-3.1-flash-lite`.
- Supabase production project: `ftmlytoapegpvxbklkyg` (Singapore). Google callback: `https://ftmlytoapegpvxbklkyg.supabase.co/auth/v1/callback`.

Chi tiết ranh giới dữ liệu và rollback: [`docs/ADR-001-RAILWAY-DATABASE.md`](docs/ADR-001-RAILWAY-DATABASE.md).
