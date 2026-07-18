# P2B MVP

P2B là workspace AI-native giúp doanh nghiệp tạo Company Passport có nguồn dẫn, đối chiếu chính sách bằng rule engine, chuẩn bị checklist, duyệt nội dung và xuất PDF.

## Chạy local

Yêu cầu: Go 1.24+, Node.js 22+, npm.

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

Mở `http://localhost:5173`. `DEV_AUTH=true` bật workspace demo tách biệt theo header `X-Workspace-ID`.

## Luồng demo đã chạy được

1. Khởi tạo doanh nghiệp từ tên, website, nhu cầu và danh sách PDF.
2. Tạo Passport candidates có page, quote, content hash và confidence.
3. Người dùng xác nhận candidates; mỗi lần xác nhận tạo version mới.
4. Xếp hạng policy đã publish và kiểm tra eligibility bằng rule tree deterministic.
5. Tạo enrichment candidates; không tự ghi đè Passport.
6. Sinh checklist từ policy version, khóa hồ sơ khi evidence bắt buộc còn thiếu.
7. Soạn, gửi review, approve snapshot, generate và tải PDF.
8. Xem cảnh báo thay đổi policy/deadline và admin review queue.

## Kiến trúc

- `web/`: React, TypeScript, Vite, TanStack Query, Radix UI, Motion.
- `api/`: Go modular monolith, `chi`, domain rules, HTTP API, worker/scheduler entrypoints.
- `api/migrations/`: Railway PostgreSQL schema và migration runner có checksum/advisory lock.
- `supabase/migrations/`: lịch sử schema Supabase và cấu hình private Storage; business data mới thuộc Railway.
- `infra/`: Railway container có Poppler, Tesseract `vie+eng`, LibreOffice và ClamAV.
- `vercel.json`: Vercel SPA hosting, immutable asset cache, security headers và route fallback.
- `docs/`: threat model và quyết định vận hành.

API contract nằm tại [`api/openapi.yaml`](api/openapi.yaml). Railway migration đầu tiên là [`api/migrations/sql/000001_p2b_core.sql`](api/migrations/sql/000001_p2b_core.sql).

## Demo adapter và production adapter

Repo chạy bằng in-memory workflow adapter để review UX. Production infrastructure dùng Railway PostgreSQL/pgvector, Supabase Auth + private Storage và Gemini; `DEV_AUTH` phải là `false`. Persistence adapter cho workflow, crawler chính thức và DOCX/PDF worker vẫn là launch blockers; policy fixture không phải tư vấn pháp lý hay dữ liệu tài trợ hiện hành.

## Deploy

- Railway: build bằng [`infra/Dockerfile`](infra/Dockerfile) và [`railway.json`](railway.json); pre-deploy tự chạy migration, healthcheck `/health/ready` kiểm DB thật.
- Vercel: import repo, dùng cấu hình [`vercel.json`](vercel.json), đặt bốn biến `VITE_*` theo `.env.example`.
- Secrets chỉ đặt ở Railway/Vercel environment; không đưa Supabase secret key hoặc `DATABASE_URL` xuống browser.
- Supabase production project: `ftmlytoapegpvxbklkyg` (Singapore). Google callback: `https://ftmlytoapegpvxbklkyg.supabase.co/auth/v1/callback`.

Chi tiết ranh giới dữ liệu và rollback: [`docs/ADR-001-RAILWAY-DATABASE.md`](docs/ADR-001-RAILWAY-DATABASE.md).
