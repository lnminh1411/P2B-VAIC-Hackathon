import os
import re
import sys
import hashlib
import argparse
import requests
import json
from datetime import datetime
from bs4 import BeautifulSoup
from tqdm import tqdm

# Ensure stdout prints UTF-8 on Windows
try:
    sys.stdout.reconfigure(encoding='utf-8')
except Exception:
    pass

import torch
from sentence_transformers import SentenceTransformer
from datasets import load_dataset
import psycopg2
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

def is_relevant(title, agency):
    if not title or not agency:
        return False
    
    title_lower = title.lower()
    agency_lower = agency.lower()
    
    # Check agency whitelist
    agency_matched = any(a in agency_lower for a in WHITELIST_AGENCIES)
    if not agency_matched:
        return False
        
    # Check title keywords
    keywords_matched = any(kw in title_lower for kw in WHITELIST_KEYWORDS)
    return keywords_matched

def clean_html_text(html_content):
    if not html_content:
        return ""
    soup = BeautifulSoup(html_content, "lxml")
    return soup.get_text("\n")

def parse_articles_from_html(html_content):
    """
    Parse HTML content and split it semantically at article boundaries (Điều X).
    """
    if not html_content:
        return []
        
    soup = BeautifulSoup(html_content, "lxml")
    paragraphs = soup.find_all(['p', 'div', 'tr'])
    
    chunks = []
    current_article = []
    current_title = "Giới thiệu"
    
    article_pattern = re.compile(r'^(điều\s+\d+|article\s+\d+)', re.IGNORECASE)
    
    for p in paragraphs:
        text = p.get_text(" ").strip()
        if not text:
            continue
            
        # Check if this paragraph starts a new Article
        if article_pattern.match(text):
            if current_article:
                chunks.append((current_title, "\n".join(current_article)))
                current_article = []
            
            # Extract article title (e.g. "Điều 1")
            match = article_pattern.match(text)
            current_title = match.group(1)
            
        current_article.append(text)
        
    if current_article:
        chunks.append((current_title, "\n".join(current_article)))
        
    # If no article structure was detected, fall back to basic paragraph grouping
    if not chunks:
        full_text = clean_html_text(html_content)
        lines = [line.strip() for line in full_text.split("\n") if line.strip()]
        # Group lines into semantic chunks of approx 1000 characters
        current_chunk = []
        current_len = 0
        chunk_idx = 1
        for line in lines:
            current_chunk.append(line)
            current_len += len(line)
            if current_len >= 1000:
                chunks.append((f"Mục {chunk_idx}", "\n".join(current_chunk)))
                current_chunk = []
                current_len = 0
                chunk_idx += 1
        if current_chunk:
            chunks.append((f"Mục {chunk_idx}", "\n".join(current_chunk)))
            
    return chunks

def parse_articles_from_markdown(markdown_text):
    """
    Parse Markdown content and split it at Article (Điều X) boundaries.
    """
    if not markdown_text:
        return []
        
    lines = markdown_text.split("\n")
    chunks = []
    current_article = []
    current_title = "Giới thiệu"
    
    article_pattern = re.compile(r'^(điều\s+\d+|article\s+\d+)', re.IGNORECASE)
    
    for line in lines:
        line_stripped = line.strip()
        if not line_stripped:
            continue
            
        # Clean markdown bold/italic formatting around the start of the line
        clean_start = re.sub(r'^[\*#_\s\-]+', '', line_stripped)
        
        if article_pattern.match(clean_start):
            if current_article:
                chunks.append((current_title, "\n".join(current_article)))
                current_article = []
            match = article_pattern.match(clean_start)
            current_title = match.group(1)
            
        current_article.append(line_stripped)
        
    if current_article:
        chunks.append((current_title, "\n".join(current_article)))
        
    return chunks

def fetch_document_body(api_url):
    """
    Fetches raw document from Ministry of Justice (MoJ) gateway.
    """
    try:
        headers = {"User-Agent": "Mozilla/5.0"}
        response = requests.get(api_url, headers=headers, timeout=15)
        if response.status_code == 200:
            res_data = response.json()
            if res_data.get("success") and "data" in res_data:
                doc_data = res_data["data"]
                
                # Extract effect status and date details
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

