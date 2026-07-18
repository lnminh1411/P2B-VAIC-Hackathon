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

## Luồng đã chạy được

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
- `api/migrations/`: PostgreSQL/Supabase schema, RLS-ready tenant keys và `pgvector`.
- `infra/`: Railway container có Poppler, Tesseract `vie+eng`, LibreOffice và ClamAV.
- `firebase.json`: SPA hosting, cache headers và route fallback.
- `docs/`: threat model và quyết định vận hành.

API contract nằm tại [`api/openapi.yaml`](api/openapi.yaml). Migration đầu tiên là [`api/migrations/000001_p2b_core.up.sql`](api/migrations/000001_p2b_core.up.sql).

## Demo adapter và production adapter

Repo chạy ngay bằng in-memory demo service để review toàn bộ UX mà không cần secrets. Trước production phải cấu hình Supabase Auth/PostgreSQL/Storage, Gemini và nguồn crawler chính thức; `DEV_AUTH` phải là `false`. Policy trong demo là fixtures minh họa, không phải tư vấn pháp lý hay dữ liệu tài trợ hiện hành.

## Deploy

- Railway: build bằng [`infra/Dockerfile`](infra/Dockerfile), healthcheck `/health/ready`; tách command thành `api`, `worker`, `crawler-scheduler`, `migrate`.
- Firebase: chạy `make build`, sau đó deploy `web/dist` với cấu hình [`firebase.json`](firebase.json).
- Secrets chỉ đặt ở Railway/Firebase environment; không đưa service-role key xuống browser.

