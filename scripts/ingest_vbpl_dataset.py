import os
import re
import sys
import hashlib
import argparse
import requests
import json
import time
import threading
from datetime import datetime
from bs4 import BeautifulSoup
from tqdm import tqdm
from concurrent.futures import ThreadPoolExecutor, as_completed

# Ensure stdout prints UTF-8 on Windows
try:
    sys.stdout.reconfigure(encoding='utf-8')
except Exception:
    pass

import torch
from sentence_transformers import SentenceTransformer
from datasets import load_dataset
import psycopg2
from psycopg2.pool import ThreadedConnectionPool
from psycopg2.extras import execute_values

# Whitelists for filtering decrees
WHITELIST_AGENCIES = [
    "chính phủ",
    "bộ tài chính",
    "bộ kế hoạch và đầu tư",
    "bộ khoa học và công nghệ",
    "ngân hàng nhà nước",
    "bộ công thương",
    "bộ thông tin và truyền thông"
]

WHITELIST_KEYWORDS = [
    "hỗ trợ",
    "doanh nghiệp",
    "thuế",
    "ưu đãi",
    "tín dụng",
    "công nghệ",
    "vốn",
    "đầu tư",
    "khởi nghiệp",
    "chương trình",
    "phát triển"
]

# Global locks and variables
gpu_lock = threading.Lock()
stats_lock = threading.Lock()
success_count = 0
checked_count = 0

def is_relevant(title, agency):
    if not title or not agency:
        return False
    
    title_lower = title.lower()
    agency_lower = agency.lower()
    
    agency_matched = any(a in agency_lower for a in WHITELIST_AGENCIES)
    if not agency_matched:
        return False
        
    keywords_matched = any(kw in title_lower for kw in WHITELIST_KEYWORDS)
    return keywords_matched

def clean_html_text(html_content):
    if not html_content:
        return ""
    soup = BeautifulSoup(html_content, "lxml")
    return soup.get_text("\n")

def parse_articles_robust(text):
    if not text:
        return []
        
    pattern = re.compile(r'\b(điều\s+\d+|article\s+\d+)\b', re.IGNORECASE)
    
    matches = list(pattern.finditer(text))
    if not matches:
        return [("Giới thiệu", text)]
        
    valid_split_indices = []
    
    for i, match in enumerate(matches):
        start, end = match.span()
        matched_str = match.group(1)
        
        before = text[max(0, start-30):start].lower().strip()
        
        is_reference = False
        ref_patterns = [
            r'\bt[aạ]i\s*$',
            r'\btheo\s*$',
            r'\bkh[oỏọôổ]an\s*\d*\s*$',
            r'\bđ[iỉíịêếể]m\s*[a-zđ0-9\s]*$',
            r'\bc[uủ]a\s*$',
            r'\bv[aà]\s*$',
            r'\bc[aá]c\s*$',
            r'\bquy\s+định\s+tại\s*$',
            r'\bđược\s+quy\s+định\s+tại\s*$'
        ]
        for ref_pat in ref_patterns:
            if re.search(ref_pat, before):
                is_reference = True
                break
                
        if not is_reference:
            valid_split_indices.append((start, end, matched_str))
            
    chunks = []
    if not valid_split_indices:
        return [("Giới thiệu", text)]
        
    first_start = valid_split_indices[0][0]
    intro_content = text[:first_start].strip()
    if intro_content:
        chunks.append(("Giới thiệu", intro_content))
        
    for idx in range(len(valid_split_indices)):
        start_curr, end_curr, title_curr = valid_split_indices[idx]
        end_next = valid_split_indices[idx+1][0] if idx+1 < len(valid_split_indices) else len(text)
        article_content = text[start_curr:end_next].strip()
        chunks.append((title_curr.capitalize(), article_content))
        
    return chunks

def parse_articles_from_html(html_content):
    if not html_content:
        return []
    soup = BeautifulSoup(html_content, "lxml")
    for script in soup(["script", "style"]):
        script.decompose()
    text = soup.get_text(" ")
    text = re.sub(r'[ \t\r\f\v]+', ' ', text)
    text = re.sub(r'\n+', '\n', text)
    return parse_articles_robust(text)

def parse_articles_from_markdown(markdown_text):
    if not markdown_text:
        return []
    clean_md = re.sub(r'[\*#_]', '', markdown_text)
    return parse_articles_robust(clean_md)

