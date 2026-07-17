import os
import sqlite3
import json
import requests
import re
from datetime import datetime
from bs4 import BeautifulSoup
import xml.etree.ElementTree as ET

# Ensure requests unverified warning is silenced
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

DB_PATH = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "p2b_database.sqlite")

def get_db_connection():
    conn = sqlite3.connect(DB_PATH, timeout=10.0)
    conn.row_factory = sqlite3.Row
    return conn

def clean_html_content(html_str: str) -> str:
    if not html_str:
        return ""
    soup = BeautifulSoup(html_str, "html.parser")
    for script in soup(["script", "style"]):
        script.decompose()
    text = soup.get_text(separator="\n")
    lines = (line.strip() for line in text.splitlines())
    chunks = (phrase.strip() for line in lines for phrase in line.split("  "))
    cleaned = "\n".join(chunk for chunk in chunks if chunk)
    return cleaned

def fetch_document_data(doc_id: str) -> dict:
    url = f"https://vbpl-bientap-gateway.moj.gov.vn/api/qtdc/public/doc/{doc_id}"
    headers = {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
    }
    try:
        response = requests.get(url, headers=headers, verify=False, timeout=8.0)
        if response.status_code == 200:
            res_json = response.json()
            if res_json.get("success") and "data" in res_json:
                return res_json["data"]
    except Exception as e:
        print(f"[Crawler Error] Failed to fetch {doc_id}: {e}")
    return None

def generate_xml_for_llm(doc_data: dict) -> str:
    root = ET.Element("document")
    root.set("id", str(doc_data.get("id") or ""))
    root.set("number", str(doc_data.get("docNum") or "N/A"))
    
    doc_type = doc_data.get("docType")
    type_name = doc_type.get("name") if isinstance(doc_type, dict) else doc_type
    root.set("type", str(type_name or "N/A"))
    
    root.set("agency", str(doc_data.get("agencyName") or "N/A"))
    root.set("issue_date", str(doc_data.get("issueDate") or "N/A"))
    root.set("effective_date", str(doc_data.get("effFrom") or "N/A"))
    
    eff_status = doc_data.get("effStatus")
    status_name = eff_status.get("name") if isinstance(eff_status, dict) else eff_status
    root.set("status", str(status_name or "N/A"))
    
    title = ET.SubElement(root, "title")
    title.text = str(doc_data.get("title") or "")
    
    content = ET.SubElement(root, "content")
    content_raw = ""
    if isinstance(doc_data.get("documentContent"), dict):
        content_raw = doc_data.get("documentContent", {}).get("content", "")
    content.text = clean_html_content(content_raw)
    
    rough_string = ET.tostring(root, encoding="utf-8")
    soup = BeautifulSoup(rough_string, "xml")
    return soup.prettify()

def main():
    # 5 Years worth of relevant document IDs (UUIDs and Legacy IDs)
    target_doc_ids = [
        # Discovered via active research & UI inspection
        "4fda4da0-80ec-11f1-ac2d-554d7f9461b5", # Thông tư 100/2026/TT-BTC
        "113ff190-8023-11f1-9806-1dab8b6c3e51", # Nghị định 280/2026/NĐ-CP
        "2e0bbb80-8017-11f1-95e2-45a2bc394098", # Thông tư mẫu kiểm toán nội bộ
        "abc1ead0-7e97-11f1-b894-6dc9dff16474", # Nghị định 278/2026/NĐ-CP
        "d4ba5790-8010-11f1-93ea-dd502af5ba0e", # Văn bản hợp nhất 68/2026
        "9d85c890-7f39-11f1-a307-ef47d5d415c6", # Nghị định 275/2026/NĐ-CP
        "ee626bf0-8007-11f1-9817-a78f5fdc4853", # Nghị định 276/2026/NĐ-CP
        "839f4ab0-79a7-11f1-84f8-c94e29623f00", # Thông tư công tác xã hội
        "9402dd90-79c5-11f1-b498-9564c12b186d", # Nghị định 272/2026/NĐ-CP
        "399d9310-7e69-11f1-be38-974ae1f59c4b", # Thông tư 96/2026/TT-BTC
        "7f147190-5009-11f1-a1c0-795b56a45f32", # Thông tư 08/2026/TT-NHNN
        "28328cd0-4aba-11f1-954d-59440f3447aa", # Thông tư 24/2026/TT-BCT
        "56b78ef0-5269-11f1-9836-b95caea4a391", # Thông tư 05/2026/TT-BTP
        "a52b2d20-532d-11f1-8ff6-81b63e115254", # Thông tư 49/2026/TT-BCA
        "f6e54110-4b5c-11f1-8c06-b5d7fb756254", # Luật sửa đổi Cơ quan đại diện
        "ec5cde10-54bc-11f1-abb8-a5ee305e759c", # Luật Hộ tịch 03/2026
        "9f320090-54c9-11f1-aa08-59e3f5ee2be8", # Luật trợ giúp pháp lý 05/2026
        "168340",                                # Nghị định 66/2024/NĐ-CP
        "159432",                                # Nghị định 117/2022/NĐ-CP
        "153735",                                # Nghị định 11/2022/NĐ-CP
        "152572",                                # Nghị định 09/2022/NĐ-CP
        "151164",                                # Nghị định 53/2021/NĐ-CP
        "177562",                                # Thông tư 27/2025/TT-BCT
        "158765",                                # Thông tư 20/2022/TT-BKHĐT
        "66801"                                  # Thông tư 200/2014/TT-BTC
    ]

    print("--- 5 Years MOJ API Ingestion Crawler ---")
    conn = get_db_connection()
    cursor = conn.cursor()
    
    # Enable xml_content column migration if needed
    try:
        cursor.execute("ALTER TABLE legal_documents ADD COLUMN xml_content TEXT")
    except sqlite3.OperationalError:
        pass
        
    ingested_count = 0
    for doc_id in target_doc_ids:
        print(f"\n[*] Fetching document ID: {doc_id} ...")
        doc_data = fetch_document_data(doc_id)
        if doc_data:
            title = doc_data.get("title", f"Văn bản {doc_id}")
            print(f"[Success] Found: {title}")
            
            # Format XML
            xml_str = generate_xml_for_llm(doc_data)
            
            # Reconstruct plaintext for chunking and vector index
            full_text = clean_html_content(doc_data.get("documentContent", {}).get("content", ""))
            chunks = [full_text[i:i+1500] for i in range(0, len(full_text), 1200)]
            
            # Insert into database
            cursor.execute(
                "INSERT OR REPLACE INTO legal_documents (id, title, chunks_json, xml_content, updated_at) VALUES (?, ?, ?, ?, ?)",
                (doc_id, title, json.dumps(chunks, ensure_ascii=False), xml_str, datetime.utcnow().isoformat())
            )
            ingested_count += 1
            print(f"[Saved] Document ID {doc_id} saved to legal_documents cache.")
        else:
            print(f"[Skip] Document ID {doc_id} could not be retrieved from MOJ gateway.")
            
    conn.commit()
    conn.close()
    print(f"\n=== Ingestion Complete. Synced {ingested_count} documents to cached SQLite database ===")

if __name__ == "__main__":
    main()
