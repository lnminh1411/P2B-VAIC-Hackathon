import os
import uuid
import json
from datetime import datetime
from fastapi import FastAPI, HTTPException, status, Query
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse
from pydantic import BaseModel, Field
from typing import Optional, List, Any

from app.engine.db import get_db_connection
from app.schemas.passport import CompanyPassport
from app.schemas.policy import PolicyOpportunity
from app.engine.retrieval import HybridRetrievalEngine
from app.engine.rule_evaluator import evaluate_rule_group
from docxtpl import DocxTemplate

app = FastAPI(
    title="P2B (Policy-to-Business) API",
    description="AI-native platform helping Vietnamese businesses navigate legal policies and incentives.",
    version="1.0.0"
)

# Enable CORS for React frontend
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # For hackathon dev, allow all
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Load retrieval engine on startup
SEED_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "seed")
retrieval_engine = HybridRetrievalEngine(SEED_DIR)

class DraftCreateRequest(BaseModel):
    company_id: str
    policy_id: str

class DraftStatusUpdateRequest(BaseModel):
    status: str  # APPROVED, REJECTED
    reviewer_comments: Optional[str] = ""

class PassportUpdateRequest(BaseModel):
    passport_data: dict

@app.get("/api/v1/passports")
def list_passports():
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT id, data_json, updated_at FROM company_passports")
    rows = cursor.fetchall()
    conn.close()
    
    result = []
    for r in rows:
        result.append({
            "id": r["id"],
            "data": json.loads(r["data_json"]),
            "updated_at": r["updated_at"]
        })
    return result

@app.get("/api/v1/passports/{id}")
def get_passport(id: str):
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
def update_passport(id: str, req: PassportUpdateRequest):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    # 1. Fetch current passport to detect changes for audit logs
    cursor.execute("SELECT data_json FROM company_passports WHERE id = ?", (id,))
    row = cursor.fetchone()
    if not row:
        conn.close()
        raise HTTPException(status_code=404, detail="Company Passport not found")
        
    old_data = json.loads(row["data_json"])
    new_data = req.passport_data
    
    # Validate structure using Pydantic CompanyPassport schema
    try:
        validated_passport = CompanyPassport(**new_data)
    except Exception as e:
        conn.close()
        raise HTTPException(status_code=422, detail=f"Invalid CompanyPassport schema: {str(e)}")
        
    # Check for changes in values and log them to audit_logs
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
            
            # Since user manually confirmed/updated this field, set status to USER_CONFIRMED and source MANUAL_INPUT
            new_prov["status"] = "USER_CONFIRMED"
            new_prov["source_type"] = "MANUAL_INPUT"
            new_prov["observed_at"] = timestamp
            
    # Update passport in database
    cursor.execute(
        "UPDATE company_passports SET data_json = ?, updated_at = ? WHERE id = ?",
        (json.dumps(new_data, ensure_ascii=False), timestamp, id)
    )
    conn.commit()
    conn.close()
    
    return {"message": "Company Passport updated successfully", "data": new_data}

@app.get("/api/v1/policies")
def list_policies(company_id: str, query: Optional[str] = ""):
    # Get company passport
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT data_json FROM company_passports WHERE id = ?", (company_id,))
    row = cursor.fetchone()
    conn.close()
    
    if not row:
        raise HTTPException(status_code=404, detail="Company Passport not found")
        
    passport = CompanyPassport(**json.loads(row["data_json"]))
    
    # Run hybrid retrieval
    results = retrieval_engine.retrieve(passport, query or "chính sách hỗ trợ doanh nghiệp", top_n=10)
    
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
def get_policy(id: str):
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT data_json FROM policy_opportunities WHERE id = ?", (id,))
    row = cursor.fetchone()
    conn.close()
    
    if not row:
        raise HTTPException(status_code=404, detail="Policy Opportunity not found")
        
    return json.loads(row["data_json"])

@app.post("/api/v1/eligibility")
def run_eligibility_verification(req: DraftCreateRequest):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    # Get passport
    cursor.execute("SELECT data_json FROM company_passports WHERE id = ?", (req.company_id,))
    p_row = cursor.fetchone()
    
    # Get policy
    cursor.execute("SELECT data_json FROM policy_opportunities WHERE id = ?", (req.policy_id,))
    opp_row = cursor.fetchone()
    conn.close()
    
    if not p_row:
        raise HTTPException(status_code=404, detail="Company Passport not found")
    if not opp_row:
        raise HTTPException(status_code=404, detail="Policy Opportunity not found")
        
    passport = CompanyPassport(**json.loads(p_row["data_json"]))
    opp = PolicyOpportunity(**json.loads(opp_row["data_json"]))
    
    status_result, details = evaluate_rule_group(passport, opp.eligibility_rules)
    return {
        "company_id": req.company_id,
        "policy_id": req.policy_id,
        "status": status_result,
        "details": details
    }