def fetch_document_body(api_url):
    try:
        headers = {"User-Agent": "Mozilla/5.0"}
        response = requests.get(api_url, headers=headers, timeout=15)
        if response.status_code == 200:
            res_data = response.json()
            if res_data.get("success") and "data" in res_data:
                doc_data = res_data["data"]
                
                status_name = None
                if doc_data.get("effStatus"):
                    status_name = doc_data["effStatus"].get("name")
                    
                eff_from = doc_data.get("effFrom")
                eff_to = doc_data.get("effTo")
                
                content_html = ""
                if doc_data.get("documentContent"):
                    content_html = doc_data["documentContent"].get("content", "")
                    
                return content_html, status_name, eff_from, eff_to
    except Exception as e:
        print(f"Error fetching from MoJ gateway ({api_url}): {e}")
    return None, None, None, None

def process_row(row, db_pool, model, dry_run, pbar):
    global success_count, checked_count
    
    # 1. Scope / Ingestion Filters
    title = row.get("title")
    issuing_authority = row.get("issuing_authority")
    doc_type = row.get("doc_type", "")
    
    if not doc_type or "nghi_dinh" not in doc_type.lower():
        return False
        
    if not is_relevant(title, issuing_authority):
        return False
        
    # 2. Check Validity / Effect Status
    extracted_json = row.get("extracted_json")
    status = None
    if extracted_json:
        try:
            ext = json.loads(extracted_json) if isinstance(extracted_json, str) else extracted_json
            status = ext.get("status")
        except Exception:
            pass
            
    api_url = row.get("api_url")
    fetched_html, fetched_status, eff_from, eff_to = None, None, None, None
    
    doc_status = status or fetched_status
    if doc_status and doc_status in ["Hết hiệu lực"]:
        return False
        
    # 3. Resolve Missing Text Body
    markdown_body = row.get("markdown", "")
    doc_html = None
    
    if not markdown_body or len(markdown_body.strip()) < 100:
        if api_url:
            fetched_html, fetched_status, eff_from, eff_to = fetch_document_body(api_url)
            if fetched_status and fetched_status in ["Hết hiệu lực"]:
                return False
            doc_html = fetched_html
        
        if not doc_html:
            return False
            
    expiration_date = row.get("expiration_date") or eff_to
    if expiration_date:
        try:
            if isinstance(expiration_date, str):
                exp_date = datetime.strptime(expiration_date.split("T")[0], "%Y-%m-%d").date()
                if exp_date < datetime.now().date():
                    return False
        except Exception:
            pass

    # 4. Article-level Chunking
    if doc_html:
        chunks = parse_articles_from_html(doc_html)
    else:
        chunks = parse_articles_from_markdown(markdown_body)
        
    if not chunks:
        return False
        
    full_document_text = "\n\n".join([c[1] for c in chunks])
    content_hash = hashlib.sha256(full_document_text.encode("utf-8")).hexdigest()

    # 5. Generate E5 Embeddings locally on GPU (thread-safe lock)
    passage_texts = [f"passage: {c[1]}" for c in chunks]
    with gpu_lock:
        embeddings = model.encode(passage_texts, batch_size=32, show_progress_bar=False)

    # 6. Database Insertion
    if not dry_run and db_pool:
        conn = db_pool.getconn()
        try:
            with conn.cursor() as cursor:
                doc_number_list = row.get("doc_number", [])
                doc_number = doc_number_list[0] if doc_number_list else None
                canonical_url = row.get("source_url") or f"https://vbpl.vn/doc-{row.get('doc_name')}"
                
                cursor.execute("""
                    INSERT INTO legal_documents (canonical_url, issuing_agency, document_number)
                    VALUES (%s, %s, %s)
                    ON CONFLICT (canonical_url) DO UPDATE 
                    SET issuing_agency = EXCLUDED.issuing_agency, document_number = EXCLUDED.document_number
                    RETURNING id
                """, (canonical_url, issuing_authority, doc_number))
                legal_doc_id = cursor.fetchone()[0]

                effective_from = row.get("issue_date") or eff_from
                if effective_from and "T" in str(effective_from):
                    effective_from = str(effective_from).split("T")[0]
                    
                effective_to = expiration_date
                if effective_to and "T" in str(effective_to):
                    effective_to = str(effective_to).split("T")[0]
                    
                raw_object_key = f"hf-ingestion/{row.get('doc_name')}"

                cursor.execute("""
                    INSERT INTO document_versions (legal_document_id, version, content_hash, raw_object_key, effective_from, effective_to, crawled_at)
                    VALUES (%s, 1, %s, %s, %s, %s, now())
                    ON CONFLICT (legal_document_id, version) DO UPDATE 
                    SET content_hash = EXCLUDED.content_hash, raw_object_key = EXCLUDED.raw_object_key
                    RETURNING id
                """, (legal_doc_id, content_hash, raw_object_key, effective_from or None, effective_to or None))
                version_id = cursor.fetchone()[0]

                cursor.execute("DELETE FROM document_chunks WHERE document_version_id = %s", (version_id,))

                chunk_rows = []
                for idx, (title_text, body_text) in enumerate(chunks):
                    emb_list = embeddings[idx].tolist()
                    chunk_content = f"{title_text}\n{body_text}" if title_text != body_text else body_text
                    chunk_rows.append((version_id, idx, chunk_content, emb_list, "multilingual-e5-base"))

                execute_values(cursor, """
                    INSERT INTO document_chunks (document_version_id, ordinal, content, embedding, embedding_model)
                    VALUES %s
                """, chunk_rows)

            conn.commit()
            db_pool.putconn(conn)
        except Exception as dbe:
            conn.rollback()
            db_pool.putconn(conn)
            print(f"\nFailed to write document {row.get('doc_name')} to database: {dbe}")
            return False

    with stats_lock:
        success_count += 1
        pbar.update(1)
        pbar.set_postfix({"ingested": success_count, "last": row.get("doc_name")})
        
    return True

