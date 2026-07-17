# P2B Frontend Visual and Build Verification Workflow

This document outlines the step-by-step workflow to verify the P2B platform's frontend build compilation, real-time decree query RAG engine, and visual features across different company personas.

---

## 1. Prerequisites & Environment Setup

Before starting verification, ensure the Python virtual environment and Node.js dependencies are fully set up.

### Start Backend Server (port 8000)
Run the FastAPI application from the `backend` directory. Uvicorn will load the local multilingual E5 embedding model and start the REST API.
```powershell
# From the workspace root (local-only)
cd backend
python -m uvicorn app.main:app --host 127.0.0.1 --port 8000

# Or to run for phone/network demo access:
python -m uvicorn app.main:app --host 0.0.0.0 --port 8000
```
*Wait for `INFO: Application startup complete.` to ensure the SentenceTransformer model has loaded.*

### Start Frontend Dev Server (port 5173)
Run the Vite development server.
```powershell
# From the workspace root (local-only)
cd frontend
npm run dev -- --host 127.0.0.1

# Or to run for phone/network demo access:
cd frontend
npm run dev -- --host 0.0.0.0
```

---

## 2. Compilation and Type Sanity Check

To guarantee no TypeScript compilation issues or missing imports:
```powershell
cd frontend
npm run build
```
Verify the build output completes successfully with no warnings or errors.

---

## 3. Visual Verification Scenarios (Multi-Persona)

The platform supports multiple corporate personas. Select them via the company dropdown in the top header.

### Scenario A: AI Innovation Persona (AItech Vietnam LLC)
Verify the AI extraction and conflict resolution path:
1. **Dropdown Selection:** Choose **AItech Vietnam LLC** in the top header.
2. **Real-time Decree Query:** In the search bar, type `chương trình nghiên cứu trí tuệ nhân tạo` and click **Search**.
   - **Expected Outcome:** The RAG engine queries the legal decree chunks. **Chương Trình Khoa Học Công Nghệ Quốc Gia Về Trí Tuệ Nhân Tạo** (citiing Decision 127/QĐ-TTg) re-ranks to the top.
3. **Upload File / AI Vision extraction:**
   - Click **Upload File** in the Company Passport card.
   - Enter `https://aitechvn.com/report.pdf` and click **Bắt đầu Trích xuất AI**.
   - Verify simulated OCR scanning stepper finishes.
   - **Expected Outcome:** `Registered Capital` passport field status is flagged as `CONFLICTED` (Red badge).
4. **Conflict Resolution:** Click **Đối Chiếu** on that row. Select **Sử dụng Nguồn B** in the side-by-side modal.
   - **Expected Outcome:** The status changes to `USER_CONFIRMED` (Grey badge) and the value updates to `12,000,000,000 VND`.
5. **Interactive Required Documents Checklist:**
   - In the checklist panel (bottom right), click **Bỏ qua** (Waive) on a missing document.
   - **Expected Outcome:** The document title strikes through and its badge changes to `WAIVED (MIỄN NỘP)`.
6. **Policy Change Alert:**
   - Click the pulsing yellow **Có Thay Đổi** badge on the top card.
   - **Expected Outcome:** A side-by-side legal clause diff comparison modal opens showing R&D threshold changes from 2.0% to 1.5%.
7. **Document Draft Creation:**
   - Click **Tạo Hồ Sơ Nháp (Draft)**. Status changes to `PENDING_REVIEW` in purple.
   - Click **Phê Duyệt & Xuất Đơn (Approve)**. Status changes to `GENERATED` in green with a `Tải Đơn (.docx)` download button.

---

### Scenario B: Global Semiconductor FDI Persona (FDI SemiVina Corp)
Verify FDI rules and positive eligibility outcomes:
1. **Dropdown Selection:** Choose **FDI SemiVina Corp** in the top header.
2. **Real-time Decree Query:** Search for `bán dẫn fdi` or `ưu đãi r&d`.
   - **Expected Outcome:** **Chính Sách Hỗ Trợ Đặc Biệt Dự Án FDI Bán Dẫn và R&D** (citing Investment Law 2020) ranks top.
3. **Eligibility Check:**
   - **Expected Outcome:** The overall eligibility status resolves to **MET** (Green indicator).
   - Click **Hoạt động trong lĩnh vực bán dẫn...** or **Đầu tư R&D đặc biệt**.
   - Verify the sliding drawer displays high confidence evidence quotes from the company's pitch deck.

---

### Scenario C: Green CleanTech Persona (SolarGreen Tech JSC)
Verify SME tax reductions and green incentives:
1. **Dropdown Selection:** Choose **SolarGreen Tech JSC** in the top header.
2. **Real-time Decree Query:** Search for `năng lượng xanh` or `quỹ tăng trưởng xanh`.
   - **Expected Outcome:** **Quỹ Tài Trợ Phát Triển Công Nghệ Xanh và Năng Lượng Tái Tạo** (citing Decision 1658/QĐ-TTg) ranks top.
3. **Eligibility Check:**
   - **Expected Outcome:** The overall eligibility is **MET**.
   - Click the rules to view green technology and asset-size citations.

---

## 4. Visual Verification Artifacts

All screenshots and screen recording files are saved in the artifact directory.
- `home_page.png` (Dashboard overview)
- `sliding_drawer.png` (Decree citations drawer)
- `conflict_modal_visible.png` (Side-by-side conflict modal)
- `policy_diff_modal.png` (Decree change diff modal)
- `interactive_checklist_and_drawer.png` (Waived documents)
- `recording.webm` (Full walkthrough video)
