import os
import sqlite3
import json
from datetime import datetime
from typing import List, Any, Optional

DB_PATH = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "p2b_database.sqlite")

def get_db_connection():
    conn = sqlite3.connect(DB_PATH, timeout=5.0)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA journal_mode=WAL;")
    conn.execute("PRAGMA synchronous=NORMAL;")
    return conn

def init_db():
    print(f"Initializing SQLite database at: {DB_PATH}")
    conn = get_db_connection()
    cursor = conn.cursor()
    
    # 1. Company Passports Table
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS company_passports (
        id TEXT PRIMARY KEY,
        data_json TEXT NOT NULL,
        updated_at TEXT NOT NULL
    )
    """)
    
    # 2. Policy Opportunities Table
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS policy_opportunities (
        id TEXT PRIMARY KEY,
        data_json TEXT NOT NULL
    )
    """)
    
    # 3. Drafts Table
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS drafts (
        id TEXT PRIMARY KEY,
        company_id TEXT NOT NULL,
        opportunity_id TEXT NOT NULL,
        status TEXT NOT NULL, -- DRAFT, PENDING_REVIEW, APPROVED, REJECTED, GENERATED
        details_json TEXT NOT NULL, -- Eligibility results JSON
        reviewer_comments TEXT,
        created_at TEXT NOT NULL,
        updated_at TEXT NOT NULL
    )
    """)
    
    # 4. Audit Logs Table
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS audit_logs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_type TEXT NOT NULL, -- PASSPORT_EDIT, DRAFT_STATUS_CHANGE
        target_id TEXT NOT NULL,
        field_name TEXT,
        old_value TEXT,
        new_value TEXT,
        timestamp TEXT NOT NULL
    )
    """)
    
    # 5. Crawler Configs Table
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS crawler_configs (
        id TEXT PRIMARY KEY,
        cron_expression TEXT NOT NULL,
        portals TEXT NOT NULL, -- Comma-separated list
        enabled INTEGER NOT NULL DEFAULT 1
    )
    """)

    # 6. Users Table
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS users (
        id TEXT PRIMARY KEY,
        email TEXT UNIQUE NOT NULL,
        hashed_password TEXT NOT NULL,
        user_type TEXT NOT NULL, -- COMPANY_MANAGER or INDIVIDUAL
        company_id TEXT, -- links to company_passports.id
        personal_passport_id TEXT, -- links to personal_passports.id
        avatar_path TEXT,
        created_at TEXT NOT NULL
    )
    """)

    # 7. Personal Passports Table
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS personal_passports (
        id TEXT PRIMARY KEY,
        full_name TEXT NOT NULL,
        birth_year INTEGER,
        location TEXT,
        occupation TEXT,
        degree TEXT,
        monthly_income INTEGER,
        uploaded_files_json TEXT,
        updated_at TEXT NOT NULL
    )
    """)
    
    # SQLite migration: alter table to add column if it doesn't exist
    try:
        cursor.execute("ALTER TABLE personal_passports ADD COLUMN uploaded_files_json TEXT")
    except sqlite3.OperationalError:
        pass

    # 8. Sessions Table
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS sessions (
        token TEXT PRIMARY KEY,
        user_id TEXT NOT NULL,
        created_at TEXT NOT NULL,
        expires_at TEXT NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
    )
    """)

    # 9. Policy Alerts Table
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS policy_alerts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        document_id TEXT NOT NULL,
        title TEXT NOT NULL,
        change_description TEXT NOT NULL,
        timestamp TEXT NOT NULL
    )
    """)

    # 10. Legal Documents Table
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS legal_documents (
        id TEXT PRIMARY KEY,
        title TEXT NOT NULL,
        chunks_json TEXT NOT NULL,
        xml_content TEXT,
        updated_at TEXT NOT NULL
    )
    """)
    
    try:
        cursor.execute("ALTER TABLE legal_documents ADD COLUMN xml_content TEXT")
    except sqlite3.OperationalError:
        pass
    
    # Seed Company Passports if empty
    cursor.execute("SELECT COUNT(*) FROM company_passports")
    if cursor.fetchone()[0] == 0:
        seed_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "seed")
        passports_file = os.path.join(seed_dir, "company_passports.json")
        if os.path.exists(passports_file):
            with open(passports_file, "r", encoding="utf-8") as f:
                passports = json.load(f)
            for cid, cdata in passports.items():
                cursor.execute(
                    "INSERT INTO company_passports (id, data_json, updated_at) VALUES (?, ?, ?)",
                    (cid, json.dumps(cdata, ensure_ascii=False), datetime.utcnow().isoformat())
                )
            print(f"Seeded {len(passports)} company passports.")
            
    # Seed Policy Opportunities if empty
    cursor.execute("SELECT COUNT(*) FROM policy_opportunities")
    if cursor.fetchone()[0] == 0:
        seed_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "seed")
        opps_file = os.path.join(seed_dir, "policy_opportunities.json")
        if os.path.exists(opps_file):
            with open(opps_file, "r", encoding="utf-8") as f:
                opps = json.load(f)
            for opp in opps:
                cursor.execute(
                    "INSERT INTO policy_opportunities (id, data_json) VALUES (?, ?)",
                    (opp["id"], json.dumps(opp, ensure_ascii=False))
                )
            print(f"Seeded {len(opps)} policy opportunities.")

    # Seed Legal Documents
    seed_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "seed")
    try:
        from app.seed.corpus_generator import create_legal_corpus
        create_legal_corpus(seed_dir)
    except Exception as e:
        print(f"[Database Warning] Failed to run corpus_generator: {e}")

    cursor.execute("SELECT COUNT(*) FROM legal_documents WHERE id = ?", ("decision_10_2021_qd_ttg",))
    if cursor.fetchone()[0] == 0:
        corpus_file = os.path.join(seed_dir, "legal_corpus.json")
        if os.path.exists(corpus_file):
            with open(corpus_file, "r", encoding="utf-8") as f:
                corpus = json.load(f)
            from bs4 import BeautifulSoup
            import xml.etree.ElementTree as ET
            for doc in corpus:
                root_el = ET.Element("document")
                root_el.set("id", doc["id"])
                root_el.set("number", doc["id"].upper().replace("_", "/"))
                root_el.set("type", "Văn bản pháp luật")
                root_el.set("status", "Còn hiệu lực")
                
                title_el = ET.SubElement(root_el, "title")
                title_el.text = doc["title"]
                
                content_text = "\n\n".join([c.get("content", "") if isinstance(c, dict) else str(c) for c in doc["chunks"]])
                content_el = ET.SubElement(root_el, "content")
                content_el.text = content_text
                
                xml_str = BeautifulSoup(ET.tostring(root_el, encoding="utf-8"), "xml").prettify()
                
                cursor.execute(
                    "INSERT OR REPLACE INTO legal_documents (id, title, chunks_json, xml_content, updated_at) VALUES (?, ?, ?, ?, ?)",
                    (doc["id"], doc["title"], json.dumps(doc["chunks"], ensure_ascii=False), xml_str, datetime.utcnow().isoformat())
                )
            print(f"Seeded {len(corpus)} legal documents with XML content.")
            
    # Seed default crawler config if empty
    cursor.execute("SELECT COUNT(*) FROM crawler_configs")
    if cursor.fetchone()[0] == 0:
        cursor.execute(
            "INSERT INTO crawler_configs (id, cron_expression, portals, enabled) VALUES (?, ?, ?, ?)",
            ("default_sync", "0 0 * * *", "vbpl.vn,nic.gov.vn", 1)
        )
        print("Seeded default crawler config.")
        
    # Seed default users if empty
    cursor.execute("SELECT COUNT(*) FROM users")
    if cursor.fetchone()[0] == 0:
        from app.engine.auth import hash_password
        import uuid
        
        default_users = [
            ("AItech_Vietnam_LLC", "aitech@p2b.vn", "Password123", "COMPANY_MANAGER"),
            ("FDI_SemiVina_Corp", "semivina@p2b.vn", "Password123", "COMPANY_MANAGER"),
            ("SolarGreen_Tech_JSC", "solargreen@p2b.vn", "Password123", "COMPANY_MANAGER")
        ]
        for cid, email, pwd, utype in default_users:
            uid = str(uuid.uuid4())
            hashed = hash_password(pwd)
            cursor.execute(
                "INSERT INTO users (id, email, hashed_password, user_type, company_id, created_at) VALUES (?, ?, ?, ?, ?, ?)",
                (uid, email, hashed, utype, cid, datetime.utcnow().isoformat())
            )
            
        # Seed an individual user
        individual_uid = str(uuid.uuid4())
        individual_pid = str(uuid.uuid4())
        hashed_ind = hash_password("Password123")
        # Insert personal passport first
        cursor.execute(
            "INSERT INTO personal_passports (id, full_name, birth_year, location, occupation, degree, monthly_income, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
            (individual_pid, "Nguyễn Văn Chuyên Gia", 1992, "Đà Nẵng", "Semiconductor Engineer", "Master", 45000000, datetime.utcnow().isoformat())
        )
        # Insert user
        cursor.execute(
            "INSERT INTO users (id, email, hashed_password, user_type, personal_passport_id, created_at) VALUES (?, ?, ?, ?, ?, ?)",
            (individual_uid, "individual@p2b.vn", hashed_ind, "INDIVIDUAL", individual_pid, datetime.utcnow().isoformat())
        )
        print("Seeded default users and personal passport.")

    conn.commit()
    conn.close()
    print("Database initialization complete.")

if __name__ == "__main__":
    init_db()