def main():
    parser = argparse.ArgumentParser(description="P2B Legal Document Ingestion Pipeline")
    parser.add_argument("--limit", type=int, default=None, help="Limit the number of ingested documents")
    parser.add_argument("--dry-run", action="store_true", help="Run without writing to database")
    parser.add_argument("--device", type=str, default=None, help="Torch device: cuda or cpu")
    args = parser.parse_args()

    device = args.device
    if not device:
        device = "cuda" if torch.cuda.is_available() else "cpu"
    print(f"Using device: {device}")

    print("Loading multilingual-e5-base embedding model...")
    model = SentenceTransformer("intfloat/multilingual-e5-base", device=device)

    # DATABASE CONNECTION POOL
    db_pool = None
    if not args.dry_run:
        if not os.getenv("DATABASE_URL") and os.path.exists(".env"):
            with open(".env", "r", encoding="utf-8") as f:
                for line in f:
                    if line.strip() and not line.startswith("#") and "=" in line:
                        k, v = line.strip().split("=", 1)
                        os.environ[k.strip()] = v.strip()
                        
        db_url = os.getenv("DATABASE_URL")
        if not db_url:
            print("DATABASE_URL environment variable is required when not in --dry-run mode")
            sys.exit(1)
        try:
            db_pool = ThreadedConnectionPool(minconn=1, maxconn=10, dsn=db_url)
            print("Successfully established database connection pool.")
        except Exception as e:
            print(f"Database connection pool setup failed: {e}")
            sys.exit(1)

    print("Streaming tmquan/vbpl-vn dataset from Hugging Face...")
    ds = load_dataset("tmquan/vbpl-vn", split="train", streaming=True)

    max_workers = 5
    print(f"Starting multi-threaded execution queue with {max_workers} threads...")
    pbar = tqdm(desc="Processing Decrees")
    
    futures = set()
    
    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        for row in ds:
            # Quick check if it matches basic decree criteria to avoid pushing irrelevant items to worker threads
            title = row.get("title", "")
            agency = row.get("issuing_authority", "")
            doc_type = row.get("doc_type", "")
            
            if not doc_type or "nghi_dinh" not in doc_type.lower():
                continue
            if not is_relevant(title, agency):
                continue
                
            # Maintain maximum of 10 concurrent active ingestion tasks in queue
            while len(futures) >= 10:
                completed = {f for f in futures if f.done()}
                futures -= completed
                if len(futures) >= 10:
                    time.sleep(0.02)
                    
            futures.add(executor.submit(process_row, row, db_pool, model, args.dry_run, pbar))
            
            if args.limit and success_count >= args.limit:
                break
                
        # Wait for any remaining tasks to finish
        for fut in as_completed(futures):
            pass

    pbar.close()
    if db_pool:
        db_pool.closeall()
        
    print(f"\nIngestion finished. Successfully processed {success_count} documents.")

if __name__ == "__main__":
    main()
