# P2B (Policy-to-Business) — Nền tảng Hỗ trợ Tiếp cận Chính sách Ưu đãi cho Doanh nghiệp

P2B là nền tảng AI-native giúp các doanh nghiệp nhỏ và vừa (SMEs) và các startup tại Việt Nam tự động hóa quy trình tìm kiếm, đánh giá điều kiện và chuẩn bị hồ sơ xin thụ hưởng các chính sách ưu đãi, tài trợ và chương trình hỗ trợ của Chính phủ.

---

## 🏗️ Kiến trúc Hệ thống (4 Vertical Slices)

1. **Hồ sơ Doanh nghiệp (Company Passport & Provenance)**:
   - Tổng hợp dữ liệu từ hồ sơ đăng ký kinh doanh, pitch deck, website doanh nghiệp.
   - Theo dõi nguồn gốc cấp trường thông tin (Field-level Provenance) với trạng thái minh bạch (`EXTRACTED`, `USER_CONFIRMED`, `CONFLICTED`, `MISSING`).
   - Xử lý mâu thuẫn dữ liệu từ các nguồn (Data Conflicts) một cách trực quan.

2. **Tìm kiếm Kết hợp (Hybrid RAG Retrieval)**:
   - Định dạng văn bản pháp luật thành các khối chunk có gán nhãn ngữ cảnh.
   - Kết hợp tìm kiếm lexical (FTS5/BM25) và tìm kiếm ngữ nghĩa Vector Cosine Similarity (mô hình `multilingual-e5-base` chạy offline).
   - Xếp hạng nâng cao bằng cơ chế Score Fusion (0.4 BM25 + 0.4 Vector + 0.2 Metadata).

3. **Công cụ Thẩm định Điều kiện (Deterministic Eligibility Engine)**:
   - Thẩm định điều kiện theo cơ chế Rule Engine có cấu trúc, hoàn toàn tách biệt khỏi các ảo tưởng (hallucination) của LLM.
   - Hỗ trợ các toán tử số học, ngày tháng và kiểm tra logic phức tạp (`ALL`, `ANY`).
   - Trả kết quả theo trạng thái logic 3 trị (`MET`, `NOT_MET`, `MISSING_INFO`).

4. **Trợ lý Điền Đơn (Human-in-the-Loop Application Prep)**:
   - Tự động hóa điền mẫu đăng ký `.docx` dựa trên các trường thông tin đã được thẩm định và xác thực.
   - Cơ chế chặn các trường thiếu hoặc mâu thuẫn để tránh lỗi hồ sơ.
   - Quy trình duyệt hồ sơ (Approve/Reject) và lưu vết Audit Logs đầy đủ.

---

## 🛠️ Công nghệ Sử dụng

- **Backend**: FastAPI (Python 3.10+), SQLite (Database & Audit trail), `docxtpl` & `python-docx` (Document Generator)
- **RAG & NLP**: `sentence-transformers` (Local embedding inference với `multilingual-e5-base`), `rank-bm25` (Lexical matching)
- **Frontend**: React (Vite, TypeScript, TailwindCSS/Vanilla CSS), Lucide Icons
- **Offline Fallback**: Tích hợp sẵn bộ nhớ đệm (embeddings cache) và mô hình cục bộ giúp chạy demo 100% không phụ thuộc Internet.

---

## 🚀 Hướng dẫn Cài đặt & Chạy ứng dụng

### 1. Chuẩn bị Backend
Di chuyển vào thư mục dự án và cài đặt các thư viện Python:
```bash
# Tạo môi trường ảo (khuyến nghị)
python -m venv venv
venv\Scripts\activate  # Windows

# Cài đặt thư viện
pip install -r requirements.txt

# Khởi tạo và seed dữ liệu cơ sở SQLite
$env:PYTHONPATH="backend"
python backend/app/engine/db.py
```

### 2. Tiền tạo bộ nhớ đệm Embeddings (Tùy chọn)
Chạy script để sinh trước cache vector cho tài liệu pháp luật (giúp chạy offline cực nhanh):
```bash
python backend/app/seed/generate_cache.py
```

### 3. Chạy Server Backend
```bash
# Chạy local:
uvicorn app.main:app --host 127.0.0.1 --port 8000

# Chạy để demo qua điện thoại/mạng nội bộ:
uvicorn app.main:app --host 0.0.0.0 --port 8000
```

### 4. Chuẩn bị Frontend
Di chuyển vào thư mục frontend và khởi động Vite dev server:
```bash
cd frontend
npm install

# Chạy local:
npm run dev -- --host 127.0.0.1 --port 5173

# Chạy để demo qua điện thoại/mạng nội bộ:
npm run dev -- --host 0.0.0.0 --port 5173
```
Mở trình duyệt truy cập: `http://localhost:5173` hoặc `http://<ip_may_tinh>:5173` trên điện thoại.


---

## 🧪 Chạy Kiểm thử (Unit Tests)

Dự án cung cấp bộ unit test tự động xác thực độ chuẩn xác của RAG và Eligibility Engine dựa trên dữ liệu Ground Truth:
```bash
$env:PYTHONPATH="backend"
python -m unittest backend/tests/test_golden_path.py
```
> Bộ test kiểm tra 3 doanh nghiệp seed với 6 chính sách khác nhau để đảm bảo logic khớp 100% với kịch bản demo.

---