def main():
    parser = argparse.ArgumentParser(description="P2B Legal Document Ingestion Pipeline")
    parser.add_argument("--limit", type=int, default=None, help="Limit the number of ingested documents")
    parser.add_argument("--dry-run", action="store_true", help="Run without writing to database")
    parser.add_argument("--device", type=str, default=None, help="Torch device: cuda or cpu (defaults to cuda if available)")
    args = parser.parse_args()

    # Determine device
    device = args.device
    if not device:
        device = "cuda" if torch.cuda.is_available() else "cpu"
    print(f"Using device: {device}")

    # Load local E5 model
    print("Loading multilingual-e5-base embedding model...")
    model = SentenceTransformer("intfloat/multilingual-e5-base", device=device)

    # DATABASE CONNECTION
    db_conn = None
    if not args.dry_run:
        # Load local .env file if DATABASE_URL is not set
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
            db_conn = psycopg2.connect(db_url)
            print("Successfully connected to Railway PostgreSQL database.")
        except Exception as e:
            print(f"Database connection failed: {e}")
            sys.exit(1)

    # STREAM DATASET
    print("Streaming tmquan/vbpl-vn dataset from Hugging Face...")
    ds = load_dataset("tmquan/vbpl-vn", split="train", streaming=True)

    success_count = 0
    checked_count = 0

    pbar = tqdm(desc="Processing Decrees")

    for row in ds:
        checked_count += 1
        
        # 1. Scope / Ingestion Filters
        title = row.get("title")
        issuing_authority = row.get("issuing_authority")
        doc_type = row.get("doc_type", "")
        
        if not doc_type or "nghi_dinh" not in doc_type.lower():
            continue
            
        if not is_relevant(title, issuing_authority):
            continue
            
        # 2. Check Validity / Effect Status
        # Parse extracted metadata
        extracted_json = row.get("extracted_json")
        status = None
        if extracted_json:
            try:
                ext = json.loads(extracted_json) if isinstance(extracted_json, str) else extracted_json
                status = ext.get("status")
            except Exception:
                pass
                
        # Fetch detailed fields from MoJ gateway if necessary
        api_url = row.get("api_url")
        fetched_html, fetched_status, eff_from, eff_to = None, None, None, None
        
        # Check status
        doc_status = status or fetched_status
        if doc_status and doc_status in ["Hết hiệu lực"]:
            continue
            
        # 3. Resolve Missing Text Body
        markdown_body = row.get("markdown", "")
        doc_html = None
        
        # If markdown body is missing or too short, fetch from gateway
        if not markdown_body or len(markdown_body.strip()) < 100:
            if api_url:
                fetched_html, fetched_status, eff_from, eff_to = fetch_document_body(api_url)
                if fetched_status and fetched_status in ["Hết hiệu lực"]:
                    continue
                doc_html = fetched_html
            
            if not doc_html:
                # No body available and gateway fetch failed, skip
                continue
                
        # Skip if expired based on date fields
        expiration_date = row.get("expiration_date") or eff_to
        if expiration_date:
            try:
                # Parse expiration date
                if isinstance(expiration_date, str):
                    exp_date = datetime.strptime(expiration_date.split("T")[0], "%Y-%m-%d").date()
                    if exp_date < datetime.now().date():
                        # Document has expired, skip
                        continue
            except Exception:
                pass

        # 4. Article-level Chunking
        if doc_html:
            chunks = parse_articles_from_html(doc_html)
        else:
            chunks = parse_articles_from_markdown(markdown_body)
            
        if not chunks:
            continue
            
        # Re-verify full document body text exists
        full_document_text = "\n\n".join([c[1] for c in chunks])
        content_hash = hashlib.sha256(full_document_text.encode("utf-8")).hexdigest()

        # 5. Generate E5 Embeddings locally on GPU
        # E5 model requires "passage: " prefix for document chunks
        passage_texts = [f"passage: {c[1]}" for c in chunks]
        embeddings = model.encode(passage_texts, batch_size=32, show_progress_bar=False)

        # 6. Database Insertion
        if not args.dry_run and db_conn:
            try:
                with db_conn.cursor() as cursor:
                    # Map doc_number from list
                    doc_number_list = row.get("doc_number", [])
                    doc_number = doc_number_list[0] if doc_number_list else None
                    canonical_url = row.get("source_url") or f"https://vbpl.vn/doc-{row.get('doc_name')}"
                    
                    # A. Insert legal_documents
                    cursor.execute("""
                        INSERT INTO legal_documents (canonical_url, issuing_agency, document_number)
                        VALUES (%s, %s, %s)
                        ON CONFLICT (canonical_url) DO UPDATE 
                        SET issuing_agency = EXCLUDED.issuing_agency, document_number = EXCLUDED.document_number
                        RETURNING id
                    """, (canonical_url, issuing_authority, doc_number))
                    legal_doc_id = cursor.fetchone()[0]

                    # B. Insert document_versions
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

                    # C. Delete old chunks if version already existed
                    cursor.execute("DELETE FROM document_chunks WHERE document_version_id = %s", (version_id,))

                    # D. Batch Insert document_chunks
                    chunk_rows = []
                    for idx, (title_text, body_text) in enumerate(chunks):
                        emb_list = embeddings[idx].tolist()
                        chunk_rows.append((version_id, idx, f"{title_text}\n{body_text}", emb_list, "multilingual-e5-base"))

                    execute_values(cursor, """
                        INSERT INTO document_chunks (document_version_id, ordinal, content, embedding, embedding_model)
                        VALUES %s
                    """, chunk_rows)

                db_conn.commit()
            except Exception as dbe:
                db_conn.rollback()
                print(f"\nFailed to write document {row.get('doc_name')} to database: {dbe}")
                continue

        success_count += 1
        pbar.update(1)
        pbar.set_postfix({"ingested": success_count, "last": row.get("doc_name")})

        if args.limit and success_count >= args.limit:
            break

    pbar.close()
    if db_conn:
        db_conn.close()
        
    print(f"\nIngestion finished. Successfully processed {success_count} documents.")

if __name__ == "__main__":
    main()
