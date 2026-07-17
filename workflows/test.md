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

---

## 5. Potential Failure & Automated Browser Test Cases

The following test scenarios can be automated using `/browser` (Playwright / Puppeteer automation) to guard against regressions:

### 5.1. Authentication & Session Lifecycles
* **TC-001: Invalid Credentials Handling**
  - **Action**: Enter non-existent email or wrong password. Click log in.
  - **Expected Failure**: Verify error toast/message "Invalid email or password" is shown and user remains on log in screen.
* **TC-002: Session Expiration & Auto-Redirect**
  - **Action**: Delete token from LocalStorage or simulate invalid session token, then click any interactive button.
  - **Expected Failure**: Page automatically redirects back to the Login screen with an alert.
* **TC-003: Double Signup Prevention**
  - **Action**: Sign up with an email that already exists in the database.
  - **Expected Failure**: System returns a validation error and blocks the registration.

### 5.2. File Ingestion & Parse Failure Recovery
* **TC-004: Unsupported File Format Ingestion**
  - **Action**: Try uploading an unsupported binary format (e.g. `.exe`, `.bin`).
  - **Expected Failure**: UI displays a clean conversion failure notification, file upload inputs are reset, and backend remains intact.
* **TC-005: Broken / Missing API Key Extraction**
  - **Action**: Simulate a missing GEMINI_API_KEY environment variable on backend.
  - **Expected Failure**: Uploading files reports an HTTP 500 error warning "GEMINI_API_KEY environment variable is missing" instead of crashing.
* **TC-006: Temporary Upload Security Audit**
  - **Action**: Verify that files uploaded to temp storage are completely removed from server filesystem upon completion or error (tested programmatically via system process check).

### 5.3. Gated Approval & State Security
* **TC-007: Gated Review Bypass Prevention**
  - **Action**: Select a policy that resolves to `NOT_MET` (e.g., Green Innovation Grant for FDI SemiVina Corp). Programmatically trigger a draft approval post request.
  - **Expected Failure**: Backend rejects with HTTP 400, and UI blocks/hides the "Approve" button.
* **TC-008: Missing/Conflicted Fields Block**
  - **Action**: Try to approve a draft while a core passport field is marked as `CONFLICTED` or `MISSING`.
  - **Expected Failure**: Review area displays warnings block listing the conflicting elements, and blocks approval.

### 5.4. Tenant Isolation & Mode Toggles
* **TC-009: Cross-Tenant Data Access Leak**
  - **Action**: Login as Tenant A (`aitech@p2b.vn`). Try to navigate to Tenant B's draft URL or send an API request to Tenant B's endpoints.
  - **Expected Failure**: Server returns HTTP 403 Forbidden.
* **TC-010: Dynamic Persona Transition Sync**
  - **Action**: Open user settings, change user type from **Company Manager** to **Individual**, and click Save.
  - **Expected Outcome**: The dashboard instantly cleans its state and renders the Personal Passport update forms, and the processed documents history loads Individual records.

