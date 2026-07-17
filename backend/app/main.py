import os
import uuid
import json
import shutil
import hashlib
from datetime import datetime, timedelta
from fastapi import FastAPI, HTTPException, status, Query, Header, Depends, UploadFile, File
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse, StreamingResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel, Field
from typing import Optional, List, Any, Dict

from app.engine.db import get_db_connection, init_db
from app.schemas.passport import CompanyPassport
from app.schemas.policy import PolicyOpportunity
from app.engine.retrieval import HybridRetrievalEngine
from app.engine.rule_evaluator import evaluate_rule_group
from app.engine.auth import hash_password, verify_password
from app.pipeline.markdown_converter import convert_to_markdown_local
from app.pipeline.extractor import rank_documents_for_field, call_gemini_extraction

from docxtpl import DocxTemplate

app = FastAPI(
    title="P2B (Policy-to-Business) API",
    description="AI-native platform helping Vietnamese businesses navigate legal policies and incentives.",
    version="1.0.0"
)

@app.on_event("startup")
def on_startup():
    init_db()

# CORS configuration - strict origin whitelist
ALLOWED_ORIGINS = [
    "http://localhost:5173",
    "http://127.0.0.1:5173",
    "https://frontend-henna-nu-49.vercel.app"
]

app.add_middleware(
    CORSMiddleware,
    allow_origins=ALLOWED_ORIGINS,
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "DELETE", "OPTIONS"],
    allow_headers=["*"],
)

# Directory Setup
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
SEED_DIR = os.path.join(BASE_DIR, "seed")
UPLOADS_DIR = os.path.join(SEED_DIR, "uploads")
AVATARS_DIR = os.path.join(SEED_DIR, "avatars")
INCOMING_DIR = os.path.join(SEED_DIR, "legal_corpus_incoming")

for d in [UPLOADS_DIR, AVATARS_DIR, INCOMING_DIR]:
    os.makedirs(d, exist_ok=True)

# Mount avatars static files
app.mount("/static/avatars", StaticFiles(directory=AVATARS_DIR), name="avatars")

retrieval_engine = HybridRetrievalEngine(SEED_DIR)

# Request / Response Schemas
class SignupRequest(BaseModel):
    email: str
    password: str
    user_type: str = "COMPANY_MANAGER" # COMPANY_MANAGER or INDIVIDUAL

class LoginRequest(BaseModel):
    email: str
    password: str

class PasswordChangeRequest(BaseModel):
    old_password: str
    new_password: str

class UserModeUpdateRequest(BaseModel):
    user_type: str # COMPANY_MANAGER or INDIVIDUAL

class DraftCreateRequest(BaseModel):
    policy_id: str

class DraftStatusUpdateRequest(BaseModel):
    status: str # APPROVED, REJECTED
    reviewer_comments: Optional[str] = ""

class PassportUpdateRequest(BaseModel):
    passport_data: dict

# Dependency: Get Current User from Session Token
def get_current_user(authorization: Optional[str] = Header(None)) -> Dict[str, Any]:
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Missing or invalid Authorization header"
        )
    token = authorization.split(" ")[1]
    
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("""
        SELECT u.id, u.email, u.user_type, u.company_id, u.personal_passport_id, u.avatar_path
        FROM sessions s
        JOIN users u ON s.user_id = u.id
        WHERE s.token = ? AND s.expires_at > ?
    """, (token, datetime.utcnow().isoformat()))
    user_row = cursor.fetchone()
    conn.close()
    
    if not user_row:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Session invalid or expired"
        )
        
    return {
        "id": user_row["id"],
        "email": user_row["email"],
        "user_type": user_row["user_type"],
        "company_id": user_row["company_id"],
        "personal_passport_id": user_row["personal_passport_id"],
        "avatar_path": user_row["avatar_path"]
    }

# ================= AUTHENTICATION ENDPOINTS =================

