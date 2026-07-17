import os
import json
import sqlite3
import datetime
import xml.etree.ElementTree as ET
from bs4 import BeautifulSoup
from datasets import load_dataset
from sentence_transformers import SentenceTransformer

# Setup paths
pipeline_dir = os.path.dirname(os.path.abspath(__file__))
backend_dir = os.path.dirname(pipeline_dir)
db_path = os.path.join(backend_dir, "p2b_database.sqlite")
cache_path = os.path.join(backend_dir, "seed", "cached_embeddings.json")

print(f"Database path: {db_path}")
print(f"Cache path: {cache_path}")

def chunk_markdown(text: str, max_chars: int = 1500) -> list:
    if not text:
        return []
    paragraphs = text.split("\n\n")
    chunks = []
    current_chunk = []
    current_len = 0
    for p in paragraphs:
        p = p.strip()
        if not p:
            continue
        p_len = len(p)
        if current_len + p_len > max_chars and current_chunk:
            chunks.append("\n\n".join(current_chunk))
            current_chunk = [p]
            current_len = p_len
        else:
            current_chunk.append(p)
            current_len += p_len + 2
    if current_chunk:
        chunks.append("\n\n".join(current_chunk))
    return chunks

def main():
    # 1. Load HuggingFace dataset
    print("Loading HuggingFace dataset tmquan/vbpl-vn (train split)...")
    dataset = load_dataset("tmquan/vbpl-vn", split="train")
    print(f"Loaded dataset containing {len(dataset)} examples.")
    
    # 2. Filter central government documents from last 10 years
    print("Filtering central government documents from last 10 years (year >= 2016)...")
    filtered_rows = []
    for r in dataset:
        try:
            year_val = int(r.get("year") or 0)
        except ValueError:
            year_val = 0
            
        if year_val >= 2016 and r.get("scope") == "trung_uong":
            filtered_rows.append(r)
            
    print(f"Found {len(filtered_rows)} central recent documents.")
    
    # Sort by issue_date descending (newest first)
    print("Sorting by issue date...")
    filtered_rows.sort(key=lambda x: x.get("issue_date") or "", reverse=True)
    
    # Take top 5,000 documents
    target_docs = filtered_rows[:5000]
    print(f"Selected top {len(target_docs)} newest documents for indexing.")
    
    # 3. Connect to Database
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    
    # 4. Insert documents and build XML structure
    print("Inserting/updating documents in SQLite database...")
    new_chunks_count = 0
    all_chunks_text = []
    
    for idx, doc in enumerate(target_docs):
        # Determine a valid ID
        doc_id = doc.get("item_id")
        if not doc_id:
            # Fallback to doc_name or hash of title
            doc_id = doc.get("doc_name") or str(hash(doc.get("title") or ""))
            
        title = doc.get("title") or "Văn bản không có tiêu đề"
        markdown_text = doc.get("markdown") or ""
        doc_number_list = doc.get("doc_number")
        doc_number = doc_number_list[0] if doc_number_list and len(doc_number_list) > 0 else "N/A"
        legal_type = doc.get("legal_type") or "Văn bản pháp luật"
        issuing_authority = doc.get("issuing_authority") or "N/A"
        issue_date = doc.get("issue_date") or "N/A"
        
        # Parse text into chunks
        chunks = chunk_markdown(markdown_text)
        if not chunks:
            chunks = [title] # Fallback if empty
            
        # Reconstruct basic XML structure
        root_el = ET.Element("document")
        root_el.set("id", doc_id)
        root_el.set("number", doc_number)
        root_el.set("type", legal_type)
        root_el.set("status", "Còn hiệu lực")
        root_el.set("issuing_body", issuing_authority)
        root_el.set("issued_at", issue_date)
        
        title_el = ET.SubElement(root_el, "title")
        title_el.text = title
        
        content_el = ET.SubElement(root_el, "content")
        content_el.text = markdown_text
        
        xml_str = BeautifulSoup(ET.tostring(root_el, encoding="utf-8"), "xml").prettify()
        
        # Save chunks info
        chunks_json = json.dumps(chunks, ensure_ascii=False)
        
        cursor.execute(
            "INSERT OR REPLACE INTO legal_documents (id, title, chunks_json, xml_content, updated_at) VALUES (?, ?, ?, ?, ?)",
            (doc_id, title, chunks_json, xml_str, datetime.datetime.utcnow().isoformat())
        )
        
        for c in chunks:
            all_chunks_text.append(c.strip())
            
        if (idx + 1) % 500 == 0:
            print(f"  Processed {idx + 1}/5000 documents...")
            
    conn.commit()
    conn.close()
    print("Database documents and chunks populated successfully!")
    
    # 5. Embed chunks in batches using local SentenceTransformer
    print("Loading local SentenceTransformer model (intfloat/multilingual-e5-base)...")
    model = SentenceTransformer('intfloat/multilingual-e5-base')
    
    # Load existing cache
    embeddings_cache = {}
    if os.path.exists(cache_path):
        try:
            with open(cache_path, "r", encoding="utf-8") as f:
                embeddings_cache = json.load(f)
            print(f"Loaded existing cache containing {len(embeddings_cache)} embeddings.")
        except Exception as e:
            print(f"Failed to load existing cache: {e}. Starting fresh.")
            
    # Find missing embeddings
    missing_prefixed = []
    missing_raw = []
    for chunk in all_chunks_text:
        prefixed = "passage: " + chunk
        if prefixed not in embeddings_cache:
            missing_raw.append(chunk)
            missing_prefixed.append(prefixed)
            
    print(f"Found {len(missing_prefixed)} new/missing chunks to embed.")
    
    if len(missing_prefixed) > 0:
        print("Computing embeddings in batches of 128...")
        try:
            # Run batch embedding
            embs = model.encode(missing_prefixed, batch_size=128, normalize_embeddings=True, show_progress_bar=True)
            embeddings = embs.tolist()
            
            # Save to cache
            for prefixed, emb in zip(missing_prefixed, embeddings):
                embeddings_cache[prefixed] = emb
                
            # Write cache back to disk
            with open(cache_path, "w", encoding="utf-8") as f:
                json.dump(embeddings_cache, f, ensure_ascii=False)
            print(f"Embeddings cache successfully updated and saved to disk. Total cached: {len(embeddings_cache)}")
        except Exception as e:
            print(f"Failed to compute/save embeddings: {e}")
    else:
        print("No new chunks to embed. Cache is up to date!")
        
    print("All tasks completed successfully!")

if __name__ == "__main__":
    main()
