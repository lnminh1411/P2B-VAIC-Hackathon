import { useState, useEffect, useRef } from 'react';
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
  History, 
  X, 
  ExternalLink, 
  User, 
  LogOut, 
  Settings, 
  Trash2, 
  Camera, 
  UserCheck, 
  AlertTriangle
} from 'lucide-react';

import vi from './locales/vi.json';
import en from './locales/en.json';

const translations: Record<string, any> = { vi, en };
const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8000/api/v1';

export default function App() {
  // Localization & Theme states
  const [locale, setLocale] = useState(() => localStorage.getItem('p2b_locale') || 'vi');
  const [theme, setTheme] = useState(() => localStorage.getItem('p2b_theme') || 'light');

  const t = (path: string) => {
    const keys = path.split('.');
    let current = translations[locale] || translations['vi'];
    for (const key of keys) {
      if (current && typeof current === 'object' && key in current) {
        current = current[key];
      } else {
        return path;
      }
    }
    return typeof current === 'string' ? current : path;
  };

  const handleThemeChange = (newTheme: string) => {
    setTheme(newTheme);
    localStorage.setItem('p2b_theme', newTheme);
    if (newTheme === 'dark') {
      document.documentElement.classList.add('dark');
      document.body.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
      document.body.classList.remove('dark');
    }
  };

  const handleLanguageChange = (newLocale: string) => {
    setLocale(newLocale);
    localStorage.setItem('p2b_locale', newLocale);
  };

  useEffect(() => {
    const savedTheme = localStorage.getItem('p2b_theme') || 'light';
    if (savedTheme === 'dark') {
      document.documentElement.classList.add('dark');
      document.body.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
      document.body.classList.remove('dark');
    }
  }, []);

  // Auth state
  const [token, setToken] = useState<string | null>(localStorage.getItem('p2b_token'));
  const [user, setUser] = useState<any>(null);
  const [authEmail, setAuthEmail] = useState('');
  const [authPassword, setAuthPassword] = useState('');
  const [isRegistering, setIsRegistering] = useState(false);
  const [registerType, setRegisterType] = useState('COMPANY_MANAGER');
  const [authError, setAuthError] = useState<string | null>(null);

  // Profile Settings Dropdown State
  const [isProfileDropdownOpen, setIsProfileDropdownOpen] = useState(false);
  const [showSettingsModal, setShowSettingsModal] = useState(false);
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [pwdMessage, setPwdMessage] = useState<string | null>(null);
  const [pwdError, setPwdError] = useState<string | null>(null);

  // Selected Company Tenant (For Company Manager mode)
  const companies = ['AItech_Vietnam_LLC', 'FDI_SemiVina_Corp', 'SolarGreen_Tech_JSC'];
  const [selectedCompanyId, setSelectedCompanyId] = useState<string>('AItech_Vietnam_LLC');
  const [companyPassport, setCompanyPassport] = useState<any>(null);

  // Personal Passport State (For Individual mode)
  const [personalPassport, setPersonalPassport] = useState<any>({
    full_name: '',
    birth_year: 0,
    location: '',
    occupation: '',
    degree: '',
    monthly_income: 0
  });

  // RAG Search & Policies
  const [searchQuery, setSearchQuery] = useState<string>('chương trình nghiên cứu trí tuệ nhân tạo');
  const [policies, setPolicies] = useState<any[]>([]);
  const [selectedPolicyId, setSelectedPolicyId] = useState<string | null>(null);
  const [selectedPolicy, setSelectedPolicy] = useState<any>(null);
  
  // Policy Alerts
  const [policyAlerts, setPolicyAlerts] = useState<any[]>([]);

  // Eligibility & Drafts (HITL)
  const [eligibility, setEligibility] = useState<any>(null);
  const [selectedRuleId, setSelectedRuleId] = useState<string | null>(null);
  const [reviewerComments, setReviewerComments] = useState<string>('');
  const [draftId, setDraftId] = useState<string | null>(null);
  const [draftStatus, setDraftStatus] = useState<string | null>(null);
  const [draftError, setDraftError] = useState<string | null>(null);

  // Manual Passport Field Edits
  const [editingField, setEditingField] = useState<string | null>(null);
  const [editValue, setEditValue] = useState<string>('');
  const [selectedField, setSelectedField] = useState<string>('company_name');

  const handleEditField = (fieldName: string, currentValue: any) => {
    setEditingField(fieldName);
    setEditValue(String(currentValue));
  };

  // Modals
  const [showAuditModal, setShowAuditModal] = useState<boolean>(false);
  const [auditLogs, setAuditLogs] = useState<any[]>([]);
  const [showSyncModal, setShowSyncModal] = useState<boolean>(false);
  const [syncLogs, setSyncLogs] = useState<string[]>([]);
  const [syncing, setSyncing] = useState<boolean>(false);
  const [showUploadModal, setShowUploadModal] = useState<boolean>(false);
  const [uploadFiles, setUploadFiles] = useState<FileList | null>(null);
  const [extracting, setExtracting] = useState(false);

  // Conflict Resolution
  const [showConflictModal, setShowConflictModal] = useState<boolean>(false);
  const [conflictFieldName, setConflictFieldName] = useState<string | null>(null);
  const [conflictFieldData, setConflictFieldData] = useState<any>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const avatarInputRef = useRef<HTMLInputElement>(null);

  // Helper: headers with auth token
  const getHeaders = () => {
    return {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    };
  };

  // 1. Fetch current user context and active profile details
  useEffect(() => {
    if (token) {
      localStorage.setItem('p2b_token', token);
      fetchUserContext();
    } else {
      localStorage.removeItem('p2b_token');
      setUser(null);
    }
  }, [token]);

  // Refetch data when company tenant, search query, or user mode switches
  useEffect(() => {
    if (!token || !user) return;
    
    if (user.user_type === 'COMPANY_MANAGER') {
      fetchPassport(selectedCompanyId);
    } else {
      fetchPersonalPassport();
    }
    
    // Clear state
    setEligibility(null);
    setSelectedPolicyId(null);
    setDraftId(null);
    setDraftStatus(null);
    setDraftError(null);
  }, [selectedCompanyId, user?.user_type, token]);

  useEffect(() => {
    if (token && user) {
      searchPolicies();
      fetchPolicyAlerts();
    }
  }, [selectedCompanyId, searchQuery, user?.user_type, token]);

  useEffect(() => {
    setDraftId(null);
    setDraftStatus(null);
    setDraftError(null);
    setReviewerComments('');
    if (selectedPolicyId && token) {
      fetchEligibility();
    }
  }, [selectedPolicyId, selectedCompanyId, user?.user_type, token]);

  const fetchUserContext = async () => {
    try {
      const res = await fetch(`${API_BASE}/users/me`, {
        headers: getHeaders()
      });
      if (res.ok) {
        const data = await res.json();
        setUser(data.user);
        if (data.user.user_type === 'COMPANY_MANAGER' && data.user.company_id) {
          setSelectedCompanyId(data.user.company_id);
        }
      } else {
        setToken(null);
      }
    } catch (err) {
      setToken(null);
    }
  };

  const fetchPassport = async (companyId: string) => {
    try {
      const res = await fetch(`${API_BASE}/passports/${companyId}`, {
        headers: getHeaders()
      });
      if (res.ok) {
        const data = await res.json();
        setCompanyPassport(data.data);
      }
    } catch (err) {
      console.error("Error fetching passport:", err);
    }
  };

  const fetchPersonalPassport = async () => {
    try {
      const res = await fetch(`${API_BASE}/users/me`, {
        headers: getHeaders()
      });
      if (res.ok) {
        const data = await res.json();
        if (data.passport) {
          setPersonalPassport(data.passport);
        }
      }
    } catch (err) {
      console.error("Error fetching personal passport:", err);
    }
  };

  const fetchPolicyAlerts = async () => {
    try {
      const res = await fetch(`${API_BASE}/policy_alerts`, {
        headers: getHeaders()
      });
      if (res.ok) {
        const data = await res.json();
        setPolicyAlerts(data);
      }
    } catch (err) {
      console.error("Error fetching policy alerts:", err);
    }
  };

  const searchPolicies = async () => {
    try {
      const res = await fetch(`${API_BASE}/policies?query=${encodeURIComponent(searchQuery)}`, {
        headers: getHeaders()
      });
      if (res.ok) {
        const data = await res.json();
        setPolicies(data);
        if (data.length > 0 && !selectedPolicyId) {
          setSelectedPolicyId(data[0].opportunity_id);
        }
      }
    } catch (err) {
      console.error("Error searching policies:", err);
    }
  };

  const fetchEligibility = async () => {
    if (!selectedPolicyId) return;
    try {
      const pRes = await fetch(`${API_BASE}/policies/${selectedPolicyId}`, {
        headers: getHeaders()
      });
      if (pRes.ok) {
        const pData = await pRes.json();
        setSelectedPolicy(pData);
      }

      const res = await fetch(`${API_BASE}/eligibility`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({ policy_id: selectedPolicyId })
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

  // Auth Operations
  const handleAuth = async (e: React.FormEvent) => {
    e.preventDefault();
    setAuthError(null);
    const endpoint = isRegistering ? 'signup' : 'login';
    const body = isRegistering 
      ? { email: authEmail, password: authPassword, user_type: registerType }
      : { email: authEmail, password: authPassword };
      
    try {
      const res = await fetch(`${API_BASE}/auth/${endpoint}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      });
      const data = await res.json();
      if (!res.ok) {
        setAuthError(data.detail || "Authentication failed");
        return;
      }
      
      if (isRegistering) {
        setIsRegistering(false);
        setAuthPassword('');
        setAuthError("Đăng ký thành công! Vui lòng đăng nhập.");
      } else {
        setToken(data.token);
      }
    } catch (err) {
      setAuthError("Cannot connect to backend server");
    }
  };

  const handleLogout = async () => {
    try {
      await fetch(`${API_BASE}/auth/logout`, {
        method: 'POST',
        headers: getHeaders()
      });
    } catch (err) {}
    setToken(null);
    setUser(null);
    setIsProfileDropdownOpen(false);
  };

  // Avatar Upload
  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!e.target.files || e.target.files.length === 0) return;
    const file = e.target.files[0];
    const formData = new FormData();
    formData.append('file', file);

    try {
      const res = await fetch(`${API_BASE}/users/avatar`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`
        },
        body: formData
      });
      if (res.ok) {
        const data = await res.json();
        setUser((prev: any) => ({ ...prev, avatar_path: data.avatar_url }));
      }
    } catch (err) {
      console.error(err);
    }
  };

  // Change Password
  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPwdError(null);
    setPwdMessage(null);
    try {
      const res = await fetch(`${API_BASE}/users/change-password`, {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify({ old_password: oldPassword, new_password: newPassword })
      });
      const data = await res.json();
      if (res.ok) {
        setPwdMessage("Đổi mật khẩu thành công!");
        setOldPassword('');
        setNewPassword('');
      } else {
        setPwdError(data.detail || "Đổi mật khẩu thất bại");
      }
    } catch (err) {
      setPwdError("Lỗi kết nối");
    }
  };

  // Toggle user mode (Company vs Individual)
  const handleToggleMode = async () => {
    if (!user) return;
    const targetMode = user.user_type === 'COMPANY_MANAGER' ? 'INDIVIDUAL' : 'COMPANY_MANAGER';
    try {
      const res = await fetch(`${API_BASE}/users/mode`, {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify({ user_type: targetMode })
      });
      if (res.ok) {
        setUser((prev: any) => ({ ...prev, user_type: targetMode }));
        setIsProfileDropdownOpen(false);
      }
    } catch (err) {
      console.error(err);
    }
  };

  // Delete account
  const handleDeleteAccount = async () => {
    if (!window.confirm("Bạn có chắc chắn muốn xóa tài khoản? Hành động này sẽ xóa toàn bộ hồ sơ và không thể hoàn tác.")) return;
    try {
      const res = await fetch(`${API_BASE}/users`, {
        method: 'DELETE',
        headers: getHeaders()
      });
      if (res.ok) {
        setToken(null);
        setUser(null);
        setShowSettingsModal(false);
        setIsProfileDropdownOpen(false);
      }
    } catch (err) {
      console.error(err);
    }
  };

  // Personal Passport Edit Save
  const handleSavePersonalPassport = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await fetch(`${API_BASE}/personal-passports/me`, {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify(personalPassport)
      });
      if (res.ok) {
        fetchPersonalPassport();
        alert("Cập nhật hồ sơ cá nhân thành công!");
      }
    } catch (err) {
      console.error(err);
    }
  };

  // Document Ingestion
  const handleMultiUpload = async () => {
    if (!uploadFiles || uploadFiles.length === 0) return;
    setExtracting(true);
    const formData = new FormData();
    for (let i = 0; i < uploadFiles.length; i++) {
      formData.append('files', uploadFiles[i]);
    }

    try {
      const endpoint = uploadFiles.length > 1 ? 'extract-multi' : 'extract';
      const res = await fetch(`${API_BASE}/${endpoint}`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`
        },
        body: formData
      });
      
      if (res.ok) {
        const data = await res.json();
        if (user?.user_type === 'COMPANY_MANAGER') {
          setCompanyPassport(data);
        } else {
          setPersonalPassport(data);
        }
        setShowUploadModal(false);
        setUploadFiles(null);
      } else {
        const errorData = await res.json();
        alert(`Extraction failed: ${errorData.detail}`);
      }
    } catch (err) {
      alert("Lỗi kết nối khi trích xuất tài liệu");
    } finally {
      setExtracting(false);
    }
  };

  // Gated Reviews Approve/Reject
  const handleCreateDraft = async () => {
    setDraftError(null);
    try {
      const res = await fetch(`${API_BASE}/drafts`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({ policy_id: selectedPolicyId })
      });
      if (res.ok) {
        const data = await res.json();
        setDraftId(data.draft_id);
        setDraftStatus(data.status);
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleUpdateDraftStatus = async (status: string) => {
    if (!draftId) return;
    setDraftError(null);
    try {
      const res = await fetch(`${API_BASE}/drafts/${draftId}/status`, {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify({ status, reviewer_comments: reviewerComments })
      });
      const data = await res.json();
      if (res.ok) {
        setDraftStatus(data.status);
      } else {
        // Render strict validation gate warning
        setDraftError(data.detail || "Giao dịch không hợp lệ.");
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleProvenanceOverride = async (fieldName: string, value: any) => {
    if (!companyPassport) return;
    const updatedProv = {
      ...companyPassport[fieldName],
      value: typeof companyPassport[fieldName].value === 'number' ? Number(value) : value
    };
    const updatedPassport = { ...companyPassport, [fieldName]: updatedProv };
    
    try {
      const res = await fetch(`${API_BASE}/passports/${selectedCompanyId}`, {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify({ passport_data: updatedPassport })
      });
      if (res.ok) {
        setCompanyPassport(updatedPassport);
        setEditingField(null);
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleResolveConflict = async (fieldName: string, value: any) => {
    if (!companyPassport) return;
    const updatedProv = {
      ...companyPassport[fieldName],
      value: value,
      status: 'EXTRACTED',
      conflicts: []
    };
    const updatedPassport = { ...companyPassport, [fieldName]: updatedProv };

    try {
      const res = await fetch(`${API_BASE}/passports/${selectedCompanyId}`, {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify({ passport_data: updatedPassport })
      });
      if (res.ok) {
        setCompanyPassport(updatedPassport);
        setShowConflictModal(false);
      }
    } catch (err) {
      console.error(err);
    }
  };

  // Sync Policies with SHA256 Diff Checking
  const handleSyncPolicies = async () => {
    setSyncing(true);
    try {
      const res = await fetch(`${API_BASE}/sync`, {
        method: 'POST',
        headers: getHeaders()
      });
      if (res.ok) {
        const data = await res.json();
        setSyncLogs(data.logs);
        fetchPolicyAlerts();
        searchPolicies();
      }
    } catch (err) {
      console.error(err);
    } finally {
      setSyncing(false);
    }
  };

  const fetchAuditLogs = async () => {
    try {
      const res = await fetch(`${API_BASE}/audit_logs`, {
        headers: getHeaders()
      });
      if (res.ok) {
        const data = await res.json();
        setAuditLogs(data);
        setShowAuditModal(true);
      }
    } catch (err) {
      console.error(err);
    }
  };

  // Helper formatting functions
  const formatValue = (key: string, prov: any) => {
    if (!prov || prov.value === undefined || prov.value === "") return "Chưa bổ sung";
    if (prov.status === "MISSING") return "Chưa bổ sung";
    
    if (key === 'revenue' || key === 'registered_capital') {
      return `${(prov.value / 1e9).toLocaleString()} tỷ VND`;
    }
    if (key === 'rd_spend_ratio') {
      return `${(prov.value * 100).toFixed(1)}%`;
    }
    if (key === 'employee_count') {
      return `${prov.value} nhân sự`;
    }
    return String(prov.value);
  };

  const formatFieldName = (key: string) => {
    const mapping: Record<string, string> = {
      company_name: 'Tên Doanh Nghiệp',
      tax_code: 'Mã Số Thuế',
      industry: 'Ngành Nghề Kinh Doanh',
      location: 'Địa Điểm Trụ Sở',
      employee_count: 'Quy Mô Nhân Sự',
      rd_spend_ratio: 'Tỷ Lệ Chi R&D',
      revenue: 'Doanh Thu Hằng Năm',
      registered_capital: 'Vốn Điều Lệ Đăng Ký'
    };
    return mapping[key] || key;
  };

  // Key Event listeners for accessibility Escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setShowAuditModal(false);
        setShowSyncModal(false);
        setShowUploadModal(false);
        setShowConflictModal(false);
        setShowSettingsModal(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  // ------------------ LOGIN / REGISTER RENDER ------------------
  if (!token) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center p-4">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-indigo-900/20 via-slate-950 to-slate-950 pointer-events-none" />
        <div className="w-full max-w-md bg-slate-900/60 border border-slate-800/80 rounded-2xl p-8 backdrop-blur-xl shadow-2xl relative z-10">
          <div className="flex items-center gap-3 justify-center mb-6">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-indigo-500 to-violet-500 flex items-center justify-center shadow-lg shadow-indigo-500/25">
              <ShieldCheck className="w-6 h-6 text-white" />
            </div>
            <span className="text-2xl font-bold text-slate-100 tracking-tight">P2B Platform</span>
          </div>

          <h2 className="text-xl font-semibold text-center text-slate-100 mb-6">
            {isRegistering ? t('auth.signupHeader') : t('auth.loginHeader')}
          </h2>
 
          <form onSubmit={handleAuth} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1.5">{t('auth.email')}</label>
              <input 
                type="email" 
                required
                className="w-full bg-slate-950 border border-slate-800 focus:border-indigo-500/50 rounded-lg px-4 py-2.5 text-slate-100 text-sm placeholder:text-slate-400 focus:outline-none focus:ring-1 focus:ring-indigo-500/20 transition-all"
                value={authEmail}
                onChange={(e) => setAuthEmail(e.target.value)}
                placeholder="chuyengia@p2b.vn"
              />
            </div>
            
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1.5">{t('auth.password')}</label>
              <input 
                type="password" 
                required
                className="w-full bg-slate-950 border border-slate-800 focus:border-indigo-500/50 rounded-lg px-4 py-2.5 text-slate-100 text-sm placeholder:text-slate-400 focus:outline-none focus:ring-1 focus:ring-indigo-500/20 transition-all"
                value={authPassword}
                onChange={(e) => setAuthPassword(e.target.value)}
                placeholder="••••••••"
              />
            </div>
 
            {isRegistering && (
              <div>
                <label className="block text-xs font-medium text-slate-400 mb-1.5">{t('settings.userType')}</label>
                <select 
                  className="w-full bg-slate-950 border border-slate-800 focus:border-indigo-500/50 rounded-lg px-3 py-2.5 text-slate-100 text-sm focus:outline-none transition-all"
                  value={registerType}
                  onChange={(e) => setRegisterType(e.target.value)}
                >
                  <option value="COMPANY_MANAGER">{t('settings.companyMode')}</option>
                  <option value="INDIVIDUAL">{t('settings.individualMode')}</option>
                </select>
              </div>
            )}
 
            {authError && (
              <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-xs text-red-400 text-center">
                {authError}
              </div>
            )}
 
            <button 
              type="submit" 
              className="w-full py-2.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-medium text-sm transition-all shadow-lg shadow-indigo-600/15"
            >
              {isRegistering ? t('auth.signupBtn') : t('auth.loginBtn')}
            </button>
          </form>
 
          <div className="mt-6 text-center text-xs text-slate-400">
            {isRegistering ? (
              <span>{t('auth.hasAccount')} <button onClick={() => { setIsRegistering(false); setAuthError(null); }} className="text-indigo-400 hover:underline">{t('auth.toggleLogin')}</button></span>
            ) : (
              <span>{t('auth.noAccount')} <button onClick={() => { setIsRegistering(true); setAuthError(null); }} className="text-indigo-400 hover:underline">{t('auth.toggleSignup')}</button></span>
            )}
          </div>
          
          <div className="mt-8 pt-6 border-t border-slate-800/80 text-[10px] text-slate-500 text-center space-y-1">
            <p>{t('auth.demoTitle')}</p>
            <p className="font-mono">aitech@p2b.vn | semivina@p2b.vn | individual@p2b.vn</p>
          </div>
        </div>
      </div>
    );
  }

  // ------------------ MAIN DASHBOARD RENDER ------------------
  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col font-sans relative overflow-x-hidden">
      <div className="absolute top-0 left-1/4 w-[500px] h-[500px] bg-indigo-500/5 rounded-full blur-[120px] pointer-events-none" />
      <div className="absolute bottom-0 right-1/4 w-[500px] h-[500px] bg-violet-500/5 rounded-full blur-[120px] pointer-events-none" />
 
      {/* HEADER NAVBAR */}
      <header className="sticky top-0 z-40 bg-slate-950/70 backdrop-blur-xl border-b border-slate-900 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-indigo-500 to-violet-500 flex items-center justify-center shadow-lg shadow-indigo-500/20">
            <ShieldCheck className="w-5 h-5 text-white" />
          </div>
          <div>
            <h1 className="text-lg font-bold tracking-tight text-slate-100">{t('nav.title')}</h1>
            <p className="text-[10px] text-slate-400">{t('nav.subtitle')}</p>
          </div>
        </div>

        <div className="flex items-center gap-4 relative">
          <button 
            onClick={() => setShowSyncModal(true)} 
            className="flex items-center gap-2 px-3 py-1.5 rounded-lg border border-slate-800/80 bg-slate-900/30 hover:bg-slate-900/70 text-xs font-medium text-slate-300 transition-all"
            aria-label={t('nav.syncAlert')}
          >
            <RefreshCw className="w-3.5 h-3.5" />
            <span>{t('nav.syncAlert')}</span>
          </button>
          
          <button 
            onClick={fetchAuditLogs} 
            className="flex items-center gap-2 px-3 py-1.5 rounded-lg border border-slate-800/80 bg-slate-900/30 hover:bg-slate-900/70 text-xs font-medium text-slate-300 transition-all"
            aria-label={t('nav.logs')}
          >
            <History className="w-3.5 h-3.5" />
            <span>{t('nav.logs')}</span>
          </button>

          {/* User profile dropdown button */}
          <div className="relative">
            <button 
              onClick={() => setIsProfileDropdownOpen(!isProfileDropdownOpen)}
              className="w-9 h-9 rounded-full bg-indigo-600/20 border border-indigo-500/30 flex items-center justify-center hover:bg-indigo-600/40 transition-all overflow-hidden"
              aria-label={t('nav.userProfile')}
            >
              {user?.avatar_path ? (
                <img src={`http://localhost:8000${user.avatar_path}`} alt="Avatar" className="w-full h-full object-cover" />
              ) : (
                <User className="w-4 h-4 text-indigo-400" />
              )}
            </button>

            {isProfileDropdownOpen && (
              <div className="absolute right-0 mt-2.5 w-64 bg-slate-900 border border-slate-800 rounded-xl shadow-2xl py-2 z-50 animate-in fade-in slide-in-from-top-2 duration-150">
                <div className="px-4 py-2.5 border-b border-slate-800/60">
                  <p className="text-xs text-slate-400 font-medium">{t('nav.userProfile')}</p>
                  <p className="text-sm font-semibold truncate text-white">{user?.email}</p>
                  <span className="inline-block mt-1 text-[10px] px-2 py-0.5 rounded bg-indigo-600/20 border border-indigo-500/25 text-indigo-300 font-mono">
                    {user?.user_type === 'COMPANY_MANAGER' ? '🏢 ' + t('settings.companyMode') : '👤 ' + t('settings.individualMode')}
                  </span>
                </div>
                
                <button 
                  onClick={handleToggleMode} 
                  className="w-full text-left px-4 py-2 hover:bg-slate-800 text-xs flex items-center gap-2.5 transition-all text-slate-300"
                >
                  <UserCheck className="w-4 h-4 text-indigo-400" />
                  Chuyển sang {user?.user_type === 'COMPANY_MANAGER' ? t('settings.individualMode') : t('settings.companyMode')}
                </button>

                <button 
                  onClick={() => { setShowSettingsModal(true); setIsProfileDropdownOpen(false); }} 
                  className="w-full text-left px-4 py-2 hover:bg-slate-800 text-xs flex items-center gap-2.5 transition-all text-slate-300"
                >
                  <Settings className="w-4 h-4 text-slate-400" />
                  {t('nav.settings')}
                </button>

                <button 
                  onClick={handleLogout} 
                  className="w-full text-left px-4 py-2 hover:bg-slate-800 text-xs flex items-center gap-2.5 text-red-400 transition-all border-t border-slate-800/60"
                >
                  <LogOut className="w-4 h-4" />
                  {t('nav.logout')}
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* DASHBOARD WORKSPACE */}
      <div className="flex-1 max-w-[1440px] w-full mx-auto px-6 py-6 grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        
        {/* PANEL 1: PROFILE / COMPANY PASSPORT CONSOLE (lg:col-span-4) */}
        <section className="lg:col-span-4 bg-slate-900/30 border border-slate-800/50 rounded-xl p-5 backdrop-blur-md space-y-5">
          <div className="flex justify-between items-center pb-3 border-b border-slate-800/50">
            <h2 className="text-sm font-semibold tracking-tight text-slate-300 uppercase">
              {user?.user_type === 'COMPANY_MANAGER' ? t('passport.companyTitle') : t('passport.personalTitle')}
            </h2>
            <button 
              onClick={() => setShowUploadModal(true)} 
              className="flex items-center gap-1.5 text-xs px-2.5 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 font-medium text-white transition-all shadow-md shadow-indigo-600/10"
            >
              <Camera className="w-3.5 h-3.5" />
              Tải hồ sơ lên
            </button>
          </div>

          {/* SWITCH COMPANY TENANT (Only for COMPANY_MANAGER mode) */}
          {user?.user_type === 'COMPANY_MANAGER' && (
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-slate-400 flex items-center gap-1.5">
                <Database className="w-3.5 h-3.5" /> Tenant Doanh nghiệp
              </label>
              <select
                className="w-full bg-slate-950 border border-slate-800/80 rounded-lg px-3 py-2 text-xs text-white focus:outline-none transition-all"
                value={selectedCompanyId}
                onChange={(e) => setSelectedCompanyId(e.target.value)}
              >
                {companies.map(c => (
                  <option key={c} value={c}>{c.replace(/_/g, ' ')}</option>
                ))}
              </select>
            </div>
          )}

          {/* DYNAMIC PASSPORT CONTENT */}
          {user?.user_type === 'COMPANY_MANAGER' ? (
            // Company passport rendering
            companyPassport ? (
              <div className="space-y-3">
                {Object.keys(companyPassport).map((key) => {
                  if (key === 'metadata') return null;
                  const field = companyPassport[key];
                  const isSelected = selectedField === key;
                  return (
                    <div 
                      key={key} 
                      onClick={() => setSelectedField(key)}
                      className={`p-3 rounded-lg border text-left cursor-pointer transition-all ${
                        isSelected 
                          ? 'bg-indigo-600/10 border-indigo-500/50 shadow-inner' 
                          : 'bg-slate-900/50 border-slate-800/80 hover:border-slate-700'
                      }`}
                    >
                      <div className="flex justify-between items-start">
                        <span className="text-xs text-slate-400 font-medium">{formatFieldName(key)}</span>
                        <span className={`text-[9px] px-1.5 py-0.5 rounded font-mono ${
                          field.status === 'EXTRACTED' ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/15' :
                          field.status === 'USER_CONFIRMED' ? 'bg-blue-500/10 text-blue-400 border border-blue-500/15' :
                          field.status === 'CONFLICTED' ? 'bg-amber-500/10 text-amber-400 border border-amber-500/15 animate-pulse' :
                          'bg-rose-500/10 text-rose-400 border border-rose-500/15'
                        }`}>
                          {field.status}
                        </span>
                      </div>
                      
                      {editingField === key ? (
                        <div className="mt-2 flex items-center gap-2" onClick={e => e.stopPropagation()}>
                          <input 
                            className="flex-1 bg-slate-950 border border-slate-800 rounded px-2.5 py-1 text-xs text-slate-100 focus:outline-none"
                            value={editValue}
                            onChange={(e) => setEditValue(e.target.value)}
                          />
                          <button onClick={() => handleProvenanceOverride(key, editValue)} className="px-2.5 py-1 bg-emerald-600 rounded text-xs">Lưu</button>
                          <button onClick={() => setEditingField(null)} className="px-2.5 py-1 bg-slate-800 rounded text-xs">Hủy</button>
                        </div>
                      ) : (
                        <div className="mt-1.5 flex justify-between items-center">
                          <span className="text-sm font-semibold text-slate-100">{formatValue(key, field)}</span>
                          <button 
                            onClick={(e) => { e.stopPropagation(); handleEditField(key, field.value); }} 
                            className="text-slate-500 hover:text-white p-1"
                            aria-label={`Sửa ${formatFieldName(key)}`}
                          >
                            <Edit2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      )}
                    </div>
                  );
                })}
                
                {companyPassport.metadata?.uploaded_files && companyPassport.metadata.uploaded_files.length > 0 && (
                  <div className="mt-4 pt-3 border-t border-slate-800/85">
                    <h3 className="text-xs font-semibold text-slate-400 mb-2 flex items-center gap-1.5">
                      <FileText className="w-3.5 h-3.5 text-indigo-400" />
                      Lịch sử tài liệu đã xử lý
                    </h3>
                    <div className="space-y-1.5 max-h-[160px] overflow-y-auto pr-1">
                      {companyPassport.metadata.uploaded_files.map((file: any, index: number) => (
                        <div key={index} className="flex justify-between items-center bg-slate-950/45 border border-slate-900 px-2.5 py-1.5 rounded-md text-[11px]">
                          <span className="text-slate-300 truncate max-w-[150px]" title={file.filename}>
                            {file.filename}
                          </span>
                          <span className="text-slate-500 font-mono text-[9px]">
                            {new Date(file.uploaded_at).toLocaleString('vi-VN', {hour: '2-digit', minute:'2-digit', day: '2-digit', month: '2-digit'})}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <div className="py-12 text-center text-xs text-slate-500 flex flex-col items-center gap-2">
                <RefreshCw className="w-6 h-6 animate-spin text-indigo-500" />
                <span>Đang tải hồ sơ...</span>
              </div>
            )
          ) : (
            // Individual passport form
            <>
              <form onSubmit={handleSavePersonalPassport} className="space-y-4">
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Họ và tên</label>
                  <input 
                    type="text"
                    className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 focus:outline-none"
                    value={personalPassport.full_name || ''}
                    onChange={(e) => setPersonalPassport({ ...personalPassport, full_name: e.target.value })}
                  />
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-medium text-slate-400 mb-1">Năm sinh</label>
                    <input 
                      type="number"
                      className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 focus:outline-none"
                      value={personalPassport.birth_year || 0}
                      onChange={(e) => setPersonalPassport({ ...personalPassport, birth_year: Number(e.target.value) })}
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-slate-400 mb-1">Thành phố</label>
                    <input 
                      type="text"
                      className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 focus:outline-none"
                      value={personalPassport.location || ''}
                      onChange={(e) => setPersonalPassport({ ...personalPassport, location: e.target.value })}
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Nghề nghiệp / Chuyên môn</label>
                  <input 
                    type="text"
                    className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 focus:outline-none"
                    value={personalPassport.occupation || ''}
                    onChange={(e) => setPersonalPassport({ ...personalPassport, occupation: e.target.value })}
                    placeholder="Semiconductor Engineer"
                  />
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-medium text-slate-400 mb-1">Bằng cấp</label>
                    <input 
                      type="text"
                      className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 focus:outline-none"
                      value={personalPassport.degree || ''}
                      onChange={(e) => setPersonalPassport({ ...personalPassport, degree: e.target.value })}
                      placeholder="Master"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-slate-400 mb-1">Thu nhập tháng (VND)</label>
                    <input 
                      type="number"
                      className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 focus:outline-none"
                      value={personalPassport.monthly_income || 0}
                      onChange={(e) => setPersonalPassport({ ...personalPassport, monthly_income: Number(e.target.value) })}
                    />
                  </div>
                </div>

                <button 
                  type="submit" 
                  className="w-full py-2 bg-indigo-600 hover:bg-indigo-500 rounded-lg text-xs font-medium text-white transition-all"
                >
                  Cập Nhật Hồ Sơ Cá Nhân
                </button>
              </form>
              
              {personalPassport.uploaded_files && personalPassport.uploaded_files.length > 0 && (
                <div className="mt-4 pt-3 border-t border-slate-800/80 text-left">
                  <h3 className="text-xs font-semibold text-slate-400 mb-2 flex items-center gap-1.5">
                    <FileText className="w-3.5 h-3.5 text-indigo-400" />
                    Lịch sử tài liệu đã xử lý
                  </h3>
                  <div className="space-y-1.5 max-h-[160px] overflow-y-auto pr-1">
                    {personalPassport.uploaded_files.map((file: any, index: number) => (
                      <div key={index} className="flex justify-between items-center bg-slate-950/40 border border-slate-900 px-2.5 py-1.5 rounded-md text-[11px]">
                        <span className="text-slate-300 truncate max-w-[150px]" title={file.filename}>
                          {file.filename}
                        </span>
                        <span className="text-slate-500 font-mono text-[9px]">
                          {new Date(file.uploaded_at).toLocaleString('vi-VN', {hour: '2-digit', minute:'2-digit', day: '2-digit', month: '2-digit'})}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
        </section>

        {/* PANEL 2: POLICIES MATCHING & CHECKLIST (lg:col-span-8) */}
        <main className="lg:col-span-8 space-y-6">
          
          {/* SEARCH & FILTERS */}
          <section className="bg-slate-900/30 border border-slate-800/50 rounded-xl p-4 flex flex-col md:flex-row gap-3">
            <div className="flex-1 bg-slate-950 border border-slate-800/80 rounded-lg px-3 py-2 flex items-center gap-2">
              <Search className="w-4 h-4 text-slate-500" />
              <input 
                className="bg-transparent flex-1 text-xs text-slate-100 focus:outline-none"
                placeholder="Tìm kiếm chính sách (ví dụ: ưu đãi thuế bán dẫn, tài trợ khoa học công nghệ...)"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
          </section>

          {/* DUAL WORKSPACE LAYOUT (Policy opportunities on left, eligibility details on right) */}
          <div className="grid grid-cols-1 md:grid-cols-12 gap-6">
            
            {/* POLICY LISTING (col-span-5) */}
            <section className="md:col-span-5 space-y-3 max-h-[600px] overflow-y-auto">
              <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Danh sách chính sách phù hợp</h3>
              {policies.length > 0 ? (
                policies.map((p) => {
                  const isSelected = selectedPolicyId === p.opportunity_id;
                  // Detect real change alerts
                  const hasAlert = policyAlerts.some(a => a.document_id === p.source_legal_documents[0]);
                  
                  return (
                    <div 
                      key={p.opportunity_id}
                      onClick={() => { setSelectedPolicyId(p.opportunity_id); setDraftError(null); }}
                      className={`p-4 rounded-xl border text-left cursor-pointer transition-all relative ${
                        isSelected 
                          ? 'bg-gradient-to-br from-indigo-900/20 to-indigo-950/20 border-indigo-500/50 shadow-lg shadow-indigo-500/5' 
                          : 'bg-slate-900/40 border-slate-800/80 hover:border-slate-700'
                      }`}
                    >
                      {hasAlert && (
                        <span className="absolute top-2 right-2 flex h-2 w-2">
                          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75"></span>
                          <span className="relative inline-flex rounded-full h-2 w-2 bg-amber-500"></span>
                        </span>
                      )}
                      <h4 className="text-xs font-bold text-slate-100 pr-4">{p.title}</h4>
                      <p className="text-[10px] text-slate-400 mt-1 line-clamp-2">{p.benefits}</p>
                      
                      <div className="mt-2.5 flex items-center justify-between">
                        <span className="text-[9px] text-slate-500 font-mono">Độ tương hợp: {(p.score * 100).toFixed(0)}%</span>
                        {p.deadline && (
                          <span className="text-[9px] px-1.5 py-0.5 rounded bg-rose-500/10 text-rose-400 font-mono">Hạn: {p.deadline}</span>
                        )}
                      </div>
                    </div>
                  );
                })
              ) : (
                <div className="p-8 text-center text-xs text-slate-500 border border-slate-800 border-dashed rounded-xl">
                  Không tìm thấy chính sách phù hợp.
                </div>
              )}
            </section>

            {/* CHECKLIST & INLINE EVIDENCE (col-span-7) */}
            <section className="md:col-span-7 space-y-4">
              {selectedPolicy && eligibility ? (
                <div className="bg-slate-900/20 border border-slate-800/60 rounded-xl p-5 space-y-5">
                  <div className="flex justify-between items-start pb-3 border-b border-slate-800/50">
                    <div>
                      <h3 className="text-sm font-bold text-slate-100">{selectedPolicy.title}</h3>
                      <p className="text-[10px] text-slate-500 mt-0.5">Văn bản nguồn: <span className="font-mono">{selectedPolicy.source_legal_documents.join(", ")}</span></p>
                    </div>
                    
                    <div className="flex items-center gap-1.5">
                      <span className="text-xs text-slate-400">Kết quả:</span>
                      <span className={`text-xs font-bold px-2 py-0.5 rounded font-mono ${
                        eligibility.status === 'MET' ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/15' :
                        eligibility.status === 'MISSING_INFO' ? 'bg-amber-500/10 text-amber-400 border border-amber-500/15' :
                        'bg-rose-500/10 text-rose-400 border border-rose-500/15'
                      }`}>
                        {eligibility.status}
                      </span>
                    </div>
                  </div>

                  {/* Policy Changed Warning Alert (Real Ingestion Alerts) */}
                  {policyAlerts.some(a => a.document_id === selectedPolicy.source_legal_documents[0]) && (
                    <div className="p-3.5 bg-amber-500/10 border border-amber-500/25 rounded-lg flex items-start gap-2.5">
                      <AlertTriangle className="w-4 h-4 text-amber-400 flex-shrink-0 mt-0.5" />
                      <div className="text-[11px] text-amber-300">
                        <p className="font-semibold">Cảnh báo cập nhật pháp lý (SHA-256 Diff Alert):</p>
                        <p className="mt-0.5">
                          {policyAlerts.find(a => a.document_id === selectedPolicy.source_legal_documents[0])?.change_description}
                        </p>
                      </div>
                    </div>
                  )}

                  {/* CRITERIA LIST */}
                  <div className="space-y-3">
                    <h4 className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Tiêu chí kiểm tra điều kiện</h4>
                    {eligibility.details?.rules?.map((rule: any) => {
                      const isRuleSelected = selectedRuleId === rule.rule_id;
                      return (
                        <div 
                          key={rule.rule_id}
                          onClick={() => setSelectedRuleId(rule.rule_id)}
                          className={`p-3 rounded-lg border text-left cursor-pointer transition-all ${
                            isRuleSelected 
                              ? 'bg-slate-900 border-indigo-500/50 shadow-inner' 
                              : 'bg-slate-950/40 border-slate-900 hover:border-slate-800'
                          }`}
                        >
                          <div className="flex justify-between items-start gap-2">
                            <span className="text-xs text-slate-100 font-medium">{rule.description}</span>
                            <div className="flex items-center gap-1.5 flex-shrink-0">
                              {rule.status === 'MET' && <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />}
                              {rule.status === 'NOT_MET' && <XCircle className="w-3.5 h-3.5 text-rose-400" />}
                              {rule.status === 'MISSING_INFO' && <AlertCircle className="w-3.5 h-3.5 text-amber-400" />}
                              <span className={`text-[9px] font-bold font-mono ${
                                rule.status === 'MET' ? 'text-emerald-400' :
                                rule.status === 'MISSING_INFO' ? 'text-amber-400' : 'text-rose-400'
                              }`}>{rule.status}</span>
                            </div>
                          </div>
                          
                          {/* Expanded Evidence View */}
                          {isRuleSelected && (
                            <div className="mt-2.5 pt-2.5 border-t border-slate-800/80 space-y-2 text-[10px]">
                              {rule.observed_value !== undefined && (
                                <div>
                                  <span className="text-slate-400 font-medium">Giá trị thực tế:</span>{' '}
                                  <span className="text-slate-100 font-mono">{String(rule.observed_value)}</span>
                                </div>
                              )}
                              {rule.citation && (
                                <div className="p-2 bg-slate-950 rounded border border-slate-800/40 space-y-1">
                                  <p className="text-slate-400 font-medium flex items-center gap-1">
                                    <FileText className="w-3 h-3 text-indigo-400" /> Trích dẫn pháp lý ({rule.citation.article}):
                                  </p>
                                  <p className="italic text-slate-300">"{rule.citation.quote}"</p>
                                  {rule.citation.source_url && (
                                    <a href={rule.citation.source_url} target="_blank" rel="noreferrer" className="text-indigo-400 hover:underline flex items-center gap-0.5 mt-1">
                                      <span>Xem cổng văn bản chính phủ</span> <ExternalLink className="w-2.5 h-2.5" />
                                    </a>
                                  )}
                                </div>
                              )}
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>

                  {/* FIELD EVIDENCE CARD & CONFLICT BANNER */}
                  {user?.user_type === 'COMPANY_MANAGER' && selectedField && companyPassport && (
                    <div className="p-3.5 bg-slate-950/60 border border-slate-800 rounded-lg space-y-3">
                      <div className="flex justify-between items-center">
                        <span className="text-xs font-semibold text-slate-400 flex items-center gap-1.5">
                          <ShieldCheck className="w-4 h-4 text-indigo-400" /> Nguồn minh chứng ({formatFieldName(selectedField)})
                        </span>
                        {companyPassport[selectedField]?.status === 'CONFLICTED' && (
                          <button 
                            onClick={() => {
                              setConflictFieldName(selectedField);
                              setConflictFieldData(companyPassport[selectedField]);
                              setShowConflictModal(true);
                            }}
                            className="text-[10px] px-2 py-0.5 rounded bg-amber-500/10 text-amber-400 border border-amber-500/20 font-bold animate-pulse"
                          >
                            Xử lý mâu thuẫn
                          </button>
                        )}
                      </div>

                      <div className="text-[11px] space-y-1.5 text-slate-300">
                        <p><span className="text-slate-500">Nguồn trích xuất:</span> <span className="font-semibold text-slate-100">{companyPassport[selectedField]?.source_type}</span></p>
                        <p><span className="text-slate-500">File tài liệu:</span> <span className="font-mono text-indigo-400">{companyPassport[selectedField]?.source_uri || 'Không có'}</span></p>
                        <p><span className="text-slate-500">Vị trí:</span> <span>{companyPassport[selectedField]?.source_location || 'Không có'}</span></p>
                        {companyPassport[selectedField]?.evidence_quote && (
                          <div className="mt-1 p-2 bg-slate-900 border border-slate-800/40 rounded text-slate-400 italic">
                            "{companyPassport[selectedField].evidence_quote}"
                          </div>
                        )}
                      </div>
                    </div>
                  )}

                  {/* SUBMIT / REVIEW STAGE */}
                  <div className="pt-3 border-t border-slate-800/50 flex flex-col gap-3">
                    {!draftStatus ? (
                      <button 
                        onClick={handleCreateDraft}
                        className="w-full py-2.5 bg-indigo-600 hover:bg-indigo-500 rounded-lg text-xs font-bold text-white tracking-wide transition-all shadow-lg shadow-indigo-600/10"
                      >
                        Tạo Hồ Sơ Nháp (Draft Application)
                      </button>
                    ) : (
                      <div className="space-y-4 bg-slate-950/40 border border-slate-800 p-4 rounded-xl">
                        <div className="flex justify-between items-center">
                          <span className="text-xs text-slate-400">Trạng thái hồ sơ:</span>
                          <span className={`text-xs font-bold font-mono px-2 py-0.5 rounded ${
                            draftStatus === 'GENERATED' ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' :
                            draftStatus === 'APPROVED' ? 'bg-indigo-500/10 text-indigo-400 border border-indigo-500/20' :
                            draftStatus === 'REJECTED' ? 'bg-rose-500/10 text-rose-400 border border-rose-500/20' :
                            'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                          }`}>{draftStatus}</span>
                        </div>

                        {draftStatus === 'PENDING_REVIEW' && (
                          <div className="space-y-3">
                            <div>
                              <label className="block text-xs font-medium text-slate-400 mb-1">Góp ý của người duyệt (Reviewer Comments)</label>
                              <textarea 
                                className="w-full h-16 bg-slate-950 border border-slate-800 rounded-lg p-2.5 text-xs text-slate-100 focus:outline-none"
                                value={reviewerComments}
                                onChange={(e) => setReviewerComments(e.target.value)}
                                placeholder="Ghi chú thẩm định hồ sơ..."
                              />
                            </div>
                            
                            {/* Strict Audit Warning Notification */}
                            {draftError && (
                              <div className="p-3 bg-rose-500/10 border border-rose-500/25 rounded-lg flex items-start gap-2">
                                <AlertTriangle className="w-3.5 h-3.5 text-rose-400 flex-shrink-0 mt-0.5" />
                                <span className="text-[10px] text-rose-400 font-mono leading-tight">{draftError}</span>
                              </div>
                            )}

                            <div className="grid grid-cols-2 gap-3">
                              <button 
                                onClick={() => handleUpdateDraftStatus('APPROVED')}
                                className="py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-bold transition-all shadow-md shadow-emerald-600/15"
                              >
                                Phê Duyệt & Xuất Đơn
                              </button>
                              <button 
                                onClick={() => handleUpdateDraftStatus('REJECTED')}
                                className="py-2 bg-rose-600/20 border border-rose-500/20 hover:bg-rose-600/30 text-rose-400 rounded-lg text-xs font-bold transition-all"
                              >
                                Từ Chối Hồ Sơ
                              </button>
                            </div>
                          </div>
                        )}

                        {draftStatus === 'GENERATED' && (
                          <a 
                            href={`${API_BASE}/drafts/${draftId}/download?authorization=Bearer ${token}`}
                            download
                            className="w-full py-2.5 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white rounded-lg text-xs font-bold flex items-center justify-center gap-2 transition-all shadow-lg shadow-emerald-600/20"
                          >
                            <Download className="w-4 h-4" />
                            Tải Đơn Đăng Ký (.docx)
                          </a>
                        )}

                        {draftStatus === 'REJECTED' && (
                          <p className="text-xs text-rose-400 text-center italic">Hồ sơ đã bị từ chối phê duyệt.</p>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              ) : (
                <div className="py-24 text-center text-xs text-slate-500 border border-slate-800 border-dashed rounded-xl flex flex-col items-center gap-2">
                  <ShieldCheck className="w-8 h-8 text-slate-600" />
                  <span>Chọn một chính sách để xem chi tiết điều kiện thẩm định.</span>
                </div>
              )}
            </section>
          </div>
        </main>
      </div>

      {/* FOOTER */}
      <footer className="py-8 border-t border-slate-900/60 mt-12 text-center text-[10px] text-slate-500 space-y-1">
        <p>© 2026 P2B Platform. Built using FastAPI, SentenceTransformers, and Gemini 3.1 Flash Lite.</p>
        <p className="font-mono text-[9px] text-slate-600">Secure isolation & provenance logging activated.</p>
      </footer>

      {/* ------------------ MODALS ------------------ */}

      {/* SETTINGS MODAL */}
      {showSettingsModal && (
        <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4" role="dialog" aria-modal="true">
          <div className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-2xl relative">
            <button onClick={() => setShowSettingsModal(false)} className="absolute top-4 right-4 text-slate-400 hover:text-white" aria-label={t('settings.closeBtn')}>
              <X className="w-4 h-4" />
            </button>
            <h3 className="text-sm font-semibold text-white uppercase tracking-wider mb-5 flex items-center gap-2">
              <Settings className="w-4 h-4 text-indigo-400" /> {t('settings.title')}
            </h3>

            <div className="space-y-6 max-h-[70vh] overflow-y-auto pr-1">
              {/* Profile Image upload section */}
              <div className="space-y-2">
                <label className="block text-xs font-medium text-slate-400">Hình đại diện (Avatar)</label>
                <div className="flex items-center gap-4">
                  <div className="w-14 h-14 rounded-full bg-slate-950 border border-slate-800 flex items-center justify-center overflow-hidden">
                    {user?.avatar_path ? (
                      <img src={`http://localhost:8000${user.avatar_path}`} alt="Avatar Preview" className="w-full h-full object-cover" />
                    ) : (
                      <User className="w-6 h-6 text-slate-500" />
                    )}
                  </div>
                  <button 
                    onClick={() => avatarInputRef.current?.click()}
                    className="px-3 py-1.5 rounded-lg border border-slate-800 bg-slate-950 text-xs font-semibold hover:bg-slate-900 transition-all"
                  >
                    {t('passport.uploadBtn')}
                  </button>
                  <input ref={avatarInputRef} type="file" className="hidden" accept="image/*" onChange={handleAvatarUpload} />
                </div>
              </div>

              {/* User Mode selection */}
              <div className="space-y-2 pt-4 border-t border-slate-850">
                <label className="block text-xs font-medium text-slate-400">{t('settings.userType')}</label>
                <div className="flex gap-2">
                  <button 
                    onClick={() => {
                      if (user?.user_type !== 'COMPANY_MANAGER') handleToggleMode();
                    }}
                    className={`flex-1 px-3 py-1.5 rounded-lg border text-xs font-semibold transition-all ${user?.user_type === 'COMPANY_MANAGER' ? 'bg-indigo-600 border-indigo-500 text-white' : 'bg-slate-950 border-slate-800 text-slate-400 hover:bg-slate-900'}`}
                  >
                    {t('settings.companyMode')}
                  </button>
                  <button 
                    onClick={() => {
                      if (user?.user_type !== 'INDIVIDUAL') handleToggleMode();
                    }}
                    className={`flex-1 px-3 py-1.5 rounded-lg border text-xs font-semibold transition-all ${user?.user_type === 'INDIVIDUAL' ? 'bg-indigo-600 border-indigo-500 text-white' : 'bg-slate-950 border-slate-800 text-slate-400 hover:bg-slate-900'}`}
                  >
                    {t('settings.individualMode')}
                  </button>
                </div>
              </div>

              {/* Theme selection */}
              <div className="space-y-2 pt-4 border-t border-slate-850">
                <label className="block text-xs font-medium text-slate-400">{t('settings.theme')}</label>
                <div className="flex gap-2">
                  <button 
                    onClick={() => handleThemeChange('light')}
                    className={`flex-1 px-3 py-1.5 rounded-lg border text-xs font-semibold transition-all ${theme === 'light' ? 'bg-indigo-600 border-indigo-500 text-white' : 'bg-slate-950 border-slate-800 text-slate-400 hover:bg-slate-900'}`}
                  >
                    {t('settings.lightMode')}
                  </button>
                  <button 
                    onClick={() => handleThemeChange('dark')}
                    className={`flex-1 px-3 py-1.5 rounded-lg border text-xs font-semibold transition-all ${theme === 'dark' ? 'bg-indigo-600 border-indigo-500 text-white' : 'bg-slate-950 border-slate-800 text-slate-400 hover:bg-slate-900'}`}
                  >
                    {t('settings.darkMode')}
                  </button>
                </div>
              </div>

              {/* Language selection */}
              <div className="space-y-2 pt-4 border-t border-slate-850">
                <label className="block text-xs font-medium text-slate-400">{t('settings.language')}</label>
                <div className="flex gap-2">
                  <button 
                    onClick={() => handleLanguageChange('vi')}
                    className={`flex-1 px-3 py-1.5 rounded-lg border text-xs font-semibold transition-all ${locale === 'vi' ? 'bg-indigo-600 border-indigo-500 text-white' : 'bg-slate-950 border-slate-800 text-slate-400 hover:bg-slate-900'}`}
                  >
                    {t('settings.vi')}
                  </button>
                  <button 
                    onClick={() => handleLanguageChange('en')}
                    className={`flex-1 px-3 py-1.5 rounded-lg border text-xs font-semibold transition-all ${locale === 'en' ? 'bg-indigo-600 border-indigo-500 text-white' : 'bg-slate-950 border-slate-800 text-slate-400 hover:bg-slate-900'}`}
                  >
                    {t('settings.en')}
                  </button>
                </div>
              </div>

              {/* Password update form */}
              <form onSubmit={handleChangePassword} className="space-y-3.5 pt-4 border-t border-slate-850">
                <p className="text-xs font-semibold text-slate-300">{t('settings.changePassword')}</p>
                <div>
                  <label className="block text-[10px] font-medium text-slate-400 mb-1">{t('settings.oldPassword')}</label>
                  <input 
                    type="password"
                    required
                    className="w-full bg-slate-950 border border-slate-800 rounded px-2.5 py-1.5 text-xs text-slate-100 focus:outline-none"
                    value={oldPassword}
                    onChange={(e) => setOldPassword(e.target.value)}
                  />
                </div>
                <div>
                  <label className="block text-[10px] font-medium text-slate-400 mb-1">{t('settings.newPassword')}</label>
                  <input 
                    type="password"
                    required
                    className="w-full bg-slate-950 border border-slate-800 rounded px-2.5 py-1.5 text-xs text-slate-100 focus:outline-none"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                  />
                </div>

                {pwdMessage && <p className="text-xs text-emerald-400 font-mono text-center">{pwdMessage}</p>}
                {pwdError && <p className="text-xs text-rose-400 font-mono text-center">{pwdError}</p>}

                <button type="submit" className="px-4 py-1.5 bg-indigo-600 hover:bg-indigo-500 rounded text-xs text-white font-medium">{t('settings.updateBtn')}</button>
              </form>

              {/* Account Deletion */}
              <div className="pt-4 border-t border-slate-850">
                <p className="text-xs font-semibold text-rose-400">Vùng nguy hiểm</p>
                <p className="text-[10px] text-slate-500 mt-1">Xóa vĩnh viễn tài khoản thành viên và tất cả các tệp dữ liệu đã tải lên.</p>
                <button 
                  onClick={handleDeleteAccount}
                  className="mt-3 px-4 py-2 bg-rose-600/10 border border-rose-500/25 hover:bg-rose-600 text-rose-400 hover:text-white rounded text-xs font-bold flex items-center gap-1.5 transition-all"
                >
                  <Trash2 className="w-3.5 h-3.5" /> Xóa tài khoản
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* MULTI UPLOAD MODAL */}
      {showUploadModal && (
        <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4" role="dialog" aria-modal="true">
          <div className="w-full max-w-lg bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-2xl relative">
            <button onClick={() => setShowUploadModal(false)} className="absolute top-4 right-4 text-slate-400 hover:text-white" aria-label="Đóng">
              <X className="w-4 h-4" />
            </button>
            
            <h3 className="text-sm font-semibold text-white uppercase tracking-wider mb-5 flex items-center gap-2">
              <FileText className="w-4 h-4 text-indigo-400" /> Tải lên tài liệu và phân tích
            </h3>

            <div className="space-y-5">
              <div 
                onClick={() => fileInputRef.current?.click()}
                className="py-12 border-2 border-slate-800 border-dashed hover:border-indigo-500/50 rounded-xl bg-slate-950/40 text-center cursor-pointer transition-all"
              >
                <Camera className="w-8 h-8 text-indigo-400 mx-auto mb-3" />
                <span className="text-xs text-slate-400 block font-medium">Click để chọn hoặc kéo thả các tệp tài liệu hỗ trợ</span>
                <span className="text-[10px] text-slate-600 block mt-1">Hỗ trợ PDF, DOCX, DOC, XLSX, XLS, PPTX, CSV... (Chuyển đổi tự động qua MarkItDown)</span>
                <input 
                  ref={fileInputRef}
                  type="file"
                  multiple
                  className="hidden" 
                  onChange={(e) => setUploadFiles(e.target.files)} 
                />
              </div>

              {uploadFiles && uploadFiles.length > 0 && (
                <div className="space-y-2">
                  <p className="text-xs text-slate-400 font-semibold">Tệp tin đã chọn ({uploadFiles.length}):</p>
                  <div className="max-h-32 overflow-y-auto space-y-1.5 pr-2">
                    {Array.from(uploadFiles).map((file, idx) => (
                      <div key={idx} className="flex justify-between items-center p-2 rounded bg-slate-950 border border-slate-800/60 text-xs">
                        <span className="font-mono truncate text-slate-300">{file.name}</span>
                        <span className="text-[10px] text-slate-500">{(file.size / 1024).toFixed(0)} KB</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {extracting ? (
                <div className="py-6 text-center text-xs text-slate-400 space-y-2 flex flex-col items-center">
                  <RefreshCw className="w-6 h-6 animate-spin text-indigo-500" />
                  <span className="font-mono">Reranking AI đang phân tích thứ tự recency/relevance tài liệu...</span>
                  <span className="text-[10px] text-slate-600">Gemini đang thực hiện trích xuất dữ liệu có cấu trúc...</span>
                </div>
              ) : (
                <button 
                  onClick={handleMultiUpload}
                  disabled={!uploadFiles}
                  className={`w-full py-2.5 rounded-lg text-xs font-bold transition-all text-white ${
                    uploadFiles ? 'bg-indigo-600 hover:bg-indigo-500 shadow-md shadow-indigo-600/10' : 'bg-slate-800 text-slate-500 cursor-not-allowed'
                  }`}
                >
                  Bắt đầu trích xuất AI
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* SYNC MODAL */}
      {showSyncModal && (
        <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4" role="dialog" aria-modal="true">
          <div className="w-full max-w-lg bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-2xl relative">
            <button onClick={() => setShowSyncModal(false)} className="absolute top-4 right-4 text-slate-400 hover:text-white" aria-label="Đóng">
              <X className="w-4 h-4" />
            </button>
            <h3 className="text-sm font-semibold text-white uppercase tracking-wider mb-5 flex items-center gap-2">
              <RefreshCw className="w-4 h-4 text-indigo-400" /> Đồng bộ văn bản pháp lý (Ingestion Sync)
            </h3>

            <div className="space-y-4">
              <p className="text-xs text-slate-400 leading-relaxed">
                Hệ thống quét thư mục incoming pháp lý và kiểm tra hàm băm SHA-256 để phát hiện sự thay đổi, tự động tạo cảnh báo và phân tích lại.
              </p>

              <button 
                onClick={handleSyncPolicies}
                disabled={syncing}
                className="w-full py-2 bg-indigo-600 hover:bg-indigo-500 rounded text-xs font-semibold text-white transition-all"
              >
                {syncing ? 'Đang thực hiện Ingestion...' : 'Kích hoạt quét đồng bộ pháp lý'}
              </button>

              {syncLogs.length > 0 && (
                <div className="space-y-2">
                  <p className="text-xs text-slate-400 font-bold">Kết quả đồng bộ:</p>
                  <div className="bg-slate-950 border border-slate-800 p-3 rounded-lg max-h-40 overflow-y-auto font-mono text-[10px] text-slate-300 space-y-1">
                    {syncLogs.map((log, idx) => <p key={idx}>{log}</p>)}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* AUDIT LOGS MODAL */}
      {showAuditModal && (
        <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4" role="dialog" aria-modal="true">
          <div className="w-full max-w-2xl bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-2xl relative">
            <button onClick={() => setShowAuditModal(false)} className="absolute top-4 right-4 text-slate-400 hover:text-white" aria-label="Đóng">
              <X className="w-4 h-4" />
            </button>
            <h3 className="text-sm font-semibold text-white uppercase tracking-wider mb-5 flex items-center gap-2">
              <History className="w-4 h-4 text-indigo-400" /> Lịch sử nhật ký hoạt động (Audit trail)
            </h3>

            <div className="max-h-96 overflow-y-auto space-y-2.5 pr-2">
              {auditLogs.length > 0 ? (
                auditLogs.map((log) => (
                  <div key={log.id} className="p-3 bg-slate-950 border border-slate-850/60 rounded-xl text-left space-y-1.5">
                    <div className="flex justify-between items-center text-[10px]">
                      <span className="font-bold text-indigo-400 font-mono">{log.event_type}</span>
                      <span className="text-slate-500 font-mono">{new Date(log.timestamp).toLocaleString()}</span>
                    </div>
                    <div className="text-xs space-y-1">
                      <p className="text-slate-400">Đối tượng ID: <span className="font-mono text-slate-300">{log.target_id}</span></p>
                      {log.field_name && <p className="text-slate-400">Trường thông tin: <span className="font-semibold text-white">{formatFieldName(log.field_name)}</span></p>}
                      <div className="flex items-center gap-1.5 mt-1 text-[11px]">
                        <span className="text-rose-400 font-mono font-semibold">{log.old_value || 'EMPTY'}</span>
                        <span className="text-slate-600">→</span>
                        <span className="text-emerald-400 font-mono font-semibold">{log.new_value || 'EMPTY'}</span>
                      </div>
                    </div>
                  </div>
                ))
              ) : (
                <div className="py-12 text-center text-xs text-slate-500">
                  Không có nhật ký hoạt động nào được ghi lại.
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* CONFLICT RESOLUTION MODAL */}
      {showConflictModal && conflictFieldName && conflictFieldData && (
        <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4" role="dialog" aria-modal="true">
          <div className="w-full max-w-lg bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-2xl relative">
            <button onClick={() => setShowConflictModal(false)} className="absolute top-4 right-4 text-slate-400 hover:text-white" aria-label="Đóng">
              <X className="w-4 h-4" />
            </button>
            <h3 className="text-sm font-semibold text-white uppercase tracking-wider mb-5 flex items-center gap-2">
              <AlertTriangle className="w-4 h-4 text-amber-500 animate-pulse" /> Giải quyết mâu thuẫn dữ liệu ({formatFieldName(conflictFieldName)})
            </h3>

            <div className="space-y-4">
              <p className="text-xs text-slate-400 leading-relaxed">
                Phát hiện mâu thuẫn giữa các tệp tài liệu khác nhau. Vui lòng chọn giá trị chính xác làm thông tin chính thức:
              </p>

              <div className="space-y-3">
                {conflictFieldData.conflicts?.map((cf: any, idx: number) => (
                  <div 
                    key={idx} 
                    onClick={() => handleResolveConflict(conflictFieldName, cf.value)}
                    className="p-3 bg-slate-950 border border-slate-800 hover:border-indigo-500/50 rounded-xl text-left cursor-pointer transition-all space-y-1.5"
                  >
                    <div className="flex justify-between items-center text-[10px]">
                      <span className="text-indigo-400 font-semibold">{cf.source_type} ({cf.source_uri})</span>
                      <span className="text-slate-500 font-mono">Tự tin: {cf.confidence}</span>
                    </div>
                    <p className="text-sm font-bold text-white">{String(cf.value)}</p>
                    {cf.evidence_quote && <p className="text-xs text-slate-400 italic font-serif">"{cf.evidence_quote}"</p>}
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}