## 🌐 Triển khai Đám mây & Tối ưu hóa Quy mô (Vercel & Railway/Render)

Hệ thống được thiết kế để triển khai đám mây dễ dàng và đáp ứng quy mô demo **1000+ người dùng đồng thời**:

### 1. Frontend (React) -> Vercel
Frontend được triển khai trên **Vercel** để tận dụng CDN toàn cầu và tự động mở rộng không giới hạn:
1. Di chuyển vào thư mục frontend: `cd frontend`
2. Khởi tạo cấu hình và liên kết dự án: `vercel link`
3. Triển khai bản Production: `vercel --prod`
*(Cấu hình định tuyến Single Page Application đã được thiết lập sẵn trong `vercel.json`)*

### 2. Backend (FastAPI & SQLite) -> Railway / Render
Backend chạy dạng Docker Container hoặc Python service trên các nền tảng như **Railway** hoặc **Render**:

*   **Tối ưu hóa bộ nhớ & Quy mô (1000+ Users):**
    Mô hình local `SentenceTransformer` yêu cầu ~1.1GB dung lượng và tốn nhiều tài nguyên CPU/RAM. Để demo mượt mà trên các gói tài nguyên nhỏ (Hobby tier), hãy cấu hình biến môi trường:
    ```env
    GEMINI_API_KEY=your_gemini_api_key_here
    ```
    Khi phát hiện `GEMINI_API_KEY`, backend sẽ tự động chuyển sang sử dụng **Google Gemini API (text-embedding-004)** để xử lý vector. Điều này giúp giảm lượng RAM tiêu hao xuống dưới **100MB**, tăng tốc độ phản hồi đáng kể và loại bỏ nguy cơ quá tải CPU.

*   **Tối ưu hóa cơ sở dữ liệu SQLite Concurrency:**
    SQLite đã được nâng cấp lên chế độ **WAL (Write-Ahead Logging)** và cấu hình thời gian chờ bận (busy timeout) là 5.0 giây. Sự thay đổi này giúp các tiến trình đọc/ghi diễn ra đồng thời mà không gặp lỗi khóa cơ sở dữ liệu (`database is locked`) khi hàng ngàn người truy cập cùng một lúc.

---

## ✨ Tính năng mới cập nhật (Recent Updates)

Dưới đây là các tính năng và cải tiến đã được tích hợp thành công vào hệ thống P2B:

1. **🌗 Chế độ Sáng/Tối (Light/Dark Mode) & Đa ngôn ngữ (Localization)**:
   - Tích hợp giao diện chuyển đổi Chế độ Sáng/Tối (mặc định là Sáng) và bộ dịch Đa ngôn ngữ Anh/Việt (mặc định là Tiếng Việt) qua các tệp cấu hình JSON (`vi.json`, `en.json`).
   - Đảm bảo độ tương phản màu sắc cao (đạt chuẩn AAA/AA) trong chế độ sáng để tránh khó đọc cho người dùng. Loại bỏ hoàn toàn các dải màu gradient chồng chéo lên tiêu đề chính.

2. **📚 Mở rộng Cơ sở dữ liệu Luật (Legal Corpus Expansion)**:
   - Cập nhật bộ tạo dữ liệu mẫu (`corpus_generator.py`) để nhập sẵn 23 Nghị định và Quyết định thực tế của Chính phủ Việt Nam liên quan đến các ngành công nghệ cao như Trí tuệ Nhân tạo (AI), Công nghiệp Bán dẫn (Semiconductor), Trung tâm Đổi mới sáng tạo Quốc gia (NIC), Năng lượng xanh (Green Energy) và Hỗ trợ doanh nghiệp nhỏ và vừa (SMEs).

3. **🔗 Định tuyến liên kết VBPL.VN chống lỗi 404**:
   - Tự động bắt và chuyển đổi toàn bộ liên kết văn bản pháp luật chứa tham số `ItemID` tĩnh (vốn thường bị thay đổi và trả về lỗi 404 trên cổng thông tin chính phủ) thành các liên kết tìm kiếm trực tiếp theo số hiệu văn bản (`vbpq-timkiem.aspx?Keyword=...`).
   - Biến các danh sách "Văn bản nguồn" trên bảng điều khiển và trong mục trích dẫn thành các nút tương tác trực tiếp.

4. **⚡ Tích hợp Gateway MOJ & Trình xem XML Cục bộ**:
   - Xây dựng công cụ thu thập dữ liệu chuyên sâu ([crawl_five_years.py](file:///e:/VAIC%20Hackathon/backend/app/pipeline/crawl_five_years.py)) kết nối trực tiếp đến cổng API Gateway không xác thực của Bộ Tư pháp (`vbpl-bientap-gateway.moj.gov.vn/api/qtdc/public/doc/{Doc_ID}`) để quét và tải xuống nội dung thô của các văn bản pháp luật trong 5 năm qua.
   - Định dạng dữ liệu thành cấu trúc cây XML chuẩn hóa, lưu trữ đệm trong cột `xml_content` của bảng `legal_documents` trong cơ sở dữ liệu SQLite.
   - Cung cấp API backend và tích hợp Trình xem tài liệu XML (XML Document Viewer Modal) trực tiếp trên giao diện frontend để người dùng tra cứu toàn văn văn bản gốc kèm thông tin metadata chi tiết mà không cần tải lại trang.

