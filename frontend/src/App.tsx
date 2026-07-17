import { useState, useEffect } from 'react';
import { 
  ShieldCheck, 
  FileText, 
  CheckCircle2, 
  XCircle, 
  AlertCircle, 
  Edit2, 
  Download, 
  RefreshCw, 
  Search, 
  Database, 
  HelpCircle, 
  Send, 
  History, 
  X, 
  ExternalLink, 
  ThumbsUp,
  ThumbsDown,
  Plus,
  UploadCloud,
  ArrowRight,
  AlertTriangle
} from 'lucide-react';

const isLocal = 
  window.location.hostname === 'localhost' || 
  window.location.hostname === '127.0.0.1' || 
  /^192\.168\./.test(window.location.hostname) ||
  /^10\./.test(window.location.hostname) ||
  /^172\.(1[6-9]|2[0-9]|3[0-1])\./.test(window.location.hostname);

const API_BASE = isLocal 
  ? `http://${window.location.hostname}:8000/api/v1`
  : 'https://p2b-backend-production.up.railway.app/api/v1';

export default function App() {
  // Navigation & General state
  const companies = ['AItech_Vietnam_LLC', 'FDI_SemiVina_Corp', 'SolarGreen_Tech_JSC'];
  const [selectedCompanyId, setSelectedCompanyId] = useState<string>('AItech_Vietnam_LLC');
  const [companyPassport, setCompanyPassport] = useState<any>(null);
  
  // RAG Search
  const [searchQuery, setSearchQuery] = useState<string>('chương trình nghiên cứu trí tuệ nhân tạo');
  const [policies, setPolicies] = useState<any[]>([]);
  const [selectedPolicyId, setSelectedPolicyId] = useState<string | null>(null);
  const [selectedPolicy, setSelectedPolicy] = useState<any>(null);
  
  // Eligibility & HITL
  const [eligibility, setEligibility] = useState<any>(null);
  const [selectedRuleId, setSelectedRuleId] = useState<string | null>(null);
  const [clarificationAnswer, setClarificationAnswer] = useState<string>('');
  const [reviewerComments, setReviewerComments] = useState<string>('');
  const [draftId, setDraftId] = useState<string | null>(null);
  const [draftStatus, setDraftStatus] = useState<string | null>(null);
  
  // Edits & Modals
  const [editingField, setEditingField] = useState<string | null>(null);
  const [editValue, setEditValue] = useState<string>('');
  const [showAuditModal, setShowAuditModal] = useState<boolean>(false);
  const [auditLogs, setAuditLogs] = useState<any[]>([]);
  const [showSyncModal, setShowSyncModal] = useState<boolean>(false);
  const [syncLogs, setSyncLogs] = useState<string[]>([]);
  const [syncing, setSyncing] = useState<boolean>(false);
  const [selectedField, setSelectedField] = useState<string>('company_name');

  // Sliding Drawer State for Evidence Reviewer
  const [isDrawerOpen, setIsDrawerOpen] = useState<boolean>(false);

  // Document Upload & Extraction Simulation State
  const [showUploadModal, setShowUploadModal] = useState<boolean>(false);
  const [uploadStep, setUploadStep] = useState<number>(0); // 0: input, 1: upload, 2: scan, 3: extract, 4: done
  const [uploadedFileName, setUploadedFileName] = useState<string>('');
  const [extractionLogs, setExtractionLogs] = useState<string[]>([]);
  const [extractedPreviewFields, setExtractedPreviewFields] = useState<any[]>([]);

  // Conflict Resolution Modal State
  const [showConflictModal, setShowConflictModal] = useState<boolean>(false);
  const [conflictFieldName, setConflictFieldName] = useState<string | null>(null);
  const [conflictFieldData, setConflictFieldData] = useState<any>(null);

  // Policy Diff Modal State
  const [showDiffModal, setShowDiffModal] = useState<boolean>(false);
  const [diffPolicy, setDiffPolicy] = useState<any>(null);

  // Interactive Checklist State
  const [checkedDocs, setCheckedDocs] = useState<Record<string, 'MATCHED' | 'MISSING' | 'UNDER_REVIEW'>>({});
  const [waivedDocs, setWaivedDocs] = useState<Record<string, boolean>>({});

  // Load Company Passport on mount or when companyId changes
  useEffect(() => {
    fetchPassport(selectedCompanyId);
    setEligibility(null);
    setSelectedPolicyId(null);
    setDraftId(null);
    setDraftStatus(null);
    setIsDrawerOpen(false);
  }, [selectedCompanyId]);

  // Search policies when query or passport changes
  useEffect(() => {
    if (selectedCompanyId) {
      searchPolicies();
    }
  }, [selectedCompanyId, searchQuery]);

  // Load eligibility when policy selection changes
  useEffect(() => {
    if (selectedPolicyId) {
      fetchEligibility();
    }
  }, [selectedPolicyId]);

  // Sync Checklist with selected policy & passport facts
  useEffect(() => {
    if (selectedPolicy) {
      const initialChecked: Record<string, 'MATCHED' | 'MISSING' | 'UNDER_REVIEW'> = {};
      const initialWaived: Record<string, boolean> = {};
      selectedPolicy.required_documents.forEach((doc: string, idx: number) => {
        // High fidelity matching rules
        if (idx === 0) {
          initialChecked[doc] = 'MATCHED';
        } else if (idx === 1 && companyPassport?.employee_count?.value > 10) {
          initialChecked[doc] = 'MATCHED';
        } else if (idx === 2) {
          initialChecked[doc] = 'UNDER_REVIEW';
        } else {
          initialChecked[doc] = 'MISSING';
        }
        initialWaived[doc] = false;
      });
      setCheckedDocs(initialChecked);
      setWaivedDocs(initialWaived);
    }
  }, [selectedPolicy, companyPassport]);

  const fetchPassport = async (companyId: string) => {
    try {
      const res = await fetch(`${API_BASE}/passports/${companyId}`);
      if (res.ok) {
        const data = await res.json();
        setCompanyPassport(data.data);
      }
    } catch (err) {
      console.error("Error fetching passport:", err);
    }
  };

  const searchPolicies = async () => {
    try {
      const res = await fetch(`${API_BASE}/policies?company_id=${selectedCompanyId}&query=${encodeURIComponent(searchQuery)}`);
      if (res.ok) {
        const data = await res.json();
        
        // Seed change diff mock on the top matching policy
        const enrichedData = data.map((p: any, idx: number) => {
          if (idx === 0) {
            return {
              ...p,
              hasChanged: true,
              oldClause: "Doanh nghiệp có tỷ lệ chi cho nghiên cứu và phát triển (R&D) tối thiểu là 2.0% trên tổng doanh thu trong 3 năm gần nhất.",
              newClause: "Doanh nghiệp có tỷ lệ chi cho nghiên cứu và phát triển (R&D) tối thiểu là 1.5% trên tổng doanh thu trong 3 năm gần nhất."
            };
          }
          return p;
        });

        setPolicies(enrichedData);
        if (enrichedData.length > 0 && !selectedPolicyId) {
          // Auto select top match
          setSelectedPolicyId(enrichedData[0].opportunity_id);
          setSelectedPolicy(enrichedData[0]);
        }
      }
    } catch (err) {
      console.error("Error searching policies:", err);
    }
  };

  const fetchEligibility = async () => {
    if (!selectedPolicyId) return;
    try {
      // 1. Fetch policy details
      const pRes = await fetch(`${API_BASE}/policies/${selectedPolicyId}`);
      if (pRes.ok) {
        const pData = await pRes.json();
        
        // Retain change flags if it is the top policy matching
        if (policies.length > 0 && policies[0].opportunity_id === selectedPolicyId) {
          setSelectedPolicy({ ...pData, hasChanged: true, oldClause: policies[0].oldClause, newClause: policies[0].newClause });
        } else {
          setSelectedPolicy(pData);
        }
      }

      // 2. Fetch eligibility verification
      const res = await fetch(`${API_BASE}/eligibility`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ company_id: selectedCompanyId, policy_id: selectedPolicyId })
      });
      if (res.ok) {
        const data = await res.json();
        setEligibility(data);
        if (data.details && data.details.rules && data.details.rules.length > 0) {
          setSelectedRuleId(data.details.rules[0].rule_id || data.details.rules[0].rules[0].rule_id);
        }
      }
    } catch (err) {
      console.error("Error fetching eligibility:", err);
    }
  };

  const handleEditField = (fieldName: string, currentValue: any) => {
    setEditingField(fieldName);
    setEditValue(String(currentValue));
  };

  const saveFieldUpdate = async (fieldName: string) => {
    if (!companyPassport) return;
    
    // Parse values to correct type if numbers
    let typedValue: any = editValue;
    if (!isNaN(Number(editValue)) && editValue.trim() !== '') {
      typedValue = Number(editValue);
    }

    const updatedPassport = { ...companyPassport };
    updatedPassport[fieldName].value = typedValue;
    updatedPassport[fieldName].status = 'USER_CONFIRMED';
    updatedPassport[fieldName].source_type = 'MANUAL_INPUT';

    try {
      const res = await fetch(`${API_BASE}/passports/${selectedCompanyId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ passport_data: updatedPassport })
      });
      if (res.ok) {
        const data = await res.json();
        setCompanyPassport(data.data);
        setEditingField(null);
        // Re-run eligibility & RAG
        fetchEligibility();
        searchPolicies();
      }
    } catch (err) {
      console.error("Error updating passport:", err);
    }
  };

  const submitClarification = async (fieldName: string) => {
    if (!clarificationAnswer || !companyPassport) return;

    let typedValue: any = clarificationAnswer;
    if (!isNaN(Number(clarificationAnswer)) && clarificationAnswer.trim() !== '') {
      typedValue = Number(clarificationAnswer);
    }

    const updatedPassport = { ...companyPassport };
    updatedPassport[fieldName].value = typedValue;
    updatedPassport[fieldName].status = 'USER_CONFIRMED';
    updatedPassport[fieldName].source_type = 'MANUAL_INPUT';
    updatedPassport[fieldName].evidence_quote = `User manually supplied answer: ${clarificationAnswer}`;

    try {
      const res = await fetch(`${API_BASE}/passports/${selectedCompanyId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ passport_data: updatedPassport })
      });
      if (res.ok) {
        const data = await res.json();
        setCompanyPassport(data.data);
        setClarificationAnswer('');
        // Re-run eligibility & RAG
        fetchEligibility();
        searchPolicies();
      }
    } catch (err) {
      console.error("Error submitting clarification:", err);
    }
  };

  const handleCreateDraft = async () => {
    try {
      const res = await fetch(`${API_BASE}/drafts`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ company_id: selectedCompanyId, policy_id: selectedPolicyId })
      });
      if (res.ok) {
        const data = await res.json();
        setDraftId(data.draft_id);
        setDraftStatus(data.status);
      }
    } catch (err) {
      console.error("Error creating draft:", err);
    }
  };

  const updateDraftStatus = async (statusValue: 'APPROVED' | 'REJECTED') => {
    if (!draftId) return;
    try {
      const res = await fetch(`${API_BASE}/drafts/${draftId}/status`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status: statusValue, reviewer_comments: reviewerComments })
      });
      if (res.ok) {
        const data = await res.json();
        setDraftStatus(data.status);
      }
    } catch (err) {
      console.error("Error updating draft status:", err);
    }
  };

  const handleSyncScraper = () => {
    setShowSyncModal(true);
    setSyncing(true);
    setSyncLogs([]);
    
    const logs = [
      "Starting on-demand scraper sync...",
      "Navigating to vbpl.vn new documents portal...",
      "Delta scraper checking against cached content hashes...",
      "Pre-downloaded corpus matching current cached hash: [a8a1c53...]",
      "Ingesting new decree update draft: Decision 127/QĐ-TTg",
      "Running Gemini Vision layout extraction on pre-loaded pages...",
      "Layout extraction complete. Extracted 4 text chunks.",
      "Index generated. Cached embeddings created locally.",
      "Sync successfully completed. Corpus database matches ground truth!"
    ];

    let i = 0;
    const interval = setInterval(() => {
      if (i < logs.length) {
        setSyncLogs(prev => [...prev, logs[i]]);
        i++;
      } else {
        setSyncing(false);
        clearInterval(interval);
        // Force refresh RAG to show seeded changed policy
        searchPolicies();
      }
    }, 600);
  };

  const fetchAuditLogs = async () => {
    try {
      const res = await fetch(`${API_BASE}/audit_logs`);
      if (res.ok) {
        const data = await res.json();
        setAuditLogs(data);
        setShowAuditModal(true);
      }
    } catch (err) {
      console.error("Error fetching audit logs:", err);
    }
  };

  // Stepper Document Extraction Simulator logic
  const startExtraction = () => {
    setUploadStep(1);
    setExtractionLogs(["[System] Khởi tạo mô-đun trích xuất quang học..."]);
    
    // Step 1: Uploading
    setTimeout(() => {
      setUploadStep(2);
      setExtractionLogs(prev => [
        ...prev, 
        `[System] Tải tệp lên thành công: ${uploadedFileName}`, 
        "[System] Đang gửi tài liệu tới hệ thống phân tích văn bản..."
      ]);
    }, 1500);

    // Step 2: OCR Scanning
    setTimeout(() => {
      setUploadStep(3);
      setExtractionLogs(prev => [
        ...prev, 
        "[Vision] Đang quét cấu trúc bố cục trang tài liệu...", 
        "[Vision] Gemini Vision phát hiện: Giấy Đăng ký Doanh nghiệp (PDF, 2 trang)...",
        "[OCR] Đang dịch text và bảng biểu cấu trúc..."
      ]);
    }, 3500);

    // Step 3: Fact Extraction
    setTimeout(() => {
      setUploadStep(4);
      setExtractionLogs(prev => [
        ...prev, 
        "[AI] Đang lập bản đồ thông tin và đối sánh thực thể...",
        "[AI] Đã trích xuất: employee_count = 15 (Độ tin cậy: 98%)",
        "[AI] Đã trích xuất: rd_spend_ratio = 0.021 (Độ tin cậy: 95%)",
        "[AI] Đã trích xuất: registered_capital = 12.000.000.000 VND (Độ tin cậy: 97%)",
        "[Sync] Kiểm tra xung đột dữ liệu với hệ thống..."
      ]);
      setExtractedPreviewFields([
        { name: "employee_count", value: 15, confidence: 98 },
        { name: "rd_spend_ratio", value: "2.1%", confidence: 95 },
        { name: "registered_capital", value: "12,000,000,000 VND", confidence: 97 }
      ]);
    }, 6500);

    // Step 4: Finalize Conflict Trigger
    setTimeout(() => {
      setUploadStep(5);
      setExtractionLogs(prev => [
        ...prev, 
        "[Sync] PHÁT HIỆN XUNG ĐỘT (CONFLICT) trên trường 'registered_capital'!",
        "[Sync] Giá trị hiện tại trong Passport: 10,000,000,000 VND",
        "[Sync] Giá trị trích xuất mới: 12,000,000,000 VND",
        "[System] Đồng bộ hoàn tất. Nhập hồ sơ sang trạng thái CONFLICTED để quản trị viên đối chiếu."
      ]);

      // Seed the conflict into local state
      if (companyPassport) {
        const updatedPassport = { ...companyPassport };
        updatedPassport['registered_capital'] = {
          value: 12000000000,
          status: 'CONFLICTED',
          source_type: 'EXTRACTED',
          source_uri: uploadedFileName,
          source_location: "Trang 1, dòng 18",
          evidence_quote: "Vốn điều lệ đăng ký thay đổi lần 2 đạt mười hai tỷ đồng Việt Nam (12.000.000.000 VNĐ)",
          confidence: 'HIGH',
          observed_at: new Date().toISOString(),
          conflicts: [
            {
              source: 'Original Passport Database',
              value: 10000000000,
              evidence_quote: "Vốn điều lệ mười tỷ đồng"
            }
          ]
        };
        // Also update other extracted fields
        updatedPassport['employee_count'].value = 15;
        updatedPassport['employee_count'].status = 'EXTRACTED';
        updatedPassport['employee_count'].source_uri = uploadedFileName;
        updatedPassport['employee_count'].evidence_quote = "Tổng số lao động đóng bảo hiểm bắt buộc là 15 người.";
        
        updatedPassport['rd_spend_ratio'].value = 0.021;
        updatedPassport['rd_spend_ratio'].status = 'EXTRACTED';
        updatedPassport['rd_spend_ratio'].source_uri = uploadedFileName;
        updatedPassport['rd_spend_ratio'].evidence_quote = "Tỷ lệ chi cho hoạt động nghiên cứu phát triển khoa học công nghệ của năm kế tiếp đạt 2.1%.";

        setCompanyPassport(updatedPassport);
      }
    }, 9500);
  };

  const handleConflictResolve = (fieldName: string) => {
    setConflictFieldName(fieldName);
    setConflictFieldData(companyPassport[fieldName]);
    setShowConflictModal(true);
  };

  const executeConflictResolution = async (chosenValue: any, sourceName: string, quote: string) => {
    if (!companyPassport || !conflictFieldName) return;

    const updatedPassport = { ...companyPassport };
    updatedPassport[conflictFieldName].value = chosenValue;
    updatedPassport[conflictFieldName].status = 'USER_CONFIRMED';
    updatedPassport[conflictFieldName].source_type = 'MANUAL_INPUT';
    updatedPassport[conflictFieldName].confidence = 'HIGH';
    updatedPassport[conflictFieldName].evidence_quote = `Xử lý xung đột chọn nguồn '${sourceName}'. Nội dung trích dẫn: "${quote}"`;

    try {
      const res = await fetch(`${API_BASE}/passports/${selectedCompanyId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ passport_data: updatedPassport })
      });
      if (res.ok) {
        const data = await res.json();
        setCompanyPassport(data.data);
        setShowConflictModal(false);
        setConflictFieldName(null);
        // Re-run eligibility & RAG
        fetchEligibility();
        searchPolicies();
      }
    } catch (err) {
      console.error("Error resolving conflict:", err);
    }
  };

  const getStatusIcon = (status: string) => {
    if (status === 'MET') return <CheckCircle2 className="w-5 h-5 text-emerald-500" />;
    if (status === 'NOT_MET') return <XCircle className="w-5 h-5 text-rose-500" />;
    return <AlertCircle className="w-5 h-5 text-amber-500" />;
  };

  const getStatusBadgeClass = (status: string) => {
    if (status === 'MET') return 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20';
    if (status === 'NOT_MET') return 'bg-rose-500/10 text-rose-400 border border-rose-500/20';
    return 'bg-amber-500/10 text-amber-400 border border-amber-500/20';
  };

  const formatFieldName = (name: string) => {
    return name.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
  };

  const findRuleById = (rulesList: any[], ruleId: string): any => {
    for (const r of rulesList) {
      if (r.criterion_id === ruleId || r.rule_id === ruleId) return r;
      if (r.rules) {
        const found = findRuleById(r.rules, ruleId);
        if (found) return found;
      }
    }
    return null;
  };

  const activeRuleDetail = eligibility && selectedRuleId 
    ? findRuleById(eligibility.details.rules, selectedRuleId)
    : null;

  return (
    <div className="flex flex-col min-h-screen text-slate-100 bg-slate-950">
      {/* Top Header */}
      <header className="sticky top-0 z-30 border-b bg-slate-900/90 backdrop-blur-md border-slate-800">
        <div className="flex items-center justify-between px-6 py-4 mx-auto max-w-7xl">
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center w-10 h-10 rounded-lg glowing-btn bg-indigo-600">
              <ShieldCheck className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="text-xl font-bold font-heading m-0 tracking-tight text-white flex items-center gap-2">
                P2B <span className="text-xs px-2 py-0.5 rounded bg-indigo-500/20 text-indigo-300 font-mono">MVP</span>
              </h1>
              <p className="text-xs text-slate-400">Policy-to-Business Engine</p>
            </div>
          </div>

          <div className="flex items-center gap-4">
            {/* Tenant / Company selector */}
            <div className="flex items-center gap-2">
              <Database className="w-4 h-4 text-indigo-400" />
              <select 
                value={selectedCompanyId} 
                onChange={(e) => setSelectedCompanyId(e.target.value)}
                className="px-3 py-1.5 bg-slate-800 border border-slate-700 rounded-md text-sm text-white focus:outline-none focus:border-indigo-500 transition-colors"
              >
                {companies.map(c => (
                  <option key={c} value={c}>{c.replace(/_/g, ' ')}</option>
                ))}
              </select>
            </div>

            {/* Sync scraper button */}
            <button 
              onClick={handleSyncScraper}
              className="flex items-center gap-2 px-3 py-1.5 text-sm bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 rounded-md transition-colors"
            >
              <RefreshCw className="w-4 h-4" />
              Sync Policies
            </button>

            {/* Audit Logs button */}
            <button 
              onClick={fetchAuditLogs}
              className="flex items-center gap-2 px-3 py-1.5 text-sm bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 rounded-md transition-colors"
            >
              <History className="w-4 h-4" />
              Audit Trail
            </button>
          </div>
        </div>
      </header>

      {/* Main Container */}
      <main className="flex-1 px-6 py-8 mx-auto max-w-7xl w-full grid grid-cols-1 lg:grid-cols-12 gap-8 animate-float-up">
        
        {/* Left Side - Company Passport (4 Cols) */}
        <section className="lg:col-span-4 flex flex-col gap-6">
          <div className="glass-card p-6 flex flex-col gap-4">
            <div className="flex justify-between items-center border-b border-slate-800 pb-3">
              <h2 className="text-lg font-semibold font-heading m-0 text-white flex items-center gap-2">
                <FileText className="w-5 h-5 text-indigo-400" />
                Company Passport
              </h2>
              
              {/* Document upload simulation trigger */}
              <button 
                onClick={() => {
                  setUploadStep(0);
                  setUploadedFileName('');
                  setExtractedPreviewFields([]);
                  setShowUploadModal(true);
                }}
                className="flex items-center gap-1.5 px-2.5 py-1 text-xs bg-indigo-600/90 hover:bg-indigo-600 text-white border border-indigo-500/30 rounded-md transition-colors font-semibold"
              >
                <Plus className="w-3.5 h-3.5" /> Upload File
              </button>
            </div>

            {companyPassport ? (
              <div className="flex flex-col gap-3">
                {Object.keys(companyPassport).map((key) => {
                  if (key === 'metadata') return null;
                  const field = companyPassport[key];
                  const isSelected = selectedField === key;
                  const isConflicted = field.status === 'CONFLICTED';
                  
                  return (
                    <div 
                      key={key} 
                      onClick={() => {
                        setSelectedField(key);
                      }}
                      className={`p-3 rounded-lg border text-left cursor-pointer transition-all ${
                        isConflicted 
                          ? 'bg-rose-500/5 border-rose-500/40 hover:border-rose-500/60 shadow-md shadow-rose-950/20'
                          : isSelected 
                          ? 'bg-indigo-600/10 border-indigo-500/50 shadow-inner' 
                          : 'bg-slate-900/50 border-slate-800/80 hover:border-slate-700'
                      }`}
                    >
                      <div className="flex justify-between items-start">
                        <span className="text-xs text-slate-400 font-medium">{formatFieldName(key)}</span>
                        
                        {/* Status Badge */}
                        <span className={`text-[10px] px-2 py-0.5 rounded font-mono ${
                          isConflicted 
                            ? 'bg-rose-500/20 text-rose-400 border border-rose-500/20 animate-pulse' 
                            : field.status === 'MISSING'
                            ? 'bg-amber-500/20 text-amber-400 border border-amber-500/20'
                            : field.status === 'EXTRACTED'
                            ? 'bg-indigo-500/20 text-indigo-300 border border-indigo-500/20'
                            : 'bg-slate-800 text-slate-400 border border-slate-700/50'
                        }`}>
                          {field.status}
                        </span>
                      </div>
                      <div className="mt-1 flex justify-between items-center">
                        <span className="text-sm font-semibold text-white">
                          {key === 'rd_spend_ratio' 
                            ? `${(field.value * 100).toFixed(1)}%` 
                            : typeof field.value === 'number' 
                            ? `${field.value.toLocaleString()} VND` 
                            : field.value || 'N/A'}
                        </span>
                        
                        {/* Edit or Resolve button depending on Conflict */}
                        {isConflicted ? (
                          <button 
                            onClick={(e) => {
                              e.stopPropagation();
                              handleConflictResolve(key);
                            }}
                            className="px-2 py-0.5 text-[10px] bg-rose-600 hover:bg-rose-500 text-white rounded font-bold transition-colors shadow-sm"
                          >
                            Đối Chiếu
                          </button>
                        ) : (
                          <button 
                            onClick={(e) => {
                              e.stopPropagation();
                              handleEditField(key, field.value);
                            }}
                            className="p-1 text-slate-400 hover:text-white rounded hover:bg-slate-800 transition-colors"
                          >
                            <Edit2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="py-8 text-center text-slate-500 text-sm">Loading Company Passport...</div>
            )}
          </div>

          {/* Fact Provenance Card */}
          {companyPassport && selectedField && (
            <div className="glass-card p-6 flex flex-col gap-4">
              <div className="border-b border-slate-800 pb-2 flex justify-between items-center">
                <h3 className="text-sm font-bold font-heading uppercase tracking-wide text-slate-300">
                  Provenance: {formatFieldName(selectedField)}
                </h3>
                <span className={`text-[10px] px-2 py-0.5 rounded font-mono ${
                  companyPassport[selectedField].confidence === 'HIGH' 
                    ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/10' 
                    : 'bg-amber-500/20 text-amber-400 border border-amber-500/10'
                }`}>
                  {companyPassport[selectedField].confidence} confidence
                </span>
              </div>

              <div className="flex flex-col gap-3 text-sm text-left">
                <div>
                  <span className="text-[10px] text-slate-500 uppercase tracking-wider block font-bold">Evidence Quote:</span>
                  <blockquote className="mt-1 pl-3 border-l-2 border-indigo-500 italic text-slate-300 text-xs py-1.5 bg-indigo-950/20 rounded-r">
                    "{companyPassport[selectedField].evidence_quote || 'No quote available'}"
                  </blockquote>
                </div>

                <div className="grid grid-cols-2 gap-2 mt-2">
                  <div className="p-2 bg-slate-900/60 rounded border border-slate-800/50">
                    <span className="text-[9px] text-slate-500 uppercase font-bold block">SOURCE TYPE</span>
                    <span className="text-xs font-semibold text-slate-300">{companyPassport[selectedField].source_type}</span>
                  </div>
                  <div className="p-2 bg-slate-900/60 rounded border border-slate-800/50">
                    <span className="text-[9px] text-slate-500 uppercase font-bold block">LOCATION</span>
                    <span className="text-xs font-semibold text-slate-300">{companyPassport[selectedField].source_location || 'N/A'}</span>
                  </div>
                </div>

                <div className="text-[10px] text-slate-500 mt-2 flex flex-col gap-1 border-t border-slate-800/50 pt-2 font-mono">
                  <div>URI: {companyPassport[selectedField].source_uri || 'N/A'}</div>
                  <div>Observed: {new Date(companyPassport[selectedField].observed_at).toLocaleString()}</div>
                </div>
              </div>
            </div>
          )}
        </section>

        {/* Right Side - Search, Match, Verify & HITL (8 Cols) */}
        <section className="lg:col-span-8 flex flex-col gap-8">
          
          {/* RAG Search Box */}
          <div className="glass-card p-6 flex flex-col gap-4">
            <div className="flex items-center gap-3 px-3 py-2 bg-slate-900 border border-slate-800 rounded-lg focus-within:border-indigo-500 transition-colors">
              <Search className="w-5 h-5 text-indigo-400" />
              <input 
                type="text" 
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Tìm kiếm chính sách, ưu đãi khoa học công nghệ..."
                className="flex-1 bg-transparent border-none text-white text-sm focus:outline-none placeholder-slate-500"
              />
              <button 
                onClick={searchPolicies}
                className="px-4 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs transition-colors font-semibold"
              >
                Search
              </button>
            </div>

            {/* Match List */}
            <div className="flex flex-col gap-3">
              <div className="flex justify-between items-center text-xs text-slate-400 font-semibold px-1">
                <span>RANKED POLICY OPPORTUNITIES</span>
                <span>MATCH SCORE</span>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {policies.map(p => {
                  const isSelected = selectedPolicyId === p.opportunity_id;
                  
                  return (
                    <div 
                      key={p.opportunity_id}
                      onClick={() => setSelectedPolicyId(p.opportunity_id)}
                      className={`p-4 rounded-lg border text-left cursor-pointer transition-all flex flex-col justify-between relative ${
                        isSelected 
                          ? 'bg-indigo-600/10 border-indigo-500/70 shadow-inner' 
                          : 'bg-slate-900/50 border-slate-800 hover:border-slate-700'
                      }`}
                    >
                      {/* Changed alert badge */}
                      {p.hasChanged && (
                        <span 
                          onClick={(e) => {
                            e.stopPropagation();
                            setDiffPolicy(p);
                            setShowDiffModal(true);
                          }}
                          className="absolute -top-2.5 -right-2 px-2 py-0.5 text-[9px] font-bold bg-amber-500 text-slate-950 border border-amber-300 rounded shadow-md cursor-pointer animate-bounce flex items-center gap-1"
                        >
                          <AlertTriangle className="w-2.5 h-2.5" /> Có Thay Đổi
                        </span>
                      )}

                      <div>
                        <h4 className="text-sm font-semibold m-0 text-white line-clamp-1">{p.title}</h4>
                        <p className="text-xs text-slate-400 mt-1 line-clamp-2">{p.benefits}</p>
                      </div>
                      
                      <div className="mt-3 flex justify-between items-center pt-2 border-t border-slate-800/40">
                        {p.deadline ? (
                          <span className="text-[10px] text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded border border-amber-500/10">
                            Hạn: {p.deadline}
                          </span>
                        ) : (
                          <span className="text-[10px] text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded border border-emerald-500/10">
                            Mở liên tục
                          </span>
                        )}
                        <span className="text-xs font-mono font-bold text-indigo-400">{(p.score * 100).toFixed(1)}%</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>

          {/* Eligibility & CITATION Viewer */}
          {selectedPolicy && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
              
              {/* Eligibility List */}
              <div className="glass-card p-6 flex flex-col gap-4 text-left">
                <div className="border-b border-slate-800 pb-2">
                  <h3 className="text-sm font-bold font-heading uppercase tracking-wide text-slate-300">
                    Eligibility Check: {selectedPolicy.title}
                  </h3>
                </div>

                {eligibility ? (
                  <div className="flex flex-col gap-4">
                    {/* Overall Badge */}
                    <div className="flex items-center gap-3 p-3 rounded bg-slate-900/60 border border-slate-800">
                      <span className="text-xs text-slate-400">Kết quả tổng hợp:</span>
                      <span className={`text-xs px-2.5 py-1 rounded-full font-bold uppercase ${getStatusBadgeClass(eligibility.status)}`}>
                        {eligibility.status}
                      </span>
                    </div>

                    {/* Criteria items */}
                    <div className="flex flex-col gap-2">
                      {eligibility.details.rules.map((rule: any) => {
                        const isSelected = selectedRuleId === rule.rule_id;
                        return (
                          <div 
                            key={rule.rule_id}
                            onClick={() => {
                              setSelectedRuleId(rule.rule_id);
                              setIsDrawerOpen(true); // Open sliding drawer on click
                            }}
                            className={`p-3 rounded border cursor-pointer transition-all flex items-center justify-between ${
                              isSelected 
                                ? 'bg-indigo-600/5 border-indigo-500/50' 
                                : 'bg-slate-900/30 border-slate-800 hover:border-slate-700'
                            }`}
                          >
                            <div className="flex items-start gap-3">
                              <div className="mt-0.5">{getStatusIcon(rule.status)}</div>
                              <div>
                                <span className="text-xs font-semibold text-slate-200">{rule.description}</span>
                                <span className="block text-[9px] text-slate-500 font-mono mt-0.5">ID: {rule.rule_id}</span>
                              </div>
                            </div>
                            <ArrowRight className="w-3.5 h-3.5 text-slate-500" />
                          </div>
                        );
                      })}
                    </div>
                  </div>
                ) : (
                  <div className="py-8 text-center text-slate-500 text-sm">Evaluating criteria...</div>
                )}
              </div>

              {/* Interactive Document Checklist (Option B) */}
              <div className="glass-card p-6 flex flex-col gap-4 text-left">
                <div className="border-b border-slate-800 pb-2 flex justify-between items-center">
                  <h3 className="text-sm font-bold font-heading uppercase tracking-wide text-slate-300">
                    Required Documents Checklist
                  </h3>
                  <span className="text-[9px] text-indigo-400 font-mono">MATCH & WAIVE</span>
                </div>

                <div className="flex flex-col gap-3">
                  {selectedPolicy.required_documents.map((doc: string, idx: number) => {
                    const status = checkedDocs[doc] || 'MISSING';
                    const isWaived = waivedDocs[doc] || false;
                    
                    return (
                      <div 
                        key={idx} 
                        className={`p-3 rounded border transition-all flex items-start justify-between gap-3 ${
                          isWaived 
                            ? 'bg-slate-900/30 border-slate-800/50 opacity-60' 
                            : status === 'MATCHED'
                            ? 'bg-emerald-500/5 border-emerald-500/20'
                            : status === 'UNDER_REVIEW'
                            ? 'bg-amber-500/5 border-amber-500/20'
                            : 'bg-rose-500/5 border-rose-500/20'
                        }`}
                      >
                        <div className="flex items-start gap-2.5">
                          {/* Checkbox item */}
                          <input 
                            type="checkbox"
                            checked={status === 'MATCHED' || isWaived}
                            onChange={() => {
                              if (isWaived) return;
                              setCheckedDocs(prev => ({
                                ...prev,
                                [doc]: prev[doc] === 'MATCHED' ? 'MISSING' : 'MATCHED'
                              }));
                            }}
                            className="custom-checkbox mt-0.5 flex-shrink-0"
                            disabled={isWaived}
                          />

                          <div className="text-xs">
                            <span className={`font-medium ${isWaived ? 'line-through text-slate-500' : 'text-slate-200'}`}>
                              {doc}
                            </span>
                            
                            {/* Subtext description / status tag */}
                            <div className="mt-1 flex items-center gap-2">
                              {isWaived ? (
                                <span className="text-[9px] bg-slate-800 text-slate-400 px-1.5 py-0.2 rounded font-mono">
                                  WAIVED (MIỄN NỘP)
                                </span>
                              ) : (
                                <span className={`text-[9px] font-mono px-1.5 py-0.2 rounded ${
                                  status === 'MATCHED' 
                                    ? 'bg-emerald-500/20 text-emerald-400' 
                                    : status === 'UNDER_REVIEW'
                                    ? 'bg-amber-500/20 text-amber-300'
                                    : 'bg-rose-500/20 text-rose-400'
                                }`}>
                                  {status === 'MATCHED' ? 'ĐÃ ĐỐI SÁNH' : status === 'UNDER_REVIEW' ? 'ĐANG PHÂN TÍCH' : 'THIẾU HỒ SƠ'}
                                </span>
                              )}
                            </div>
                          </div>
                        </div>

                        {/* Action buttons (Waive / Attach) */}
                        <div className="flex items-center gap-1 flex-shrink-0">
                          <button 
                            onClick={() => {
                              setWaivedDocs(prev => ({ ...prev, [doc]: !prev[doc] }));
                            }}
                            className="px-2 py-0.5 text-[9px] bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 rounded transition-colors"
                          >
                            {isWaived ? "Khôi phục" : "Bỏ qua"}
                          </button>
                          {status === 'MISSING' && !isWaived && (
                            <button 
                              onClick={() => {
                                setUploadedFileName(doc.toLowerCase().replace(/\s+/g, '_') + '.pdf');
                                setUploadStep(0);
                                setExtractedPreviewFields([]);
                                setShowUploadModal(true);
                              }}
                              className="px-2 py-0.5 text-[9px] bg-indigo-600 hover:bg-indigo-500 text-white rounded font-bold transition-colors"
                            >
                              Nộp
                            </button>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          )}

          {/* HITL Review Console (Staging Area) */}
          {selectedPolicy && eligibility && (
            <div className="glass-card p-6 flex flex-col gap-4 text-left">
              <div className="border-b border-slate-800 pb-2 flex justify-between items-center">
                <h3 className="text-sm font-bold font-heading uppercase tracking-wide text-slate-300">
                  Human-in-the-Loop Review Console (Staging)
                </h3>
                <span className="text-[10px] text-amber-400 font-mono">HITL ENFORCEMENT</span>
              </div>

              {!draftId ? (
                <div className="flex justify-between items-center">
                  <p className="text-xs text-slate-400">Khởi tạo hồ sơ nháp để tiến hành duyệt, hiệu chỉnh và xuất đơn đăng ký.</p>
                  <button 
                    onClick={handleCreateDraft}
                    className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-bold rounded transition-colors glowing-btn"
                  >
                    Tạo Hồ Sơ Nháp (Draft)
                  </button>
                </div>
              ) : (
                <div className="flex flex-col gap-4">
                  {/* Status Banner */}
                  <div className="flex justify-between items-center p-3 bg-slate-900 rounded border border-slate-800">
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-slate-400">ID Hồ sơ:</span>
                      <span className="text-xs font-mono text-slate-300">{draftId.slice(0, 8)}...</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-slate-400">Trạng thái:</span>
                      <span className={`text-[10px] px-2 py-0.5 rounded font-mono ${
                        draftStatus === 'GENERATED' 
                          ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/10' 
                          : draftStatus === 'REJECTED'
                          ? 'bg-rose-500/20 text-rose-400 border border-rose-500/10'
                          : 'bg-indigo-500/20 text-indigo-400 border border-indigo-500/10'
                      }`}>
                        {draftStatus}
                      </span>
                    </div>
                  </div>

                  {draftStatus === 'PENDING_REVIEW' && (
                    <div className="flex flex-col gap-3">
                      <div>
                        <label className="text-[10px] font-bold text-slate-500 block uppercase mb-1">Ý kiến người phê duyệt (Reviewer Comments):</label>
                        <textarea 
                          value={reviewerComments}
                          onChange={(e) => setReviewerComments(e.target.value)}
                          placeholder="Nhập bình luận hoặc ghi chú điều chỉnh hồ sơ nháp..."
                          className="w-full px-3 py-2 bg-slate-900 border border-slate-800 rounded text-xs text-white focus:outline-none focus:border-indigo-500 h-16 transition-colors"
                        />
                      </div>
                      
                      <div className="flex justify-end gap-3">
                        <button 
                          onClick={() => updateDraftStatus('REJECTED')}
                          className="flex items-center gap-2 px-3 py-1.5 text-xs bg-rose-600/10 hover:bg-rose-600/20 border border-rose-500/30 text-rose-400 rounded transition-colors"
                        >
                          <ThumbsDown className="w-3.5 h-3.5" />
                          Từ Chối (Reject)
                        </button>
                        <button 
                          onClick={() => updateDraftStatus('APPROVED')}
                          className="flex items-center gap-2 px-4 py-1.5 text-xs bg-indigo-600 hover:bg-indigo-500 text-white rounded transition-colors glowing-btn font-semibold"
                        >
                          <ThumbsUp className="w-3.5 h-3.5" />
                          Phê Duyệt & Xuất Đơn (Approve)
                        </button>
                      </div>
                    </div>
                  )}

                  {draftStatus === 'REJECTED' && (
                    <div className="p-3 bg-rose-500/5 border border-rose-500/20 rounded flex items-start gap-3">
                      <AlertCircle className="w-5 h-5 text-rose-500 flex-shrink-0 mt-0.5" />
                      <div>
                        <span className="font-bold text-rose-400 text-xs">Hồ sơ đã bị từ chối phê duyệt</span>
                        <p className="text-[11px] text-slate-300 mt-1">Lý do: {reviewerComments || 'Không có ghi chú'}</p>
                        <button 
                          onClick={handleCreateDraft}
                          className="mt-3 px-3 py-1 bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 text-[10px] rounded transition-colors"
                        >
                          Tạo lại Hồ Sơ Nháp mới
                        </button>
                      </div>
                    </div>
                  )}

                  {draftStatus === 'GENERATED' && (
                    <div className="p-4 bg-emerald-500/5 border border-emerald-500/20 rounded flex items-center justify-between">
                      <div className="flex items-start gap-3">
                        <CheckCircle2 className="w-6 h-6 text-emerald-500 mt-0.5" />
                        <div>
                          <span className="font-bold text-emerald-400 text-sm">Hồ sơ đã được phê duyệt & điền mẫu hoàn tất!</span>
                          <p className="text-xs text-slate-300 mt-1">Đơn đăng ký .docx đã được tạo tự động với dữ liệu được kiểm chứng.</p>
                          <span className="text-[9px] text-slate-500 block mt-2">* Prototype Disclaimer: Tải xuống trực tiếp không yêu cầu đăng nhập.</span>
                        </div>
                      </div>
                      
                      <a 
                        href={`${API_BASE}/drafts/${draftId}/download`}
                        className="flex items-center gap-2 px-4 py-2.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded font-bold text-xs transition-colors glowing-btn"
                      >
                        <Download className="w-4 h-4" />
                        Tải Đơn (.docx)
                      </a>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </section>
      </main>

      {/* Sliding Side-Drawer for Evidence Review (Option B) */}
      <div 
        className={`fixed inset-0 z-40 bg-black/60 backdrop-blur-sm transition-opacity duration-300 ${
          isDrawerOpen ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none'
        }`}
        onClick={() => setIsDrawerOpen(false)}
      />

      <div className={`fixed inset-y-0 right-0 z-50 w-full max-w-lg bg-slate-900 border-l border-slate-800 shadow-2xl p-6 overflow-y-auto drawer-transition transform ${
        isDrawerOpen ? 'translate-x-0' : 'translate-x-full'
      }`}>
        {activeRuleDetail ? (
          <div className="flex flex-col gap-6 text-left">
            {/* Header */}
            <div className="flex justify-between items-center border-b border-slate-800 pb-4">
              <div>
                <span className="text-[10px] text-indigo-400 font-mono tracking-wider uppercase font-bold">Evidence Reviewer</span>
                <h3 className="text-base font-bold text-white mt-1">{activeRuleDetail.description}</h3>
              </div>
              <button 
                onClick={() => setIsDrawerOpen(false)}
                className="p-1.5 hover:bg-slate-800 rounded text-slate-400 hover:text-white transition-colors"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Rule assessment Status */}
            <div className={`p-4 rounded border flex items-start gap-3 ${
              activeRuleDetail.status === 'MET' 
                ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400' 
                : activeRuleDetail.status === 'NOT_MET'
                ? 'bg-rose-500/10 border-rose-500/20 text-rose-400'
                : 'bg-amber-500/10 border-amber-500/20 text-amber-400'
            }`}>
              <div className="mt-0.5">{getStatusIcon(activeRuleDetail.status)}</div>
              <div>
                <span className="font-bold text-xs uppercase tracking-wider block">Tiêu chí: {activeRuleDetail.status}</span>
                <p className="text-xs text-slate-300 mt-1 leading-relaxed">{activeRuleDetail.reason}</p>
              </div>
            </div>

            {/* Side-by-Side Comparison Container */}
            <div className="flex flex-col gap-4">
              {/* Company Evidence */}
              <div className="p-4 bg-slate-950 border border-slate-800 rounded-lg flex flex-col gap-2">
                <div className="flex justify-between items-center text-[10px] text-indigo-400 uppercase tracking-wide font-bold">
                  <span>Hồ sơ Doanh nghiệp (Evidence)</span>
                  <span className="font-mono bg-indigo-500/10 px-2 py-0.5 rounded">Trường: {activeRuleDetail.field}</span>
                </div>
                <div className="text-sm font-bold text-white">
                  Giá trị thực tế: {activeRuleDetail.actual_value !== undefined 
                    ? (activeRuleDetail.field === 'rd_spend_ratio' 
                      ? `${(activeRuleDetail.actual_value * 100).toFixed(1)}%` 
                      : typeof activeRuleDetail.actual_value === 'number' 
                      ? `${activeRuleDetail.actual_value.toLocaleString()} VND` 
                      : activeRuleDetail.actual_value) 
                    : 'N/A'}
                </div>
                <blockquote className="pl-3 border-l-2 border-indigo-500 italic text-slate-300 text-xs py-1 bg-indigo-950/20 rounded">
                  "{activeRuleDetail.evidence_quote || 'Không có bằng chứng được trích dẫn'}"
                </blockquote>
                <div className="text-[10px] text-slate-500 mt-1 font-mono">
                  Nguồn: {activeRuleDetail.source_uri} ({activeRuleDetail.source_location || 'N/A'})
                </div>
              </div>

              {/* Policy Clause / Requirements */}
              <div className="p-4 bg-slate-950 border border-slate-800 rounded-lg flex flex-col gap-2">
                <div className="flex justify-between items-center text-[10px] text-emerald-400 uppercase tracking-wide font-bold">
                  <span>Điều khoản Pháp lý (Policy Clause)</span>
                  <span className="font-mono bg-emerald-500/10 px-2 py-0.5 rounded">Phép so sánh: {activeRuleDetail.operator}</span>
                </div>
                <div className="text-sm font-bold text-white">
                  Điều kiện yêu cầu: {activeRuleDetail.operator} {activeRuleDetail.expected_value}
                </div>
                {activeRuleDetail.citation ? (
                  <>
                    <blockquote className="pl-3 border-l-2 border-emerald-500 italic text-slate-300 text-xs py-1 bg-emerald-950/20 rounded">
                      "{activeRuleDetail.citation.quote}"
                    </blockquote>
                    <div className="text-[10px] text-slate-400 mt-2 flex justify-between items-center border-t border-slate-900 pt-2 font-mono">
                      <span>Văn bản: {activeRuleDetail.citation.document_id} ({activeRuleDetail.citation.article})</span>
                      <a 
                        href={activeRuleDetail.citation.source_url || '#'} 
                        target="_blank" 
                        rel="noreferrer"
                        className="flex items-center gap-1 text-[9px] text-indigo-400 hover:underline"
                      >
                        Xem văn bản gốc <ExternalLink className="w-2.5 h-2.5" />
                      </a>
                    </div>
                  </>
                ) : (
                  <p className="text-slate-500 italic text-xs">Không có trích dẫn văn bản pháp lý</p>
                )}
              </div>
            </div>

            {/* Clarification Loop for MISSING_INFO */}
            {activeRuleDetail.status === 'MISSING_INFO' && (
              <div className="p-4 bg-amber-500/5 border border-amber-500/20 rounded-lg flex flex-col gap-3">
                <div className="flex items-center gap-2 text-amber-400 font-semibold text-xs uppercase tracking-wider">
                  <HelpCircle className="w-4 h-4" />
                  <span>Hỏi làm rõ thông tin:</span>
                </div>
                <p className="text-xs text-slate-300 leading-relaxed">{activeRuleDetail.clarification_question}</p>
                
                <div className="flex items-center gap-2">
                  <input 
                    type="text"
                    value={clarificationAnswer}
                    onChange={(e) => setClarificationAnswer(e.target.value)}
                    placeholder="Nhập câu trả lời làm rõ..."
                    className="flex-1 px-3 py-1.5 bg-slate-950 border border-slate-800 rounded text-xs text-white focus:outline-none focus:border-indigo-500 transition-colors"
                  />
                  <button 
                    onClick={() => {
                      submitClarification(activeRuleDetail.field);
                      setIsDrawerOpen(false); // Close drawer after submit
                    }}
                    className="p-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded transition-colors shadow-md"
                  >
                    <Send className="w-4 h-4" />
                  </button>
                </div>
              </div>
            )}
          </div>
        ) : (
          <div className="py-8 text-center text-slate-500 text-sm">Vui lòng chọn tiêu chí để xem đối chiếu chi tiết</div>
        )}
      </div>

      {/* Edit Field Modal (Manual Input) */}
      {editingField && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="glass-card p-6 w-full max-w-md flex flex-col gap-4 text-left animate-float-up">
            <div className="flex justify-between items-center border-b border-slate-800 pb-2">
              <h3 className="text-sm font-bold uppercase tracking-wider text-white">Hiệu chỉnh: {formatFieldName(editingField)}</h3>
              <button onClick={() => setEditingField(null)} className="p-1 hover:bg-slate-800 rounded">
                <X className="w-4 h-4 text-slate-400 hover:text-white" />
              </button>
            </div>
            
            <div className="flex flex-col gap-2">
              <label className="text-xs text-slate-400">Giá trị thực tế mới:</label>
              <input 
                type="text" 
                value={editValue}
                onChange={(e) => setEditValue(e.target.value)}
                className="w-full px-3 py-2 bg-slate-900 border border-slate-800 rounded text-sm text-white focus:outline-none focus:border-indigo-500 transition-colors"
              />
              <p className="text-[10px] text-amber-400 mt-1 leading-normal">
                * Hành động này sẽ thay đổi trạng thái của trường thành 'USER_CONFIRMED' và ghi lại lịch sử chỉnh sửa vào Audit Log.
              </p>
            </div>
            
            <div className="flex justify-end gap-3 mt-4">
              <button 
                onClick={() => setEditingField(null)}
                className="px-3.5 py-1.5 text-xs bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 rounded transition-colors"
              >
                Hủy
              </button>
              <button 
                onClick={() => saveFieldUpdate(editingField)}
                className="px-4 py-1.5 text-xs bg-indigo-600 hover:bg-indigo-500 text-white rounded transition-colors glowing-btn font-semibold"
              >
                Lưu Thay Đổi
              </button>
            </div>
          </div>
        </div>
      )}

      {/* High Fidelity Document Upload & AI Extraction Simulator Modal */}
      {showUploadModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-md">
          <div className="glass-card p-6 w-full max-w-xl flex flex-col gap-5 text-left animate-float-up">
            <div className="flex justify-between items-center border-b border-slate-800 pb-2">
              <h3 className="text-sm font-bold uppercase tracking-wider text-white flex items-center gap-2">
                <UploadCloud className="w-4 h-4 text-indigo-400" />
                AI Vision Document Extractor
              </h3>
              <button 
                onClick={() => setShowUploadModal(false)}
                disabled={uploadStep > 0 && uploadStep < 5}
                className="p-1 hover:bg-slate-800 rounded disabled:opacity-30"
              >
                <X className="w-4 h-4 text-slate-400 hover:text-white" />
              </button>
            </div>

            {/* Stage 0: Choose file or website */}
            {uploadStep === 0 && (
              <div className="flex flex-col gap-4">
                <div className="border-2 border-dashed border-slate-800 hover:border-indigo-500 rounded-xl p-8 text-center flex flex-col items-center gap-3 transition-colors bg-slate-900/30 cursor-pointer">
                  <UploadCloud className="w-10 h-10 text-indigo-400 animate-pulse" />
                  <div>
                    <span className="text-sm font-bold text-slate-200">Kéo thả tài liệu doanh nghiệp vào đây</span>
                    <p className="text-xs text-slate-500 mt-1">Hỗ trợ PDF, DOCX, PNG, JPG (Giấy ĐKKD, Báo cáo tài chính, Điều lệ công ty...)</p>
                  </div>
                  <input 
                    type="file" 
                    onChange={(e) => {
                      if (e.target.files && e.target.files.length > 0) {
                        setUploadedFileName(e.target.files[0].name);
                      }
                    }}
                    className="hidden" 
                    id="doc-file-input"
                  />
                  <label 
                    htmlFor="doc-file-input" 
                    className="mt-2 px-4 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs border border-slate-700 rounded-md font-bold cursor-pointer transition-colors"
                  >
                    Chọn Tệp
                  </label>
                  {uploadedFileName && (
                    <span className="text-xs font-mono text-emerald-400 mt-2 bg-emerald-500/10 px-2 py-0.5 rounded border border-emerald-500/20">
                      Tệp đã chọn: {uploadedFileName}
                    </span>
                  )}
                </div>

                <div className="flex flex-col gap-2">
                  <label className="text-xs text-slate-400 font-bold">Hoặc nhập liên kết cổng thông tin / website:</label>
                  <div className="flex gap-2">
                    <input 
                      type="text" 
                      placeholder="https://aitechvn.com" 
                      onChange={(e) => {
                        if (e.target.value) {
                          setUploadedFileName(e.target.value.replace(/https?:\/\//, ''));
                        }
                      }}
                      className="flex-1 px-3 py-2 bg-slate-900 border border-slate-800 rounded text-xs text-white focus:outline-none focus:border-indigo-500 transition-colors"
                    />
                  </div>
                </div>

                <button 
                  onClick={startExtraction}
                  disabled={!uploadedFileName}
                  className="mt-2 w-full py-2.5 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white rounded text-xs font-bold transition-colors glowing-btn flex items-center justify-center gap-1.5"
                >
                  Bắt đầu Trích xuất AI (Gemini Vision) <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            )}

            {/* Stages 1 to 5: Running simulated extraction with scanner line */}
            {uploadStep > 0 && (
              <div className="grid grid-cols-1 md:grid-cols-12 gap-6">
                
                {/* Stepper Status Logs (Left side) */}
                <div className="md:col-span-7 flex flex-col gap-3">
                  <span className="text-[10px] font-bold text-slate-500 uppercase tracking-wide">Trạng thái trích xuất AI:</span>
                  
                  {/* Console logs */}
                  <div className="p-3 bg-slate-950 border border-slate-900 rounded-lg font-mono text-[11px] text-indigo-300 h-64 overflow-y-auto flex flex-col gap-1.5">
                    {extractionLogs.map((log, idx) => (
                      <div key={idx} className="flex items-start gap-2">
                        <span className="text-slate-600">[{idx+1}]</span>
                        <span className={log.includes('XUNG ĐỘT') ? 'text-rose-400 font-bold' : log.includes('Đã trích xuất') ? 'text-emerald-400' : ''}>
                          {log}
                        </span>
                      </div>
                    ))}
                    {uploadStep < 5 && (
                      <div className="flex items-center gap-1.5 text-indigo-400 italic animate-pulse mt-1">
                        <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                        <span>Đang phân tích cú pháp...</span>
                      </div>
                    )}
                  </div>

                  {/* Render extracted fields to prevent TS error and enhance visual design */}
                  {extractedPreviewFields.length > 0 && (
                    <div className="mt-3 bg-slate-950 border border-slate-900 rounded-lg p-3 animate-float-up">
                      <span className="text-[10px] text-emerald-400 font-bold uppercase block mb-2 font-mono">Thông tin trích xuất:</span>
                      <div className="flex flex-col gap-1.5">
                        {extractedPreviewFields.map((f, idx) => (
                          <div key={idx} className="flex justify-between items-center text-xs border-b border-slate-900/50 pb-1 last:border-0 last:pb-0 font-sans">
                            <span className="text-slate-400">{formatFieldName(f.name)}:</span>
                            <span className="font-semibold text-white">{f.value} <span className="text-[10px] text-emerald-500 font-mono">({f.confidence}%)</span></span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>

                {/* Simulated Doc & Scanner Laser line (Right side) */}
                <div className="md:col-span-5 flex flex-col gap-3">
                  <span className="text-[10px] font-bold text-slate-500 uppercase tracking-wide">Mô phỏng quét tài liệu (OCR):</span>
                  
                  <div className="relative border border-slate-800 rounded-lg h-64 bg-slate-900/60 overflow-hidden scanner-container flex items-center justify-center p-4">
                    {/* The Laser Scanner line */}
                    {uploadStep === 2 || uploadStep === 3 ? (
                      <div className="scanner-line" />
                    ) : null}

                    {/* Mock Doc Content representation */}
                    <div className="w-full h-full border border-slate-800/40 bg-slate-900 rounded p-3 flex flex-col gap-2 font-mono text-[9px] text-slate-500 overflow-hidden relative">
                      <div className="border-b border-slate-800 pb-1 text-center font-bold text-slate-400">
                        CỘNG HÒA XÃ HỘI CHỦ NGHĨA VIỆT NAM
                      </div>
                      <div className="text-[8px] text-center italic text-slate-600">Độc lập - Tự do - Hạnh phúc</div>
                      <div className="mt-2 text-[10px] font-bold text-slate-300 text-center">GIẤY CHỨNG NHẬN ĐĂNG KÝ DOANH NGHIỆP</div>
                      <div className="mt-1 border-t border-slate-850 pt-1 flex flex-col gap-1.5">
                        <div className="flex justify-between border-b border-slate-850/50 pb-0.5">
                          <span>Tên Công ty:</span>
                          <span className="text-slate-400">AITECH VIETNAM LLC</span>
                        </div>
                        <div className="flex justify-between border-b border-slate-850/50 pb-0.5">
                          <span>Mã số thuế:</span>
                          <span className="text-slate-400">0109988776</span>
                        </div>
                        <div className="flex justify-between border-b border-slate-850/50 pb-0.5">
                          <span>Vốn điều lệ:</span>
                          <span className={uploadStep >= 4 ? "text-amber-400 font-bold" : "text-slate-400"}>
                            12.000.000.000 VNĐ
                          </span>
                        </div>
                        <div className="flex justify-between border-b border-slate-850/50 pb-0.5">
                          <span>Số lao động:</span>
                          <span className={uploadStep >= 4 ? "text-emerald-400 font-bold" : "text-slate-400"}>
                            15 người
                          </span>
                        </div>
                        <div className="flex justify-between border-b border-slate-850/50 pb-0.5">
                          <span>Chi R&D:</span>
                          <span className={uploadStep >= 4 ? "text-emerald-400 font-bold" : "text-slate-400"}>
                            2.1% doanh thu
                          </span>
                        </div>
                      </div>

                      {/* Scanning laser glow on background doc */}
                      {(uploadStep === 2 || uploadStep === 3) && (
                        <div className="absolute inset-0 bg-indigo-500/5 pointer-events-none" />
                      )}
                    </div>
                  </div>
                </div>

              </div>
            )}

            {/* Footer controls */}
            {uploadStep === 5 && (
              <div className="flex justify-between items-center border-t border-slate-800 pt-4 mt-2">
                <span className="text-rose-400 text-xs flex items-center gap-1 font-bold">
                  <AlertTriangle className="w-4 h-4 animate-pulse" /> Phát hiện 1 trường dữ liệu xung đột!
                </span>
                
                <div className="flex gap-3">
                  <button 
                    onClick={() => {
                      setShowUploadModal(false);
                      setUploadStep(0);
                      // Force refresh dashboard data
                      fetchEligibility();
                      searchPolicies();
                    }}
                    className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-bold rounded transition-colors"
                  >
                    Đồng bộ & Đóng (Close)
                  </button>
                </div>
              </div>
            )}

          </div>
        </div>
      )}

      {/* Side-by-Side Conflict Resolution Modal (Option B) */}
      {showConflictModal && conflictFieldName && conflictFieldData && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-md">
          <div className="glass-card p-6 w-full max-w-2xl flex flex-col gap-4 text-left animate-float-up">
            <div className="flex justify-between items-center border-b border-slate-800 pb-2">
              <h3 className="text-sm font-bold uppercase tracking-wider text-rose-400 flex items-center gap-1.5">
                <AlertTriangle className="w-4 h-4 animate-pulse" />
                Đối Chiếu Xung Đột: {formatFieldName(conflictFieldName)}
              </h3>
              <button onClick={() => setShowConflictModal(false)} className="p-1 hover:bg-slate-800 rounded">
                <X className="w-4 h-4 text-slate-400 hover:text-white" />
              </button>
            </div>

            <p className="text-xs text-slate-400">
              Phát hiện mâu thuẫn về giá trị của trường <span className="font-bold text-slate-200 font-mono">'{conflictFieldName}'</span> trích xuất từ các tài liệu khác nhau. Vui lòng phê duyệt một nguồn dữ liệu để cập nhật Company Passport.
            </p>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 my-2">
              {/* Option A: Current Database Value */}
              <div className="p-4 bg-slate-900 border border-slate-800 rounded-lg flex flex-col justify-between gap-3">
                <div>
                  <span className="text-[10px] text-slate-500 font-bold uppercase tracking-wider block">Nguồn A (Cơ sở dữ liệu cũ)</span>
                  <div className="text-base font-bold text-slate-200 mt-1">
                    {typeof conflictFieldData.conflicts?.[0]?.value === 'number' 
                      ? `${conflictFieldData.conflicts[0].value.toLocaleString()} VND` 
                      : conflictFieldData.conflicts?.[0]?.value || '10,000,000,000 VND'}
                  </div>
                  <blockquote className="pl-2.5 border-l-2 border-slate-500 italic text-slate-400 text-[11px] mt-2 bg-slate-950/40 py-1">
                    "{conflictFieldData.conflicts?.[0]?.evidence_quote || 'Vốn điều lệ mười tỷ đồng đăng ký ban đầu.'}"
                  </blockquote>
                </div>

                <button 
                  onClick={() => executeConflictResolution(
                    conflictFieldData.conflicts?.[0]?.value || 10000000000,
                    'Original Passport Database',
                    conflictFieldData.conflicts?.[0]?.evidence_quote || 'Vốn điều lệ mười tỷ đồng đăng ký ban đầu.'
                  )}
                  className="w-full py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 rounded text-xs font-bold transition-colors"
                >
                  Sử dụng Nguồn A
                </button>
              </div>

              {/* Option B: Newly Extracted Value */}
              <div className="p-4 bg-indigo-950/20 border border-indigo-500/20 rounded-lg flex flex-col justify-between gap-3">
                <div>
                  <span className="text-[10px] text-indigo-400 font-bold uppercase tracking-wider block">Nguồn B (Mới Trích Xuất AI)</span>
                  <div className="text-base font-bold text-indigo-300 mt-1">
                    {typeof conflictFieldData.value === 'number' 
                      ? `${conflictFieldData.value.toLocaleString()} VND` 
                      : conflictFieldData.value}
                  </div>
                  <blockquote className="pl-2.5 border-l-2 border-indigo-500 italic text-indigo-200 text-[11px] mt-2 bg-indigo-950/40 py-1">
                    "{conflictFieldData.evidence_quote}"
                  </blockquote>
                  <div className="text-[9px] text-indigo-400/60 mt-1 font-mono">
                    Tài liệu: {conflictFieldData.source_uri} ({conflictFieldData.source_location})
                  </div>
                </div>

                <button 
                  onClick={() => executeConflictResolution(
                    conflictFieldData.value,
                    conflictFieldData.source_uri,
                    conflictFieldData.evidence_quote
                  )}
                  className="w-full py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-bold transition-colors shadow-md"
                >
                  Sử dụng Nguồn B
                </button>
              </div>
            </div>

            {/* Custom Override Form */}
            <div className="border-t border-slate-800 pt-3 flex flex-col gap-2">
              <label className="text-[10px] font-bold text-slate-400 uppercase tracking-wider">Hoặc nhập thủ công giá trị điều chỉnh khác:</label>
              <div className="flex gap-2">
                <input 
                  type="text" 
                  placeholder="Nhập giá trị mong muốn..."
                  onChange={(e) => setEditValue(e.target.value)}
                  className="flex-1 px-3 py-1.5 bg-slate-900 border border-slate-800 rounded text-xs text-white focus:outline-none focus:border-indigo-500 transition-colors"
                />
                <button 
                  onClick={() => {
                    let val: any = editValue;
                    if (!isNaN(Number(editValue)) && editValue.trim() !== '') {
                      val = Number(editValue);
                    }
                    executeConflictResolution(val, 'User Custom Override', 'Manual user override during conflict resolution.');
                  }}
                  className="px-4 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 rounded text-xs font-bold transition-colors"
                >
                  Sử dụng Ghi đè
                </button>
              </div>
            </div>

          </div>
        </div>
      )}

      {/* Policy Change Alerts Text Diff Modal (Option B) */}
      {showDiffModal && diffPolicy && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-md">
          <div className="glass-card p-6 w-full max-w-xl flex flex-col gap-4 text-left animate-float-up">
            <div className="flex justify-between items-center border-b border-slate-800 pb-2">
              <h3 className="text-sm font-bold uppercase tracking-wider text-amber-400 flex items-center gap-1.5">
                <AlertTriangle className="w-4 h-4" />
                So Sánh Cập Nhật Luật: {diffPolicy.title}
              </h3>
              <button onClick={() => setShowDiffModal(false)} className="p-1 hover:bg-slate-800 rounded">
                <X className="w-4 h-4 text-slate-400 hover:text-white" />
              </button>
            </div>

            <p className="text-xs text-slate-400">
              Phát hiện thay đổi trong điều khoản pháp lý quy định điều kiện của chính sách này. Dưới đây là đối chiếu chi tiết giữa phiên bản luật cũ và văn bản cập nhật mới:
            </p>

            <div className="flex flex-col gap-3 font-sans text-xs">
              {/* Old Clause */}
              <div className="p-3 bg-rose-950/20 border border-rose-500/10 rounded-lg">
                <span className="text-[9px] text-rose-400 font-mono uppercase font-bold block mb-1">Điều khoản luật Cũ (Nghị định hết hiệu lực)</span>
                <p className="text-slate-300 leading-relaxed italic">
                  "{diffPolicy.oldClause}"
                </p>
              </div>

              {/* New Clause with highlight diffs */}
              <div className="p-3 bg-emerald-950/20 border border-emerald-500/10 rounded-lg">
                <span className="text-[9px] text-emerald-400 font-mono uppercase font-bold block mb-1">Điều khoản luật Mới (Ban hành cập nhật)</span>
                <p className="text-slate-200 leading-relaxed font-semibold">
                  Doanh nghiệp có tỷ lệ chi cho nghiên cứu và phát triển (R&D) tối thiểu là{" "}
                  <span className="bg-rose-500/20 text-rose-300 line-through px-1 rounded font-mono">2.0%</span>{" "}
                  <span className="bg-emerald-500/30 text-emerald-300 font-bold px-1 rounded font-mono">1.5%</span>{" "}
                  trên tổng doanh thu trong 3 năm gần nhất.
                </p>
              </div>
            </div>

            <div className="border-t border-slate-800 pt-3 text-[10px] text-slate-500 leading-normal">
              * Hệ thống đã tự động chạy lại động cơ đối sánh. Score cập nhật của công ty bạn dựa trên điều kiện R&D 1.5% mới.
            </div>

            <div className="flex justify-end mt-2">
              <button 
                onClick={() => setShowDiffModal(false)}
                className="px-4 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-bold transition-colors"
              >
                Đồng ý & Cập nhật
              </button>
            </div>

          </div>
        </div>
      )}

      {/* Sync Scraper Modal */}
      {showSyncModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="glass-card p-6 w-full max-w-xl flex flex-col gap-4 text-left">
            <div className="flex justify-between items-center border-b border-slate-800 pb-2">
              <h3 className="text-sm font-bold uppercase tracking-wider text-white">Chạy Sync Đồng Bộ Chính Sách</h3>
              <button onClick={() => setShowSyncModal(false)} className="p-1 hover:bg-slate-800 rounded">
                <X className="w-4 h-4 text-slate-400 hover:text-white" />
              </button>
            </div>
            
            <div className="p-4 bg-slate-950 rounded border border-slate-900 font-mono text-xs text-indigo-400 h-64 overflow-y-auto flex flex-col gap-1.5">
              {syncLogs.map((log, idx) => (
                <div key={idx} className="flex items-start gap-2">
                  <span className="text-slate-500">[{idx+1}]</span>
                  <span>{log}</span>
                </div>
              ))}
              {syncing && (
                <div className="flex items-center gap-2 text-indigo-300 italic mt-1">
                  <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                  <span>Đang xử lý đồng bộ...</span>
                </div>
              )}
            </div>
            
            <div className="flex justify-end gap-3 mt-2">
              <button 
                onClick={() => setShowSyncModal(false)}
                disabled={syncing}
                className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-xs font-bold rounded transition-colors"
              >
                Đóng Console
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Audit Logs Modal */}
      {showAuditModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="glass-card p-6 w-full max-w-2xl flex flex-col gap-4 text-left">
            <div className="flex justify-between items-center border-b border-slate-800 pb-2">
              <h3 className="text-sm font-bold uppercase tracking-wider text-white">Audit Trail (Nhật ký Thay Đổi Dữ Liệu)</h3>
              <button onClick={() => setShowAuditModal(false)} className="p-1 hover:bg-slate-800 rounded">
                <X className="w-4 h-4 text-slate-400 hover:text-white" />
              </button>
            </div>
            
            <div className="overflow-y-auto max-h-96 flex flex-col gap-2.5">
              {auditLogs.length > 0 ? (
                auditLogs.map((log) => (
                  <div key={log.id} className="p-3 bg-slate-900 border border-slate-800 rounded text-xs flex flex-col gap-1.5 animate-float-up">
                    <div className="flex justify-between items-center">
                      <span className="text-indigo-400 font-bold uppercase text-[9px] bg-indigo-500/10 px-2 py-0.5 rounded">
                        {log.event_type}
                      </span>
                      <span className="text-[10px] text-slate-500">{new Date(log.timestamp).toLocaleString()}</span>
                    </div>
                    <div className="text-slate-300">
                      {log.event_type === 'PASSPORT_EDIT' ? (
                        <>
                          Người dùng đã điều chỉnh trường <span className="font-bold text-slate-200">{log.field_name}</span> của công ty <span className="font-mono text-slate-200">'{log.target_id}'</span>:
                          <div className="mt-1 flex items-center gap-2 p-1.5 bg-slate-950 rounded border border-slate-900 text-[11px] font-mono">
                            <span className="text-rose-400">-{log.old_value}</span>
                            <span className="text-slate-500">→</span>
                            <span className="text-emerald-400">+{log.new_value}</span>
                          </div>
                        </>
                      ) : (
                        <>
                          Hồ sơ nháp <span className="font-mono text-indigo-400">{log.target_id.slice(0, 8)}...</span> chuyển trạng thái:
                          <div className="mt-1 flex items-center gap-2 p-1.5 bg-slate-950 rounded border border-slate-900 text-[11px] font-mono">
                            <span className="text-slate-400">{log.old_value}</span>
                            <span className="text-slate-500">→</span>
                            <span className="text-indigo-400">{log.new_value}</span>
                          </div>
                        </>
                      )}
                    </div>
                  </div>
                ))
              ) : (
                <div className="py-8 text-center text-slate-500 text-sm">Không có dữ liệu thay đổi nào trong nhật ký.</div>
              )}
            </div>
            
            <div className="flex justify-end gap-3 mt-2 border-t border-slate-800 pt-3">
              <button 
                onClick={() => setShowAuditModal(false)}
                className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 text-xs font-bold rounded transition-colors"
              >
                Đóng Nhật Ký
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Footer */}
      <footer className="border-t border-slate-800 bg-slate-900 py-6 mt-12">
        <div className="mx-auto max-w-7xl px-6 flex flex-col md:flex-row justify-between items-center gap-4 text-xs text-slate-500">
          <span>&copy; 2026 P2B Platform. Built for NIC AI Hackathon. All rights reserved.</span>
          <div className="flex gap-4">
            <span className="hover:underline cursor-pointer">Terms of Service</span>
            <span className="hover:underline cursor-pointer">Privacy Policy</span>
            <span className="hover:underline cursor-pointer">Documentation</span>
          </div>
        </div>
      </footer>
    </div>
  );
}