@app.post("/api/v1/auth/signup")
def signup(req: SignupRequest):
    if req.user_type not in ["COMPANY_MANAGER", "INDIVIDUAL"]:
        raise HTTPException(status_code=400, detail="Invalid user_type. Must be COMPANY_MANAGER or INDIVIDUAL")
        
    conn = get_db_connection()
    cursor = conn.cursor()
    
    # Check if user already exists
    cursor.execute("SELECT id FROM users WHERE email = ?", (req.email,))
    if cursor.fetchone():
        conn.close()
        raise HTTPException(status_code=400, detail="Email already registered")
        
    user_id = str(uuid.uuid4())
    hashed_pwd = hash_password(req.password)
    timestamp = datetime.utcnow().isoformat()
    
    company_id = None
    personal_passport_id = None
    
    if req.user_type == "COMPANY_MANAGER":
        # Create a new empty company passport
        company_id = f"company_{uuid.uuid4().hex[:8]}"
        empty_passport = {
            "company_name": {"value": "", "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "tax_code": {"value": "", "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "industry": {"value": "", "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "location": {"value": "", "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "employee_count": {"value": 0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "rd_spend_ratio": {"value": 0.0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "revenue": {"value": 0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "registered_capital": {"value": 0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "metadata": {}
        }
        cursor.execute(
            "INSERT INTO company_passports (id, data_json, updated_at) VALUES (?, ?, ?)",
            (company_id, json.dumps(empty_passport, ensure_ascii=False), timestamp)
        )
    else:
        # Create empty personal passport
        personal_passport_id = f"personal_{uuid.uuid4().hex[:8]}"
        cursor.execute(
            """
            INSERT INTO personal_passports (id, full_name, birth_year, location, occupation, degree, monthly_income, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (personal_passport_id, "", 0, "", "", "", 0, timestamp)
        )
        
    cursor.execute(
        """
        INSERT INTO users (id, email, hashed_password, user_type, company_id, personal_passport_id, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        """,
        (user_id, req.email, hashed_pwd, req.user_type, company_id, personal_passport_id, timestamp)
    )
    
    conn.commit()
    conn.close()
    
    return {"message": "Signup successful", "user_id": user_id}

@app.post("/api/v1/auth/login")
def login(req: LoginRequest):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    cursor.execute("SELECT id, hashed_password, user_type, company_id, personal_passport_id FROM users WHERE email = ?", (req.email,))
    user_row = cursor.fetchone()
    if not user_row:
        conn.close()
        raise HTTPException(status_code=400, detail="Invalid email or password")
        
    if not verify_password(req.password, user_row["hashed_password"]):
        conn.close()
        raise HTTPException(status_code=400, detail="Invalid email or password")
        
    # Generate session token
    token = str(uuid.uuid4())
    expires_at = (datetime.utcnow() + timedelta(days=7)).isoformat()
    created_at = datetime.utcnow().isoformat()
    
    cursor.execute(
        "INSERT INTO sessions (token, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)",
        (token, user_row["id"], created_at, expires_at)
    )
    
    conn.commit()
    conn.close()
    
    return {
        "token": token,
        "user_type": user_row["user_type"],
        "company_id": user_row["company_id"],
        "personal_passport_id": user_row["personal_passport_id"]
    }

@app.post("/api/v1/auth/logout")
def logout(authorization: Optional[str] = Header(None)):
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(status_code=400, detail="Invalid Authorization header")
    token = authorization.split(" ")[1]
    
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("DELETE FROM sessions WHERE token = ?", (token,))
    conn.commit()
    conn.close()
    
    return {"message": "Logout successful"}

# ================= USER SETTINGS ENDPOINTS =================

@app.get("/api/v1/users/me")
def get_me(user: Dict[str, Any] = Depends(get_current_user)):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    passport_data = None
    if user["user_type"] == "COMPANY_MANAGER":
        cursor.execute("SELECT data_json FROM company_passports WHERE id = ?", (user["company_id"],))
        row = cursor.fetchone()
        if row:
            passport_data = json.loads(row["data_json"])
    else:
        cursor.execute("SELECT id, full_name, birth_year, location, occupation, degree, monthly_income FROM personal_passports WHERE id = ?", (user["personal_passport_id"],))
        row = cursor.fetchone()
        if row:
            passport_data = {
                "full_name": row["full_name"],
                "birth_year": row["birth_year"],
                "location": row["location"],
                "occupation": row["occupation"],
                "degree": row["degree"],
                "monthly_income": row["monthly_income"]
            }
            
    conn.close()
    return {
        "user": user,
        "passport": passport_data
    }

@app.post("/api/v1/users/avatar")
def upload_avatar(file: UploadFile = File(...), user: Dict[str, Any] = Depends(get_current_user)):
    _, ext = os.path.splitext(file.filename.lower())
    if ext not in ['.png', '.jpg', '.jpeg', '.gif']:
        raise HTTPException(status_code=400, detail="Only images allowed (.png, .jpg, .jpeg, .gif)")
        
    avatar_filename = f"{user['id']}{ext}"
    avatar_path = os.path.join(AVATARS_DIR, avatar_filename)
    
    with open(avatar_path, "wb") as buffer:
        shutil.copyfileobj(file.file, buffer)
        
    # Update avatar path in DB
    avatar_url = f"/static/avatars/{avatar_filename}"
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("UPDATE users SET avatar_path = ? WHERE id = ?", (avatar_url, user["id"]))
    conn.commit()
    conn.close()
    
    return {"avatar_url": avatar_url}

@app.put("/api/v1/users/change-password")
def change_password(req: PasswordChangeRequest, user: Dict[str, Any] = Depends(get_current_user)):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    cursor.execute("SELECT hashed_password FROM users WHERE id = ?", (user["id"],))
    row = cursor.fetchone()
    if not row or not verify_password(req.old_password, row["hashed_password"]):
        conn.close()
        raise HTTPException(status_code=400, detail="Previous password is incorrect")
        
    hashed_new = hash_password(req.new_password)
    cursor.execute("UPDATE users SET hashed_password = ? WHERE id = ?", (hashed_new, user["id"]))
    conn.commit()
    conn.close()
    
    return {"message": "Password changed successfully"}

@app.put("/api/v1/users/mode")
def update_user_mode(req: UserModeUpdateRequest, user: Dict[str, Any] = Depends(get_current_user)):
    if req.user_type not in ["COMPANY_MANAGER", "INDIVIDUAL"]:
        raise HTTPException(status_code=400, detail="Invalid user_type")
        
    conn = get_db_connection()
    cursor = conn.cursor()
    
    # Check if they already have the target passport initialized
    cursor.execute("SELECT company_id, personal_passport_id FROM users WHERE id = ?", (user["id"],))
    row = cursor.fetchone()
    
    company_id = row["company_id"]
    personal_passport_id = row["personal_passport_id"]
    timestamp = datetime.utcnow().isoformat()
    
    if req.user_type == "COMPANY_MANAGER" and not company_id:
        company_id = f"company_{uuid.uuid4().hex[:8]}"
        empty_passport = {
            "company_name": {"value": "", "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "tax_code": {"value": "", "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "industry": {"value": "", "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "location": {"value": "", "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "employee_count": {"value": 0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "rd_spend_ratio": {"value": 0.0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "revenue": {"value": 0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "registered_capital": {"value": 0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            "metadata": {}
        }
        cursor.execute("INSERT INTO company_passports (id, data_json, updated_at) VALUES (?, ?, ?)", (company_id, json.dumps(empty_passport), timestamp))
        cursor.execute("UPDATE users SET company_id = ? WHERE id = ?", (company_id, user["id"]))
        
    elif req.user_type == "INDIVIDUAL" and not personal_passport_id:
        personal_passport_id = f"personal_{uuid.uuid4().hex[:8]}"
        cursor.execute(
            "INSERT INTO personal_passports (id, full_name, birth_year, location, occupation, degree, monthly_income, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
            (personal_passport_id, "", 0, "", "", "", 0, timestamp)
        )
        cursor.execute("UPDATE users SET personal_passport_id = ? WHERE id = ?", (personal_passport_id, user["id"]))
        
    cursor.execute("UPDATE users SET user_type = ? WHERE id = ?", (req.user_type, user["id"]))
    conn.commit()
    conn.close()
    
    return {"message": "User mode updated successfully", "user_type": req.user_type}

@app.delete("/api/v1/users")
def delete_user(user: Dict[str, Any] = Depends(get_current_user)):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    # 1. Delete associated passports
    if user["company_id"]:
        cursor.execute("DELETE FROM company_passports WHERE id = ?", (user["company_id"],))
    if user["personal_passport_id"]:
        cursor.execute("DELETE FROM personal_passports WHERE id = ?", (user["personal_passport_id"],))
        
    # 2. Delete user row (cascades sessions via FK)
    cursor.execute("DELETE FROM users WHERE id = ?", (user["id"],))
    
    # 3. Clean up avatar file if exists
    if user["avatar_path"]:
        filename = os.path.basename(user["avatar_path"])
        file_path = os.path.join(AVATARS_DIR, filename)
        if os.path.exists(file_path):
            try:
                os.remove(file_path)
            except Exception:
                pass
                
    conn.commit()
    conn.close()
    
    return {"message": "User account and all associated data deleted successfully"}

# ================= COMPANY PASSPORT & PROVENANCE ENDPOINTS =================

@app.get("/api/v1/passports/{id}")
def get_passport(id: str, user: Dict[str, Any] = Depends(get_current_user)):
    # Tenant boundary checks
    if user["user_type"] == "COMPANY_MANAGER" and user["company_id"] != id:
        raise HTTPException(status_code=403, detail="Forbidden: You do not have access to this company passport")
        
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT id, data_json, updated_at FROM company_passports WHERE id = ?", (id,))
    row = cursor.fetchone()
    conn.close()
    
    if not row:
        raise HTTPException(status_code=404, detail="Company Passport not found")
        
    return {
        "id": row["id"],
        "data": json.loads(row["data_json"]),
        "updated_at": row["updated_at"]
    }

@app.put("/api/v1/passports/{id}")
def update_passport(id: str, req: PassportUpdateRequest, user: Dict[str, Any] = Depends(get_current_user)):
    if user["user_type"] == "COMPANY_MANAGER" and user["company_id"] != id:
        raise HTTPException(status_code=403, detail="Forbidden: You do not have access to this company passport")
        
    conn = get_db_connection()
    cursor = conn.cursor()
    
    cursor.execute("SELECT data_json FROM company_passports WHERE id = ?", (id,))
    row = cursor.fetchone()
    if not row:
        conn.close()
        raise HTTPException(status_code=404, detail="Company Passport not found")
        
    old_data = json.loads(row["data_json"])
    new_data = req.passport_data
    
    try:
        CompanyPassport(**new_data)
    except Exception as e:
        conn.close()
        raise HTTPException(status_code=422, detail=f"Invalid CompanyPassport: {str(e)}")
        
    timestamp = datetime.utcnow().isoformat()
    for field_name, new_prov in new_data.items():
        if field_name == "metadata":
            continue
            
        old_val = old_data.get(field_name, {}).get("value")
        new_val = new_prov.get("value")
        
        if old_val != new_val:
            cursor.execute(
                """
                INSERT INTO audit_logs (event_type, target_id, field_name, old_value, new_value, timestamp)
                VALUES (?, ?, ?, ?, ?, ?)
                """,
                ("PASSPORT_EDIT", id, field_name, str(old_val), str(new_val), timestamp)
            )
            new_prov["status"] = "USER_CONFIRMED"
            new_prov["source_type"] = "MANUAL_INPUT"
            new_prov["observed_at"] = timestamp
            
    cursor.execute("UPDATE company_passports SET data_json = ?, updated_at = ? WHERE id = ?", (json.dumps(new_data, ensure_ascii=False), timestamp, id))
    conn.commit()
    conn.close()
    
    return {"message": "Company Passport updated successfully", "data": new_data}

# Endpoint to update personal passport directly
@app.put("/api/v1/personal-passports/me")
def update_personal_passport(passport_data: Dict[str, Any], user: Dict[str, Any] = Depends(get_current_user)):
    if user["user_type"] != "INDIVIDUAL":
        raise HTTPException(status_code=400, detail="Only individual users can edit personal passports")
        
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute(
        """
        UPDATE personal_passports 
        SET full_name = ?, birth_year = ?, location = ?, occupation = ?, degree = ?, monthly_income = ?, updated_at = ?
        WHERE id = ?
        """,
        (
            passport_data.get("full_name", ""),
            int(passport_data.get("birth_year", 0)),
            passport_data.get("location", ""),
            passport_data.get("occupation", ""),
            passport_data.get("degree", ""),
            int(passport_data.get("monthly_income", 0)),
            datetime.utcnow().isoformat(),
            user["personal_passport_id"]
        )
    )
    conn.commit()
    conn.close()
    return {"message": "Personal Passport updated successfully"}

# ================= HYBRID RAG RETRIEVAL ENDPOINTS =================

@app.get("/api/v1/policies")
def list_policies(query: Optional[str] = "", user: Dict[str, Any] = Depends(get_current_user)):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    # 1. Fetch appropriate passport representation based on user type
    if user["user_type"] == "COMPANY_MANAGER":
        cursor.execute("SELECT data_json FROM company_passports WHERE id = ?", (user["company_id"],))
        row = cursor.fetchone()
        conn.close()
        if not row:
            raise HTTPException(status_code=404, detail="Company Passport not found")
        passport = CompanyPassport(**json.loads(row["data_json"]))
    else:
        # Wrap PersonalPassport as a simplified mock CompanyPassport to satisfy the retriever schema
        cursor.execute("SELECT full_name, location, occupation, monthly_income FROM personal_passports WHERE id = ?", (user["personal_passport_id"],))
        row = cursor.fetchone()
        conn.close()
        if not row:
            raise HTTPException(status_code=404, detail="Personal Passport not found")
            
        timestamp = datetime.utcnow().isoformat()
        passport = CompanyPassport(
            company_name={"value": row["full_name"], "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            tax_code={"value": "", "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            industry={"value": row["occupation"], "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            location={"value": row["location"], "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            employee_count={"value": 1, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            rd_spend_ratio={"value": 0.0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            revenue={"value": row["monthly_income"] * 12, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            registered_capital={"value": 0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            metadata={}
        )
        
    results = retrieval_engine.retrieve(passport, query or "chính sách hỗ trợ", top_n=10)
    
    # Filter policies: in individual mode, prioritize individual policies
    formatted_results = []
    for r in results:
        opp = r["opportunity"]
        formatted_results.append({
            "opportunity_id": opp.id,
            "title": opp.title,
            "benefits": opp.benefits,
            "target_companies": opp.target_companies,
            "geography": opp.geography,
            "deadline": opp.deadline,
            "required_documents": opp.required_documents,
            "source_legal_documents": opp.source_legal_documents,
            "score": r["score"],
            "bm25_score": r["bm25_score"],
            "vector_score": r["vector_score"],
            "metadata_score": r["metadata_score"]
        })
        
    return formatted_results

@app.get("/api/v1/policies/{id}")
def get_policy(id: str, user: Dict[str, Any] = Depends(get_current_user)):
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT data_json FROM policy_opportunities WHERE id = ?", (id,))
    row = cursor.fetchone()
    conn.close()
    
    if not row:
        raise HTTPException(status_code=404, detail="Policy Opportunity not found")
        
    return json.loads(row["data_json"])

# ================= ELIGIBILITY ENGINE ENDPOINTS =================

@app.post("/api/v1/eligibility")
def run_eligibility_verification(req: DraftCreateRequest, user: Dict[str, Any] = Depends(get_current_user)):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    # Get appropriate passport
    if user["user_type"] == "COMPANY_MANAGER":
        cursor.execute("SELECT data_json FROM company_passports WHERE id = ?", (user["company_id"],))
        p_row = cursor.fetchone()
        if not p_row:
            conn.close()
            raise HTTPException(status_code=404, detail="Company Passport not found")
        passport = CompanyPassport(**json.loads(p_row["data_json"]))
    else:
        cursor.execute("SELECT full_name, location, occupation, monthly_income FROM personal_passports WHERE id = ?", (user["personal_passport_id"],))
        p_row = cursor.fetchone()
        if not p_row:
            conn.close()
            raise HTTPException(status_code=404, detail="Personal Passport not found")
        timestamp = datetime.utcnow().isoformat()
        passport = CompanyPassport(
            company_name={"value": p_row["full_name"], "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            tax_code={"value": "", "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            industry={"value": p_row["occupation"], "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            location={"value": p_row["location"], "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            employee_count={"value": 1, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            rd_spend_ratio={"value": 0.0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            revenue={"value": p_row["monthly_income"] * 12, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            registered_capital={"value": 0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            metadata={}
        )
        
    cursor.execute("SELECT data_json FROM policy_opportunities WHERE id = ?", (req.policy_id,))
    opp_row = cursor.fetchone()
    conn.close()
    
    if not opp_row:
        raise HTTPException(status_code=404, detail="Policy Opportunity not found")
        
    opp = PolicyOpportunity(**json.loads(opp_row["data_json"]))
    status_result, details = evaluate_rule_group(passport, opp.eligibility_rules)
    
    return {
        "status": status_result,
        "details": details
    }

# ================= DRAFT & REVIEW (HITL) ENDPOINTS =================

@app.post("/api/v1/drafts")
def create_draft(req: DraftCreateRequest, user: Dict[str, Any] = Depends(get_current_user)):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    company_id = user["company_id"] or user["personal_passport_id"]
    
    # 1. Fetch passport & policy
    if user["user_type"] == "COMPANY_MANAGER":
        cursor.execute("SELECT data_json FROM company_passports WHERE id = ?", (company_id,))
        p_row = cursor.fetchone()
        if not p_row:
            conn.close()
            raise HTTPException(status_code=404, detail="Company Passport not found")
        passport = CompanyPassport(**json.loads(p_row["data_json"]))
    else:
        cursor.execute("SELECT full_name, location, occupation, monthly_income FROM personal_passports WHERE id = ?", (company_id,))
        p_row = cursor.fetchone()
        if not p_row:
            conn.close()
            raise HTTPException(status_code=404, detail="Personal Passport not found")
        timestamp = datetime.utcnow().isoformat()
        passport = CompanyPassport(
            company_name={"value": p_row["full_name"], "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            tax_code={"value": "", "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            industry={"value": p_row["occupation"], "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            location={"value": p_row["location"], "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            employee_count={"value": 1, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            rd_spend_ratio={"value": 0.0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            revenue={"value": p_row["monthly_income"] * 12, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            registered_capital={"value": 0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            metadata={}
        )
        
    cursor.execute("SELECT data_json FROM policy_opportunities WHERE id = ?", (req.policy_id,))
    opp_row = cursor.fetchone()
    if not opp_row:
        conn.close()
        raise HTTPException(status_code=404, detail="Policy Opportunity not found")
        
    opp = PolicyOpportunity(**json.loads(opp_row["data_json"]))
    
    # 2. Evaluate rules
    status_result, details = evaluate_rule_group(passport, opp.eligibility_rules)
    
    # 3. Create Draft record
    draft_id = str(uuid.uuid4())
    timestamp = datetime.utcnow().isoformat()
    
    cursor.execute(
        """
        INSERT INTO drafts (id, company_id, opportunity_id, status, details_json, reviewer_comments, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        """,
        (
            draft_id,
            company_id,
            req.policy_id,
            "PENDING_REVIEW",
            json.dumps(details, ensure_ascii=False),
            "",
            timestamp,
            timestamp
        )
    )
    
    conn.commit()
    conn.close()
    return {"draft_id": draft_id, "status": "PENDING_REVIEW"}

@app.get("/api/v1/drafts")
def list_drafts(user: Dict[str, Any] = Depends(get_current_user)):
    company_id = user["company_id"] or user["personal_passport_id"]
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT id, company_id, opportunity_id, status, reviewer_comments, created_at, updated_at FROM drafts WHERE company_id = ?", (company_id,))
    rows = cursor.fetchall()
    conn.close()
    
    result = []
    for r in rows:
        result.append({
            "id": r["id"],
            "company_id": r["company_id"],
            "opportunity_id": r["opportunity_id"],
            "status": r["status"],
            "reviewer_comments": r["reviewer_comments"],
            "created_at": r["created_at"],
            "updated_at": r["updated_at"]
        })
    return result

@app.get("/api/v1/drafts/{id}")
def get_draft(id: str, user: Dict[str, Any] = Depends(get_current_user)):
    company_id = user["company_id"] or user["personal_passport_id"]
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT id, company_id, opportunity_id, status, details_json, reviewer_comments, created_at, updated_at FROM drafts WHERE id = ? AND company_id = ?", (id, company_id))
    row = cursor.fetchone()
    conn.close()
    
    if not row:
        raise HTTPException(status_code=404, detail="Draft not found or access forbidden")
        
    return {
        "id": row["id"],
        "company_id": row["company_id"],
        "opportunity_id": row["opportunity_id"],
        "status": row["status"],
        "details": json.loads(row["details_json"]),
        "reviewer_comments": row["reviewer_comments"],
        "created_at": row["created_at"],
        "updated_at": row["updated_at"]
    }

@app.put("/api/v1/drafts/{id}/status")
def update_draft_status(id: str, req: DraftStatusUpdateRequest, user: Dict[str, Any] = Depends(get_current_user)):
    company_id = user["company_id"] or user["personal_passport_id"]
    
    if req.status not in ["APPROVED", "REJECTED"]:
        raise HTTPException(status_code=400, detail="Invalid transition status. Must be APPROVED or REJECTED")
        
    conn = get_db_connection()
    cursor = conn.cursor()
    
    cursor.execute("SELECT company_id, opportunity_id, status FROM drafts WHERE id = ? AND company_id = ?", (id, company_id))
    row = cursor.fetchone()
    if not row:
        conn.close()
        raise HTTPException(status_code=404, detail="Draft not found or access forbidden")
        
    opportunity_id = row["opportunity_id"]
    old_status = row["status"]
    
    if old_status != "PENDING_REVIEW":
        conn.close()
        raise HTTPException(status_code=400, detail=f"Cannot transition draft in status '{old_status}'")
        
    # GATED REVIEW GATEWAY: Run strict server validation
    # Fetch passport & policy to verify MET status
    if user["user_type"] == "COMPANY_MANAGER":
        cursor.execute("SELECT data_json FROM company_passports WHERE id = ?", (company_id,))
        p_row = cursor.fetchone()
        passport = CompanyPassport(**json.loads(p_row["data_json"]))
    else:
        cursor.execute("SELECT full_name, location, occupation, monthly_income FROM personal_passports WHERE id = ?", (company_id,))
        p_row = cursor.fetchone()
        timestamp = datetime.utcnow().isoformat()
        passport = CompanyPassport(
            company_name={"value": p_row["full_name"], "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            tax_code={"value": "", "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            industry={"value": p_row["occupation"], "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            location={"value": p_row["location"], "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            employee_count={"value": 1, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            rd_spend_ratio={"value": 0.0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            revenue={"value": p_row["monthly_income"] * 12, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "HIGH", "status": "EXTRACTED", "conflicts": []},
            registered_capital={"value": 0, "source_type": "MANUAL_INPUT", "source_uri": "", "source_location": "", "evidence_quote": "", "observed_at": timestamp, "confidence": "LOW", "status": "MISSING", "conflicts": []},
            metadata={}
        )
        
    cursor.execute("SELECT data_json FROM policy_opportunities WHERE id = ?", (opportunity_id,))
    opp_row = cursor.fetchone()
    opp = PolicyOpportunity(**json.loads(opp_row["data_json"]))
    
    eligibility_status, details = evaluate_rule_group(passport, opp.eligibility_rules)
    
    if req.status == "APPROVED":
        # Block if eligibility status is NOT MET or MISSING
        if eligibility_status != "MET":
            conn.close()
            raise HTTPException(
                status_code=400,
                detail=f"Cannot approve draft. Eligibility evaluation returned: '{eligibility_status}'"
            )
            
        # Block if any checked fields have conflicts or missing statuses
        conflicted_fields = []
        missing_fields = []
        
        def extract_rules_from_details(g_details: dict) -> list:
            flat = []
            if "rules" in g_details:
                for r in g_details["rules"]:
                    if "rules" in r:
                        flat.extend(extract_rules_from_details(r))
                    else:
                        flat.append(r)
            return flat

        flat_rules = extract_rules_from_details(details)
        for r_check in flat_rules:
            # Match passport fields used
            field_name = r_check.get("field")
            if field_name and hasattr(passport, field_name):
                f_provenance = getattr(passport, field_name)
                if f_provenance.status == "CONFLICTED":
                    conflicted_fields.append(field_name)
                elif f_provenance.status == "MISSING":
                    missing_fields.append(field_name)
                    
        if conflicted_fields or missing_fields:
            conn.close()
            raise HTTPException(
                status_code=400,
                detail=f"Cannot approve draft due to unresolved conflicts in fields {conflicted_fields} or missing fields {missing_fields}"
            )
            
    timestamp = datetime.utcnow().isoformat()
    new_status = req.status
    
    # Save IMMUTABLE SNAPSHOT to the draft details row on approval
    cursor.execute(
        "UPDATE drafts SET status = ?, reviewer_comments = ?, details_json = ?, updated_at = ? WHERE id = ?",
        (new_status, req.reviewer_comments, json.dumps(details, ensure_ascii=False), timestamp, id)
    )
    
    cursor.execute(
        """
        INSERT INTO audit_logs (event_type, target_id, field_name, old_value, new_value, timestamp)
        VALUES (?, ?, ?, ?, ?, ?)
        """,
        ("DRAFT_STATUS_CHANGE", id, "status", old_status, new_status, timestamp)
    )
    
    # Fill Document using the immutable snapshot data
    if new_status == "APPROVED":
        template_path = os.path.join(SEED_DIR, "grant_template.docx")
        if os.path.exists(template_path):
            try:
                doc = DocxTemplate(template_path)
                context = {
                    "company_name": passport.company_name.value if passport.company_name.status != "MISSING" else "",
                    "tax_code": passport.tax_code.value if passport.tax_code.status != "MISSING" else "",
                    "location": passport.location.value if passport.location.status != "MISSING" else "",
                    "employee_count": passport.employee_count.value if passport.employee_count.status != "MISSING" else "",
                    "rd_spend_ratio": f"{passport.rd_spend_ratio.value * 100}%" if passport.rd_spend_ratio.status != "MISSING" else "",
                    "registered_capital": f"{passport.registered_capital.value:,} VND" if passport.registered_capital.status != "MISSING" else "",
                    "revenue": f"{passport.revenue.value:,} VND" if passport.revenue.status != "MISSING" else ""
                }
                doc.render(context)
                
                output_file_name = f"filled_grant_{id}.docx"
                output_path = os.path.join(SEED_DIR, output_file_name)
                doc.save(output_path)
                
                # Auto-transition status to GENERATED
                cursor.execute(
                    "UPDATE drafts SET status = ?, updated_at = ? WHERE id = ?",
                    ("GENERATED", timestamp, id)
                )
                new_status = "GENERATED"
            except Exception as e:
                print(f"[Error] Document Template filling failed: {e}")
                
    conn.commit()
    conn.close()
    return {"id": id, "status": new_status, "reviewer_comments": req.reviewer_comments}

@app.get("/api/v1/drafts/{id}/download")
def download_draft(id: str, user: Dict[str, Any] = Depends(get_current_user)):
    company_id = user["company_id"] or user["personal_passport_id"]
    conn = get_db_connection()
    cursor = conn.cursor()
    
    # Restrict downloads by tenant
    cursor.execute("SELECT status FROM drafts WHERE id = ? AND company_id = ?", (id, company_id))
    row = cursor.fetchone()
    conn.close()
    
    if not row:
        raise HTTPException(status_code=404, detail="Draft not found or access forbidden")
        
    if row["status"] != "GENERATED":
        raise HTTPException(status_code=400, detail=f"Cannot download draft with status '{row['status']}'. It must be approved first.")
        
    output_file_name = f"filled_grant_{id}.docx"
    output_path = os.path.join(SEED_DIR, output_file_name)
    
    if not os.path.exists(output_path):
        raise HTTPException(status_code=404, detail="Document file not found.")
        
    return FileResponse(output_path, filename=f"P2B_Draft_{id}.docx")

# ================= MULTI-DOCUMENT INGESTION & AGENT ENDPOINTS =================

@app.post("/api/v1/extract")
def extract_document(file: UploadFile = File(...), user: Dict[str, Any] = Depends(get_current_user)):
    """Single file upload extraction endpoint (Goal 2)"""
    temp_path = os.path.join(UPLOADS_DIR, f"temp_{uuid.uuid4().hex}_{file.filename}")
    with open(temp_path, "wb") as buffer:
        shutil.copyfileobj(file.file, buffer)
        
    try:
        _, ext = os.path.splitext(file.filename.lower())
        mime_type = "application/pdf" if ext == ".pdf" else "text/plain"
        
        # Converted to markdown locally if not PDF
        if ext != '.pdf':
            try:
                markdown_text = convert_to_markdown_local(temp_path)
                # Overwrite temp_path with converted markdown text
                with open(temp_path, 'w', encoding='utf-8') as f:
                    f.write(markdown_text)
                mime_type = "text/plain"
            except Exception as e:
                raise HTTPException(status_code=400, detail=f"MarkItDown conversion failed: {str(e)}")
                
        # Call structured extraction via Gemini 3.1
        extracted_data = call_gemini_extraction(temp_path, mime_type, user["user_type"])
        
        # Persist extracted passport back to database
        timestamp = datetime.utcnow().isoformat()
        conn = get_db_connection()
        cursor = conn.cursor()
        
        if user["user_type"] == "COMPANY_MANAGER":
            passport_id = user["company_id"]
            # Compile fields with provenance
            company_passport = {}
            for field in ["company_name", "tax_code", "industry", "location", "employee_count", "rd_spend_ratio", "revenue", "registered_capital"]:
                val = extracted_data.get(field, "")
                quote = extracted_data.get("evidence_quotes", {}).get(field, "")
                loc = extracted_data.get("page_locations", {}).get(field, "")
                
                company_passport[field] = {
                    "value": val,
                    "source_type": "BUSINESS_REGISTRATION" if "đăng ký" in file.filename.lower() else "FINANCIAL_REPORT",
                    "source_uri": file.filename,
                    "source_location": loc,
                    "evidence_quote": quote,
                    "observed_at": timestamp,
                    "confidence": "HIGH" if quote else "LOW",
                    "status": "EXTRACTED" if quote else "MISSING",
                    "conflicts": []
                }
            company_passport["metadata"] = {}
            
            cursor.execute("UPDATE company_passports SET data_json = ?, updated_at = ? WHERE id = ?", (json.dumps(company_passport, ensure_ascii=False), timestamp, passport_id))
            
            # Log edit to audit logs
            cursor.execute("INSERT INTO audit_logs (event_type, target_id, field_name, old_value, new_value, timestamp) VALUES (?, ?, ?, ?, ?, ?)",
                           ("PASSPORT_EDIT", passport_id, "all_extracted", "EMPTY", file.filename, timestamp))
            
            result_passport = company_passport
        else:
            # Individual
            passport_id = user["personal_passport_id"]
            cursor.execute(
                """
                UPDATE personal_passports 
                SET full_name = ?, birth_year = ?, location = ?, occupation = ?, degree = ?, monthly_income = ?, updated_at = ?
                WHERE id = ?
                """,
                (
                    extracted_data.get("full_name", ""),
                    int(extracted_data.get("birth_year", 0)),
                    extracted_data.get("location", ""),
                    extracted_data.get("occupation", ""),
                    extracted_data.get("degree", ""),
                    int(extracted_data.get("monthly_income", 0)),
                    timestamp,
                    passport_id
                )
            )
            result_passport = extracted_data
            
        conn.commit()
        conn.close()
        
        # Clean up temp file
        if os.path.exists(temp_path):
            os.remove(temp_path)
            
        return result_passport
        
    except Exception as e:
        if os.path.exists(temp_path):
            os.remove(temp_path)
        raise HTTPException(status_code=500, detail=f"Extraction failed: {str(e)}")

@app.post("/api/v1/extract-multi")
def extract_multiple_documents(files: List[UploadFile] = File(...), user: Dict[str, Any] = Depends(get_current_user)):
    """Intelligent multi-document sorting and extraction agent (Goal 7)"""
    if not files:
        raise HTTPException(status_code=400, detail="No files uploaded")
        
    temp_docs = []
    
    # Save and convert files
    for f in files:
        temp_path = os.path.join(UPLOADS_DIR, f"temp_multi_{uuid.uuid4().hex}_{f.filename}")
        with open(temp_path, "wb") as buffer:
            shutil.copyfileobj(f.file, buffer)
            
        _, ext = os.path.splitext(f.filename.lower())
        doc_text = ""
        mime_type = "application/pdf"
        
        if ext == ".pdf":
            mime_type = "application/pdf"
            doc_text = f.filename
        else:
            try:
                doc_text = convert_to_markdown_local(temp_path)
                mime_type = "text/plain"
            except Exception as e:
                # Cleanup and error
                for td in temp_docs:
                    if os.path.exists(td["path"]):
                        os.remove(td["path"])
                raise HTTPException(status_code=400, detail=f"MarkItDown conversion failed for file '{f.filename}': {str(e)}")
                
        temp_docs.append({
            "name": f.filename,
            "path": temp_path,
            "text": doc_text or f.filename,
            "mime_type": mime_type
        })
        
    try:
        timestamp = datetime.utcnow().isoformat()
        conn = get_db_connection()
        cursor = conn.cursor()
        
        if user["user_type"] == "COMPANY_MANAGER":
            passport_id = user["company_id"]
            
            # Rank documents for two core field categories to optimize API calls
            best_reg_doc = rank_documents_for_field("đăng ký doanh nghiệp, mã số thuế, trụ sở, vốn điều lệ", temp_docs)
            best_finance_doc = rank_documents_for_field("báo cáo tài chính, doanh thu, nhân sự, nghiên cứu và phát triển R&D", temp_docs)
            
            print(f"[Agent Multi-Sort] Selected '{best_reg_doc['name']}' for registration fields.")
            print(f"[Agent Multi-Sort] Selected '{best_finance_doc['name']}' for financial/R&D fields.")
            
            reg_extracted = call_gemini_extraction(best_reg_doc["path"], best_reg_doc["mime_type"], "COMPANY_MANAGER")
            fin_extracted = call_gemini_extraction(best_finance_doc["path"], best_finance_doc["mime_type"], "COMPANY_MANAGER")
            
            company_passport = {}
            for field in ["company_name", "tax_code", "location", "registered_capital"]:
                val = reg_extracted.get(field, "")
                quote = reg_extracted.get("evidence_quotes", {}).get(field, "")
                loc = reg_extracted.get("page_locations", {}).get(field, "")
                
                company_passport[field] = {
                    "value": val,
                    "source_type": "BUSINESS_REGISTRATION",
                    "source_uri": best_reg_doc["name"],
                    "source_location": loc,
                    "evidence_quote": quote,
                    "observed_at": timestamp,
                    "confidence": "HIGH" if quote else "LOW",
                    "status": "EXTRACTED" if quote else "MISSING",
                    "conflicts": []
                }
                
            for field in ["industry", "employee_count", "rd_spend_ratio", "revenue"]:
                val = fin_extracted.get(field, "")
                quote = fin_extracted.get("evidence_quotes", {}).get(field, "")
                loc = fin_extracted.get("page_locations", {}).get(field, "")
                
                company_passport[field] = {
                    "value": val,
                    "source_type": "FINANCIAL_REPORT",
                    "source_uri": best_finance_doc["name"],
                    "source_location": loc,
                    "evidence_quote": quote,
                    "observed_at": timestamp,
                    "confidence": "HIGH" if quote else "LOW",
                    "status": "EXTRACTED" if quote else "MISSING",
                    "conflicts": []
                }
            company_passport["metadata"] = {}
            
            cursor.execute("UPDATE company_passports SET data_json = ?, updated_at = ? WHERE id = ?", (json.dumps(company_passport, ensure_ascii=False), timestamp, passport_id))
            cursor.execute("INSERT INTO audit_logs (event_type, target_id, field_name, old_value, new_value, timestamp) VALUES (?, ?, ?, ?, ?, ?)",
                           ("PASSPORT_EDIT", passport_id, "multi_extracted", "EMPTY", f"files: {list(set([best_reg_doc['name'], best_finance_doc['name']]))}", timestamp))
            
            result_passport = company_passport
            
        else:
            # Individual User Multi Extraction
            best_resume_doc = rank_documents_for_field("sơ yếu lý lịch, thông tin cá nhân, bằng cấp, thu nhập", temp_docs)
            personal_extracted = call_gemini_extraction(best_resume_doc["path"], best_resume_doc["mime_type"], "INDIVIDUAL")
            
            passport_id = user["personal_passport_id"]
            cursor.execute(
                """
                UPDATE personal_passports 
                SET full_name = ?, birth_year = ?, location = ?, occupation = ?, degree = ?, monthly_income = ?, updated_at = ?
                WHERE id = ?
                """,
                (
                    personal_extracted.get("full_name", ""),
                    int(personal_extracted.get("birth_year", 0)),
                    personal_extracted.get("location", ""),
                    personal_extracted.get("occupation", ""),
                    personal_extracted.get("degree", ""),
                    int(personal_extracted.get("monthly_income", 0)),
                    timestamp,
                    passport_id
                )
            )
            result_passport = personal_extracted
            
        conn.commit()
        conn.close()
        
        # Cleanup temp docs
        for td in temp_docs:
            if os.path.exists(td["path"]):
                os.remove(td["path"])
                
        return result_passport
        
    except Exception as e:
        for td in temp_docs:
            if os.path.exists(td["path"]):
                os.remove(td["path"])
        raise HTTPException(status_code=500, detail=f"Multi-document extraction failed: {str(e)}")

# ================= AUDIT LOGS ENDPOINTS =================

@app.get("/api/v1/audit_logs")
def list_audit_logs(user: Dict[str, Any] = Depends(get_current_user)):
    conn = get_db_connection()
    cursor = conn.cursor()
    # Filter logs scoped by active tenant (company ID or personal passport ID)
    target_id = user["company_id"] or user["personal_passport_id"]
    cursor.execute("""
        SELECT id, event_type, target_id, field_name, old_value, new_value, timestamp 
        FROM audit_logs 
        WHERE target_id = ? OR target_id IN (SELECT id FROM drafts WHERE company_id = ?)
        ORDER BY id DESC
    """, (target_id, target_id))
    rows = cursor.fetchall()
    conn.close()
    
    result = []
    for r in rows:
        result.append({
            "id": r["id"],
            "event_type": r["event_type"],
            "target_id": r["target_id"],
            "field_name": r["field_name"],
            "old_value": r["old_value"],
            "new_value": r["new_value"],
            "timestamp": r["timestamp"]
        })
    return result

# ================= VERIFIABLE POLICY SYNC & ALERTS ENDPOINTS =================

@app.post("/api/v1/sync")
def sync_legal_documents(user: Dict[str, Any] = Depends(get_current_user)):
    """Verifiable SHA-256 diff hash policy sync engine (Goal 3)"""
    conn = get_db_connection()
    cursor = conn.cursor()
    
    sync_logs = []
    timestamp = datetime.utcnow().isoformat()
    
    if not os.path.exists(INCOMING_DIR):
        conn.close()
        return {"status": "success", "synced_count": 0, "logs": ["No incoming directory found"]}
        
    incoming_files = [f for f in os.listdir(INCOMING_DIR) if f.endswith(".md")]
    if not incoming_files:
        conn.close()
        return {"status": "success", "synced_count": 0, "logs": ["No new incoming legal documents found"]}
        
    synced_count = 0
    for filename in incoming_files:
        file_path = os.path.join(INCOMING_DIR, filename)
        doc_id = os.path.splitext(filename)[0]
        
        with open(file_path, "r", encoding="utf-8") as f:
            content = f.read()
            
        content_hash = hashlib.sha256(content.encode("utf-8")).hexdigest()
        
        cursor.execute("SELECT id, data_json FROM policy_opportunities WHERE id = ?", (doc_id,))
        opp_row = cursor.fetchone()
        
        if opp_row:
            corpus_path = os.path.join(SEED_DIR, "legal_corpus.json")
            if os.path.exists(corpus_path):
                with open(corpus_path, "r", encoding="utf-8") as jf:
                    corpus = json.load(jf)
                
                doc_in_corpus = next((d for d in corpus if d["id"] == doc_id), None)
                if doc_in_corpus:
                    old_hash = doc_in_corpus.get("content_hash", "")
                    if old_hash != content_hash:
                        synced_count += 1
                        change_desc = f"Nội dung văn bản {doc_in_corpus['title']} được cập nhật đổi điều khoản điều kiện."
                        
                        cursor.execute(
                            "INSERT INTO policy_alerts (document_id, title, change_description, timestamp) VALUES (?, ?, ?, ?)",
                            (doc_id, doc_in_corpus["title"], change_desc, timestamp)
                        )
                        
                        doc_in_corpus["content_hash"] = content_hash
                        doc_in_corpus["chunks"] = [content]
                        
                        with open(corpus_path, "w", encoding="utf-8") as out_jf:
                            json.dump(corpus, out_jf, ensure_ascii=False, indent=2)
                            
                        from app.seed.generate_cache import pre_warm_embeddings
                        pre_warm_embeddings(SEED_DIR)
                        
                        sync_logs.append(f"Cập nhật: {doc_in_corpus['title']} (Phát hiện Hash thay đổi từ {old_hash[:8]} thành {content_hash[:8]})")
                        
    conn.commit()
    conn.close()
    
    return {
        "status": "success",
        "synced_count": synced_count,
        "logs": sync_logs or ["Tất cả văn bản pháp lý trùng khớp, không có cập nhật mới."]
    }

@app.get("/api/v1/policy_alerts")
def get_policy_alerts(user: Dict[str, Any] = Depends(get_current_user)):
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT id, document_id, title, change_description, timestamp FROM policy_alerts ORDER BY id DESC LIMIT 10")
    rows = cursor.fetchall()
    conn.close()
    
    result = []
    for r in rows:
        result.append({
            "id": r["id"],
            "document_id": r["document_id"],
            "title": r["title"],
            "change_description": r["change_description"],
            "timestamp": r["timestamp"]
        })
    return result