@app.post("/api/v1/drafts")
def create_draft(req: DraftCreateRequest):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    # Fetch passport and policy to run eligibility
    cursor.execute("SELECT data_json FROM company_passports WHERE id = ?", (req.company_id,))
    p_row = cursor.fetchone()
    cursor.execute("SELECT data_json FROM policy_opportunities WHERE id = ?", (req.policy_id,))
    opp_row = cursor.fetchone()
    
    if not p_row:
        conn.close()
        raise HTTPException(status_code=404, detail="Company Passport not found")
    if not opp_row:
        conn.close()
        raise HTTPException(status_code=404, detail="Policy Opportunity not found")
        
    passport = CompanyPassport(**json.loads(p_row["data_json"]))
    opp = PolicyOpportunity(**json.loads(opp_row["data_json"]))
    
    # Evaluate rules
    status_result, details = evaluate_rule_group(passport, opp.eligibility_rules)
    
    # Create draft record
    draft_id = str(uuid.uuid4())
    timestamp = datetime.utcnow().isoformat()
    
    cursor.execute(
        """
        INSERT INTO drafts (id, company_id, opportunity_id, status, details_json, reviewer_comments, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        """,
        (
            draft_id,
            req.company_id,
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
def list_drafts():
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT id, company_id, opportunity_id, status, reviewer_comments, created_at, updated_at FROM drafts")
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
def get_draft(id: str):
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT id, company_id, opportunity_id, status, details_json, reviewer_comments, created_at, updated_at FROM drafts WHERE id = ?", (id,))
    row = cursor.fetchone()
    conn.close()
    
    if not row:
        raise HTTPException(status_code=404, detail="Draft not found")
        
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
def update_draft_status(id: str, req: DraftStatusUpdateRequest):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    cursor.execute("SELECT company_id, opportunity_id, status, details_json FROM drafts WHERE id = ?", (id,))
    row = cursor.fetchone()
    if not row:
        conn.close()
        raise HTTPException(status_code=404, detail="Draft not found")
        
    company_id = row["company_id"]
    opportunity_id = row["opportunity_id"]
    old_status = row["status"]
    
    timestamp = datetime.utcnow().isoformat()
    new_status = req.status
    
    # 1. Update status and comments in DB
    cursor.execute(
        "UPDATE drafts SET status = ?, reviewer_comments = ?, updated_at = ? WHERE id = ?",
        (new_status, req.reviewer_comments, timestamp, id)
    )
    
    # 2. Write audit log
    cursor.execute(
        """
        INSERT INTO audit_logs (event_type, target_id, field_name, old_value, new_value, timestamp)
        VALUES (?, ?, ?, ?, ?, ?)
        """,
        ("DRAFT_STATUS_CHANGE", id, "status", old_status, new_status, timestamp)
    )
    
    # 3. If approved, fill document template (Template Filler)
    if new_status == "APPROVED":
        cursor.execute("SELECT data_json FROM company_passports WHERE id = ?", (company_id,))
        p_row = cursor.fetchone()
        
        if p_row:
            passport = CompanyPassport(**json.loads(p_row["data_json"]))
            
            # Fill the template
            template_path = os.path.join(SEED_DIR, "grant_template.docx")
            if os.path.exists(template_path):
                try:
                    doc = DocxTemplate(template_path)
                    
                    # Fill only fields that have evidence and are not missing
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
                    
                    # Save filled draft to seed directory
                    output_file_name = f"filled_grant_{id}.docx"
                    output_path = os.path.join(SEED_DIR, output_file_name)
                    doc.save(output_path)
                    
                    # Update status to GENERATED
                    cursor.execute(
                        "UPDATE drafts SET status = ?, updated_at = ? WHERE id = ?",
                        ("GENERATED", timestamp, id)
                    )
                    new_status = "GENERATED"
                except Exception as e:
                    print(f"[Error] Template filling failed: {e}")
                    
    conn.commit()
    conn.close()
    
    return {"id": id, "status": new_status, "reviewer_comments": req.reviewer_comments}

@app.get("/api/v1/drafts/{id}/download")
def download_draft(id: str):
    output_file_name = f"filled_grant_{id}.docx"
    output_path = os.path.join(SEED_DIR, output_file_name)
    
    if not os.path.exists(output_path):
        # Fallback to the seed template if not generated yet
        fallback_path = os.path.join(SEED_DIR, "filled_grant_demo.docx")
        if os.path.exists(fallback_path):
            return FileResponse(fallback_path, filename="filled_grant_demo.docx")
        raise HTTPException(status_code=404, detail="Draft document file not found.")
        
    # prototype disclaimer in header/logs
    print(f"[Prototype Disclaimer] Serving unauthenticated download for draft: {id}")
    return FileResponse(output_path, filename=f"P2B_Draft_{id}.docx")

@app.get("/api/v1/audit_logs")
def list_audit_logs():
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT id, event_type, target_id, field_name, old_value, new_value, timestamp FROM audit_logs ORDER BY id DESC")
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
