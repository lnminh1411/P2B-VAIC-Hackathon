# P2B - Workspace AI-native Hỗ trợ Đăng ký Chương trình Tài trợ Doanh nghiệp
**P2B** là một không gian làm việc số (workspace) tích hợp trí tuệ nhân tạo (AI-native) toàn diện, giúp doanh nghiệp tự động hóa quy trình phân tích tài liệu pháp lý, đối chiếu tiêu chuẩn tài trợ và chuẩn bị hồ sơ ứng tuyển chất lượng cao.

*(Catchy Description)*: Lập hồ sơ doanh nghiệp có trích dẫn nguồn, đối chiếu tự động với hàng trăm chính sách nhà nước và xuất đơn ứng tuyển chuẩn chỉ chỉ trong vài phút.

---

## 🔗 Demo Trực tuyến / Live Demo
Trải nghiệm phiên bản thử nghiệm trực tuyến tại: [https://p2b-zeta.vercel.app/](https://p2b-zeta.vercel.app/)

---

## Mục lục
1. [Tính năng cốt lõi & Quy trình Người dùng](#tính-năng-cốt-lõi--quy-trình-người-dùng)
2. [Tổng quan Công nghệ (Tech Stack)](#tổng-quan-công-nghệ-tech-stack)
3. [Hướng dẫn Thiết lập Local (Full-Stack)](#hướng-dẫn-thiết-lập-local-full-stack)
4. [Cấu hình Biến môi trường](#cấu-hình-biến-môi-trường)

---

## Tính năng cốt lõi & Quy trình Người dùng

### 1. Khởi tạo Workspace & Lập Hồ sơ Doanh nghiệp (Company Passport)
*   **Tải tài liệu:** Doanh nghiệp tải lên các văn bản chứng minh pháp lý (Giấy phép kinh doanh, Báo cáo tài chính, Chứng nhận đầu tư...) lên vùng lưu trữ bảo mật ([Supabase Private Storage](https://supabase.com/docs/guides/storage/shared/private-buckets)).
*   **Trích xuất cấu trúc bằng AI:** Hệ thống sử dụng [Microsoft MarkItDown](https://github.com/microsoft/markitdown) để chuyển đổi PDF/Docx thành Markdown, sau đó gọi mô hình `gemini-3.1-flash-lite` để trích xuất các thông tin định dạng sẵn.
*   **Đảm bảo không ảo giác (Zero Hallucination):** Mỗi thông tin trích xuất bắt buộc phải có câu trích dẫn (`quote`) khớp chính xác từng ký tự trong văn bản gốc mới được đưa vào trạng thái Chờ Duyệt (`NEEDS_REVIEW`).
*   **Duyệt và Cập nhật:** Người dùng đối chiếu nguồn, xác nhận thông tin đúng để tạo phiên bản Hồ sơ Doanh nghiệp chính thức mới.

### 2. Động cơ Luật Điều kiện Xác định (Deterministic Eligibility Rule Engine)
*   **Không phụ thuộc vào AI (AI-free Evaluation):** Động cơ sử dụng mã nguồn Go để tính toán chính xác điều kiện của doanh nghiệp so với tiêu chuẩn chính sách (gồm các so sánh chuỗi, ngày tháng, số học phức tạp như `EQ`, `IN`, `CONTAINS`, `GT`, `GTE`, `LT`, `LTE`, `EXISTS`, `DATE_BEFORE`, `DATE_AFTER`).
*   **Phân loại trạng thái:** Kết quả trả về gồm `MET` (Đạt), `NOT_MET` (Không đạt) hoặc `MISSING_INFO` (Thiếu thông tin chứng minh).
*   **Xếp hạng tự động:** Gợi ý các chính sách hỗ trợ/tài trợ có tỷ lệ khớp cao nhất cho doanh nghiệp.

### 3. Danh sách kiểm tra & Chuẩn bị Hồ sơ Ứng tuyển (Evidence-Gated Checklist)
*   **Sinh Checklist tự động:** Dựa trên các điều kiện luật của chính sách, hệ thống tạo checklist các văn bản bắt buộc cần nộp.
*   **Khóa hồ sơ theo bằng chứng:** Chỉ khi toàn bộ các trường thông tin quan trọng được xác minh nguồn dẫn, đơn ứng tuyển mới cho phép hoàn tất và phê duyệt.
*   **Xuất đơn PDF:** Tự động tạo và điền các mẫu đơn ứng tuyển dạng PDF/Docx chứa đầy đủ chữ ký số và bằng chứng đi kèm để nộp lên cơ quan quản lý.

### 4. Kênh Quản trị & Cảnh báo Chính sách (Admin Queue & Policy Alerts)
*   **Phát hành chính sách:** Admin có thể cập nhật các phiên bản quy định, chương trình tài trợ mới nhất.
*   **Cảnh báo thông minh:** Hệ thống tự động gửi thông báo đến các doanh nghiệp khi có sự thay đổi chính sách ảnh hưởng trực tiếp đến trạng thái điều kiện của họ.

---

## Tổng quan Công nghệ (Tech Stack)

*   **Backend**: Go 1.26+, [go-chi/chi](https://github.com/go-chi/chi) router, `pgx/v5` PostgreSQL client, embedded SQL migrations.
*   **Frontend**: React 19, TypeScript, Vite, [TanStack Query v5](https://tanstack.com/query/latest), [Motion](https://motion.dev/) (animations), Radix UI, Vanilla CSS (biến CSS tùy chỉnh).
*   **Dịch vụ ngoài & Lưu trữ**: Supabase (Xác thực, Storage riêng tư), Railway PostgreSQL 17 (pgvector 0.8.2).
*   **Trích xuất & AI**: Microsoft MarkItDown CLI, Google Gemini API (`gemini-3.1-flash-lite`).
*   **Môi trường chạy sản phẩm (Production)**: Docker (chứa Poppler, Tesseract `vie+eng`, LibreOffice, ClamAV) chạy trên Railway; SPA hosting chạy trên Vercel.

---

## Nguồn dữ liệu
Dự án sử dụng cơ sở dữ liệu văn bản pháp quy Việt Nam từ các nguồn chính thức và bộ dữ liệu cộng đồng:
*   [Cổng thông tin điện tử Pháp điển / VBPL](https://vbpl.vn)
*   Bộ dữ liệu văn bản pháp luật VBPL trên Hugging Face: [tmquan/vbpl-vn Dataset](https://huggingface.co/datasets/tmquan/vbpl-vn)

---

## Hướng dẫn Thiết lập Local (Full-Stack)

### 1. Yêu cầu hệ thống
*   **Go** 1.24+ (hoặc 1.26)
*   **Node.js** 22+ & **npm**
*   **Python** 3.10+ (cần thiết cho local extraction worker chạy thư viện `markitdown`)
*   Cài đặt công cụ CLI `markitdown`:
    ```bash
    pip install markitdown[pdf]==0.1.6
    ```

### 2. Khởi động nhanh chế độ Phát triển (Dev Mode)
Trong chế độ phát triển mặc định (`DEV_AUTH=true`), hệ thống sử dụng bộ lưu trữ bộ nhớ (in-memory) giả lập. Bạn **không cần** kết nối PostgreSQL hay Supabase để chạy ứng dụng!

1.  **Sao chép cấu hình môi trường:**
    ```bash
    cp .env.example .env
    ```
2.  **Khởi động Backend (Go API):**
    ```bash
    make api
    # Hoặc: cd api && DEV_AUTH=true go run ./cmd/api
    ```
3.  **Khởi động Frontend (React/Vite):**
    Mở một terminal mới và chạy:
    ```bash
    make web
    # Hoặc: cd web && npm install && npm run dev
    ```
4.  Truy cập ứng dụng tại địa chỉ: `http://localhost:5173`. Header mặc định `X-Workspace-ID` được dùng để phân chia workspace độc lập.

### 3. Chạy Kiểm thử & Định dạng
*   **Chạy toàn bộ unit test:**
    ```bash
    make test
    ```
*   **Kiểm tra lỗi cú pháp và lint:**
    ```bash
    make lint
    ```
*   **Biên dịch dự án:**
    ```bash
    make build
    ```

### 4. Kết nối Cơ sở dữ liệu thật (Chế độ Production Local)
Nếu muốn chạy với PostgreSQL cục bộ hoặc môi trường Supabase:
1. Đảm bảo đặt `DEV_AUTH=false` và `VITE_DEV_AUTH=false` trong file `.env`.
2. Điền đầy đủ thông tin kết nối `DATABASE_URL` (PostgreSQL), các khóa `SUPABASE_*` và `GEMINI_API_KEY`.
3. Chạy lệnh migrate để tạo bảng:
   ```bash
   make migrate
   ```

---

## Cấu hình Biến môi trường
Chi tiết các biến môi trường cấu hình tại file `.env`:
*   `DEV_AUTH`: Thiết lập `true` để bỏ qua xác thực Supabase JWT, hữu dụng cho phát triển local.
*   `DATABASE_URL`: Đường dẫn kết nối PostgreSQL của Railway (chỉ dùng khi `DEV_AUTH=false`).
*   `SUPABASE_URL` / `SUPABASE_SECRET_KEY`: Dùng cấu hình xác thực và vùng chứa Storage.
*   `GEMINI_API_KEY`: API Key kết nối dịch vụ Google Gemini.

---
---

# P2B - AI-Native Grant Eligibility & Application Workspace
**P2B** is an AI-native workspace designed to help companies streamline the grant application process by automating document extraction, checking eligibility criteria through a deterministic rule engine, and preparing application checklists.

*(Catchy Description)*: Instantly build verified Company Passports, automatically evaluate matching grants, and generate audit-ready applications with clear evidence provenance in minutes.

---

## 🔗 Live Demo
Experience the live application at: [https://p2b-zeta.vercel.app/](https://p2b-zeta.vercel.app/)

---

## Table of Contents
1. [Core Features & User Flows](#core-features--user-flows)
2. [Tech Stack Overview](#tech-stack-overview)
3. [Full-Stack Local Development Setup](#full-stack-local-development-setup)
4. [Environment Configuration](#environment-configuration)

---

## Core Features & User Flows

### 1. Onboarding & Company Passport Generation
*   **Document Upload:** Users securely upload business evidence (Business Licenses, Financial Statements, Investment Certificates) straight to Supabase Private Storage.
*   **AI-Structured Extraction:** The pipeline uses Microsoft MarkItDown to parse the documents into clean Markdown, then calls Gemini 3.1 Flash-Lite to extract canonical fields.
*   **Zero Hallucinations:** Every candidate field must map to an exact quote found in the source text to prevent AI hallucinations. Unmatched/unverified facts are flagged as `NEEDS_REVIEW`.
*   **Verification Workflow:** Users review the facts, resolve conflicts, and promote them to create a versioned, immutable Company Passport.

### 2. Deterministic Eligibility Rule Engine
*   **AI-Free Matching:** Go-based engine evaluates fields against policies using precise comparison operators (like `EQ`, `IN`, `CONTAINS`, `GT`, `GTE`, `LT`, `LTE`, `EXISTS`, `DATE_BEFORE`, `DATE_AFTER`).
*   **Evaluation Statuses:** Tracks status as `MET`, `NOT_MET`, or `MISSING_INFO` (if a field is unconfirmed).
*   **Smart Ranking:** Grants are sorted dynamically based on matching criteria scores.

### 3. Evidence-Gated Checklists & Applications
*   **Automated Checklist:** Generates document and info check-items mapped to policy requirements.
*   **Gated Actions:** Ensures application submission is locked until necessary evidence has been reviewed and confirmed.
*   **PDF Package Export:** Compiles confirmed fields and checklists to export a structured PDF ready for submission.

### 4. Admin Queue & Policy Change Alerts
*   **Policy Publishing:** Admins manage active policy versions in the database.
*   **Real-time Alerts:** Automatically monitors changes to active policies and notifies tenants when a policy update changes their eligibility status.

---

## Tech Stack Overview

*   **Backend**: Go 1.26+, [go-chi/chi](https://github.com/go-chi/chi) router, `pgx/v5` PostgreSQL client, embedded SQL migrations.
*   **Frontend**: React 19, TypeScript, Vite, [TanStack Query v5](https://tanstack.com/query/latest), [Motion](https://motion.dev/) (animations), Radix UI, Vanilla CSS (using CSS variables).
*   **External Services & Databases**: Supabase (Auth, Private Storage), Railway PostgreSQL 17 (pgvector 0.8.2).
*   **Extraction & AI**: Microsoft MarkItDown CLI, Google Gemini API (`gemini-3.1-flash-lite`).
*   **Production Hosting**: Docker-based containers (containing Poppler, Tesseract `vie+eng`, LibreOffice, ClamAV) deployed to Railway; Vercel for SPA web hosting.

---

## Data Sources
The system utilizes Vietnamese official legal document datasets sourced from:
*   [Vietnam Official Legal Documents Portal / VBPL](https://vbpl.vn)
*   Hugging Face Vietnamese VBPL Dataset: [tmquan/vbpl-vn Dataset](https://huggingface.co/datasets/tmquan/vbpl-vn)

---

## Full-Stack Local Development Setup

### 1. Prerequisites
*   **Go** 1.24+ (or 1.26)
*   **Node.js** 22+ & **npm**
*   **Python** 3.10+ (required for local extraction worker running `markitdown`)
*   Install the `markitdown` CLI tool:
    ```bash
    pip install markitdown[pdf]==0.1.6
    ```

### 2. Fast Dev Mode Start
By default, the workspace is configured to use development mode (`DEV_AUTH=true`) which runs with in-memory database adapters. You **do not need** a live PostgreSQL database or Supabase setup to run and play with the app locally!

1.  **Clone the environment file:**
    ```bash
    cp .env.example .env
    ```
2.  **Start the Go API Server:**
    ```bash
    make api
    # Or: cd api && DEV_AUTH=true go run ./cmd/api
    ```
3.  **Start the React Web Client:**
    In a new terminal window, run:
    ```bash
    make web
    # Or: cd web && npm install && npm run dev
    ```
4.  Open `http://localhost:5173` in your browser. The default `X-Workspace-ID` header is used to isolate workspaces.

### 3. Tests & Linters
*   **Run all tests:**
    ```bash
    make test
    ```
*   **Run syntax and lint checks:**
    ```bash
    make lint
    ```
*   **Build the full stack:**
    ```bash
    make build
    ```

### 4. Connecting a Database (Production-ready Local Run)
If you want to run with local PostgreSQL or live Supabase services:
1. Set `DEV_AUTH=false` and `VITE_DEV_AUTH=false` in your `.env` file.
2. Supply real credentials for `DATABASE_URL` (PostgreSQL), `SUPABASE_*` credentials, and `GEMINI_API_KEY`.
3. Apply migrations to initialize the database schema:
   ```bash
   make migrate
   ```

---

## Environment Configuration
Key environment configurations inside `.env`:
*   `DEV_AUTH`: Set to `true` to bypass Supabase JWT validation, ideal for quick local development.
*   `DATABASE_URL`: Connection string for Railway PostgreSQL instance (used when `DEV_AUTH=false`).
*   `SUPABASE_URL` / `SUPABASE_SECRET_KEY`: Used for authentication and bucket signing operations.
*   `GEMINI_API_KEY`: API Key for Google Gemini services.
