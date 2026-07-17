import os
import sqlite3
import json
from datetime import datetime
from typing import List, Any, Optional

DB_PATH = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "p2b_database.sqlite")

def get_db_connection():
    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
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
            
    # Seed default crawler config if empty
    cursor.execute("SELECT COUNT(*) FROM crawler_configs")
    if cursor.fetchone()[0] == 0:
        cursor.execute(
            "INSERT INTO crawler_configs (id, cron_expression, portals, enabled) VALUES (?, ?, ?, ?)",
            ("default_sync", "0 0 * * *", "vbpl.vn,nic.gov.vn", 1)
        )
        print("Seeded default crawler config.")
        
    conn.commit()
    conn.close()
    print("Database initialization complete.")

if __name__ == "__main__":
    init_db()
