# P2B Frontend Visual and Build Verification Workflow

This document outlines the step-by-step workflow to verify the P2B platform's frontend build compilation and visual features using automated tools and a browser visual testing agent.

---

## 1. Prerequisites & Environment Setup

Before starting verification, ensure the Python virtual environment and Node.js dependencies are fully set up.

### Start Backend Server (port 8000)
Run the FastAPI application from the `backend` directory. Uvicorn will load the local multilingual E5 embedding model and start the REST API.
```powershell
# From the workspace root
cd backend
python -m uvicorn app.main:app --host 127.0.0.1 --port 8000
```
*Wait for `INFO: Application startup complete.` to ensure the SentenceTransformer model has loaded.*

### Start Frontend Dev Server (port 5173)
Run the Vite development server with Tailwind CSS v4 configured.
```powershell
# From the workspace root
cd frontend
npm run dev
```

---

## 2. Compilation and Type Sanity Check

To guarantee no TypeScript compilation issues or missing imports:
```powershell
cd frontend
npm run build
```
Verify the build output completes successfully with no warnings or errors. Expected output chunk sizes:
- `dist/index.html` ~ 0.45 kB
- `dist/assets/index-*.css` ~ 37.3 kB
- `dist/assets/index-*.js` ~ 249 kB

---

## 3. Visual Verification Trajectory

The browser subagent visual verification follows this visual checking path:

### Step 3.1: Dashboard Verification
- **Target URL**: `http://localhost:5173/`
- **Action**: Verify the responsive side-by-side grid is populated with seeded Company Passport facts (left column) and Ranked Policy Opportunities (right column) fetched from the backend.
- **Artifact**: `home_page.png`

### Step 3.2: Eligibility Sliding Drawer
- **Action**: Click a rule under the eligibility list (e.g. *"Hoạt động trong ngành Trí tuệ nhân tạo (Artificial Intelligence)"*).
- **Expected Outcome**: A side-drawer slides in from the right. It displays:
  - Evaluation status badge (`MET`, `NOT_MET`, or `MISSING_INFO`).
  - Company Evidence quote, source location, and URI.
  - Policy Clause description, expected operator condition, law quote citation, and original document URL link.
- **Artifact**: `sliding_drawer.png`

### Step 3.3: Document Upload Stepper Simulation
- **Action**: Click **Upload File** in the Company Passport card. Enter a mock document target link (e.g., `https://aitechvn.com/report.pdf`) and click **Bắt đầu Trích xuất AI**.
- **Expected Outcome**:
  - Step 1: Upload progress bar completes.
  - Step 2: Simulated OCR laser scanning runs (a glowing horizontal laser line moves vertically on the mock document card).
  - Step 3: Extracted facts pop up one-by-one with matching confidence scores.
- **Artifacts**: `document_upload_modal.png` (upload state), `conflict_state.png` (completion state highlighting conflict logs)

### Step 3.4: Conflict Resolution UI
- **Action**: When scanning completes, the `Registered Capital` passport field status is flagged as `CONFLICTED`. Click **Đối Chiếu** on that row.
- **Expected Outcome**: A modal appears showing Source A (old database value) and Source B (AI Vision extracted value) side-by-side. Click **Sử dụng Nguồn B** or enter a custom override value.
- **Expected Backend Integration**: The passport state updates to `USER_CONFIRMED` via API PUT request, and the eligibility check runs automatically.
- **Artifact**: `conflict_modal_visible.png`

### Step 3.5: Interactive Required Documents Checklist
- **Action**: Scroll to **Required Documents Checklist**. Select a missing document and click **Bỏ qua** (Waive).
- **Expected Outcome**: The checklist item status updates to `WAIVED`, strikes through the text, and disables the toggle checkbox.
- **Artifact**: `interactive_checklist_and_drawer.png`

### Step 3.6: Policy Change Alerts
- **Action**: Locate the top opportunity card highlighting a yellow pulsing **Có Thay Đổi** badge. Click on the badge.
- **Expected Outcome**: A diff modal appears displaying a side-by-side red-green comparison comparing the old legislative clause vs the new updated legislative clause (e.g., R&D requirement changed from 2.0% to 1.5%).
- **Artifact**: `policy_diff_modal.png`

---

## 4. Visual Verification Artifacts

All screenshots and screen recording files are saved in:
`C:\Users\lnminh1411\.gemini\antigravity\brain\e0f22a0d-4008-440e-921d-01d6ea4f704c\`

### Captured Screenshots
1. Dashboard home layout: `home_page.png`
2. Eligibility drawer details: `sliding_drawer.png`
3. OCR extraction start: `document_upload_modal.png`
4. Stepper conflict log: `conflict_state.png`
5. Side-by-side conflict resolution: `conflict_modal_visible.png`
6. Tri-state documents checklist: `interactive_checklist_and_drawer.png`
7. Legal diff modal: `policy_diff_modal.png`

### Full Web Session Video Recording
- Screen capture recording: `recording.webm`
