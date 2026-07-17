import os
import json
import numpy as np
from typing import List, Any
from rank_bm25 import BM25Okapi
from sentence_transformers import SentenceTransformer
from app.schemas.policy import PolicyOpportunity
from app.schemas.passport import CompanyPassport

def cosine_similarity(v1: List[float], v2: List[float]) -> float:
    a = np.array(v1)
    b = np.array(v2)
    norm_a = np.linalg.norm(a)
    norm_b = np.linalg.norm(b)
    if norm_a == 0 or norm_b == 0:
        return 0.0
    return float(np.dot(a, b) / (norm_a * norm_b))

class HybridRetrievalEngine:
    def __init__(self, seed_dir: str):
        self.seed_dir = seed_dir
        self.opportunities_path = os.path.join(seed_dir, "policy_opportunities.json")
        self.legal_corpus_path = os.path.join(seed_dir, "legal_corpus.json")
        self.cache_path = os.path.join(seed_dir, "cached_embeddings.json")
        
        # Check if we should use Gemini API for embeddings
        self.use_gemini = False
        self.gemini_client = None
        if os.environ.get("GEMINI_API_KEY"):
            try:
                from google import genai
                self.gemini_client = genai.Client()
                self.use_gemini = True
                self.cache_path = os.path.join(seed_dir, "cached_embeddings_gemini.json")
                print("Using Google Gemini API (text-embedding-004) for embeddings.")
            except ImportError:
                print("[Warning] google-genai library not found. Falling back to local SentenceTransformer.")

        # Load cached embeddings if available (fallback)
        self.embeddings_cache = {}
        
        def decompress_cache():
            gz_path = self.cache_path + ".gz"
            
            # Merge split parts if full gz file is missing
            if not os.path.exists(gz_path):
                part_num = 1
                parts = []
                while True:
                    part_name = f"{gz_path}.part{part_num:03d}"
                    if not os.path.exists(part_name):
                        break
                    parts.append(part_name)
                    part_num += 1
                if parts:
                    print(f"Merging {len(parts)} split cache parts into {gz_path}...")
                    tmp_gz = gz_path + ".tmp"
                    try:
                        with open(tmp_gz, 'wb') as dest:
                            for p in parts:
                                with open(p, 'rb') as src:
                                    dest.write(src.read())
                        os.replace(tmp_gz, gz_path)
                        print("Cache parts merged successfully.")
                    except Exception as e:
                        print(f"[Retrieval Error] Failed to merge cache parts: {e}")
                        if os.path.exists(tmp_gz):
                            try:
                                os.remove(tmp_gz)
                            except Exception:
                                pass

            if os.path.exists(gz_path):
                print(f"Decompressing embeddings cache from {gz_path}...")
                import gzip
                import shutil
                tmp_path = self.cache_path + ".tmp"
                try:
                    with gzip.open(gz_path, 'rb') as f_in:
                        with open(tmp_path, 'wb') as f_out:
                            shutil.copyfileobj(f_in, f_out)
                    os.replace(tmp_path, self.cache_path)
                    print("Cache decompressed successfully.")
                except Exception as e:
                    print(f"[Retrieval Error] Failed to decompress cache: {e}")
                    if os.path.exists(tmp_path):
                        try:
                            os.remove(tmp_path)
                        except Exception:
                            pass

        if not os.path.exists(self.cache_path):
            decompress_cache()

        if os.path.exists(self.cache_path):
            try:
                with open(self.cache_path, "r", encoding="utf-8") as f:
                    self.embeddings_cache = json.load(f)
            except Exception as e:
                print(f"[Retrieval Warning] Cache file corrupted ({e}). Retrying decompression...")
                try:
                    os.remove(self.cache_path)
                except Exception:
                    pass
                decompress_cache()
                if os.path.exists(self.cache_path):
                    try:
                        with open(self.cache_path, "r", encoding="utf-8") as f:
                            self.embeddings_cache = json.load(f)
                    except Exception as e2:
                        print(f"[Retrieval Error] Redecompression failed: {e2}")
                
        self.model = None
        if not self.use_gemini:
            # Initialize local SentenceTransformer for multilingual-e5-base (now cached locally)
            print("Loading local SentenceTransformer (intfloat/multilingual-e5-base)...")
            try:
                self.model = SentenceTransformer('intfloat/multilingual-e5-base')
            except Exception as e:
                print(f"[Warning] Failed to load local SentenceTransformer: {e}. Falling back to cached embeddings only.")
                self.model = None

        # Load opportunities and documents dynamically
        self.load_opportunities_and_documents()

    def load_opportunities_and_documents(self):
        """
        Loads all opportunities and legal documents from static seed files and SQLite tables.
        Re-builds decree chunks and BM25 index.
        """
        # 1. Load static opportunities
        opportunities = {}
        try:
            if os.path.exists(self.opportunities_path):
                with open(self.opportunities_path, "r", encoding="utf-8") as f:
                    opps_data = json.load(f)
                    opportunities = {opp["id"]: PolicyOpportunity(**opp) for opp in opps_data}
        except Exception as e:
            print(f"[Retrieval Warning] Failed to load static opportunities: {e}")
            
        # 2. Load from SQLite policy_opportunities table
        try:
            from app.engine.db import get_db_connection
            conn = get_db_connection()
            cursor = conn.cursor()
            cursor.execute("SELECT id, data_json FROM policy_opportunities")
            for row in cursor.fetchall():
                try:
                    opportunities[row["id"]] = PolicyOpportunity(**json.loads(row["data_json"]))
                except Exception:
                    pass
            conn.close()
        except Exception as e:
            print(f"[Retrieval Warning] Failed to load opportunities from SQLite: {e}")
            
        self.opportunities = list(opportunities.values())
        
        # 3. Load static legal documents (decrees)
        docs_dict = {}
        try:
            if os.path.exists(self.legal_corpus_path):
                with open(self.legal_corpus_path, "r", encoding="utf-8") as f:
                    legal_docs = json.load(f)
                    docs_dict = {doc["id"]: doc for doc in legal_docs}
        except Exception as e:
            print(f"[Retrieval Warning] Failed to load static legal corpus: {e}")
            
        # 4. Load from SQLite legal_documents table
        try:
            from app.engine.db import get_db_connection
            conn = get_db_connection()
            cursor = conn.cursor()
            cursor.execute("SELECT id, title, chunks_json FROM legal_documents")
            for row in cursor.fetchall():
                try:
                    doc_id = row["id"]
                    docs_dict[doc_id] = {
                        "id": doc_id,
                        "title": row["title"],
                        "chunks": json.loads(row["chunks_json"])
                    }
                except Exception:
                    pass
            conn.close()
        except Exception as e:
            print(f"[Retrieval Warning] Failed to load legal documents from SQLite: {e}")
            
        # 5. Build chunks corpus
        self.chunks = []
        for doc_id, doc in docs_dict.items():
            for chunk in doc.get("chunks", []):
                self.chunks.append({
                    "doc_id": doc_id,
                    "text": chunk
                })
                
        # 6. Initialize BM25 lexical index
        self.bm25_corpus = [c["text"].lower().split() for c in self.chunks]
        if self.bm25_corpus:
            self.bm25 = BM25Okapi(self.bm25_corpus)
        else:
            self.bm25 = None
            
        # 7. Pre-warm vector cache
        self.pre_warm_cache()

    def pre_warm_cache(self):
        """
        Pre-embeds all decree chunks using active model in batch if they aren't already cached.
        """
        missing_chunks = []
        missing_prefixed = []
        
        for chunk in self.chunks:
            chunk_text = chunk["text"].strip()
            prefixed_text = "passage: " + chunk_text
            if prefixed_text not in self.embeddings_cache:
                missing_chunks.append(chunk_text)
                missing_prefixed.append(prefixed_text)
                
        if not missing_prefixed:
            return
            
        print(f"Pre-warming embeddings cache: embedding {len(missing_prefixed)} new chunks in batch...")
        
        embeddings = []
        
        if self.use_gemini and self.gemini_client:
            try:
                batch_size = 100
                for i in range(0, len(missing_prefixed), batch_size):
                    batch = missing_prefixed[i:i+batch_size]
                    response = self.gemini_client.models.embed_content(
                        model="text-embedding-004",
                        contents=batch,
                    )
                    embeddings.extend([emb.values for emb in response.embeddings])
            except Exception as e:
                print(f"[Retrieval Warning] Gemini batch embedding text-embedding-004 failed: {e}. Trying embedding-001...")
                try:
                    embeddings = []
                    batch_size = 100
                    for i in range(0, len(missing_prefixed), batch_size):
                        batch = missing_prefixed[i:i+batch_size]
                        response = self.gemini_client.models.embed_content(
                            model="embedding-001",
                            contents=batch,
                        )
                        embeddings.extend([emb.values for emb in response.embeddings])
                except Exception as e2:
                    print(f"[Retrieval Error] Gemini batch embedding-001 failed: {e2}. Populating zero vector fallbacks.")
                    embeddings = [[0.0] * 768] * len(missing_prefixed)
                
        elif self.model:
            try:
                embs = self.model.encode(missing_prefixed, batch_size=32, normalize_embeddings=True, show_progress_bar=False)
                embeddings = embs.tolist()
            except Exception as e:
                print(f"[Retrieval Error] Local batch embedding failed: {e}")
                
        if embeddings and len(embeddings) == len(missing_prefixed):
            for prefixed_text, emb in zip(missing_prefixed, embeddings):
                self.embeddings_cache[prefixed_text] = emb
            self.save_cache_to_disk()
            print(f"Embeddings cache successfully pre-warmed and saved to disk. Total cached: {len(self.embeddings_cache)}")
        else:
            print("[Retrieval Warning] Batch embedding did not complete successfully or returned mismatched size.")

    def get_embedding(self, text: str, is_query: bool = False) -> List[float]:
        """
        Gets text embedding. Uses Gemini API if active, otherwise local SentenceTransformer.
        Prepends 'query: ' or 'passage: ' as required by multilingual-e5 models.
        """
        cleaned_text = text.strip()
        prefix = "query: " if is_query else "passage: "
        prefixed_text = prefix + cleaned_text
        
        # Check cache first
        if prefixed_text in self.embeddings_cache:
            return self.embeddings_cache[prefixed_text]
            
        if self.use_gemini and self.gemini_client:
            try:
                response = self.gemini_client.models.embed_content(
                    model="text-embedding-004",
                    contents=prefixed_text,
                )
                embedding = response.embeddings[0].values
                self.embeddings_cache[prefixed_text] = embedding
                self.save_cache_to_disk()
                return embedding
            except Exception as e:
                print(f"[Retrieval Warning] Gemini text-embedding-004 failed: {e}. Trying embedding-001...")
                try:
                    response = self.gemini_client.models.embed_content(
                        model="embedding-001",
                        contents=prefixed_text,
                    )
                    embedding = response.embeddings[0].values
                    self.embeddings_cache[prefixed_text] = embedding
                    self.save_cache_to_disk()
                    return embedding
                except Exception as e2:
                    print(f"[Retrieval Warning] Gemini embedding-001 failed: {e2}. Caching zero vector fallback.")
                    # CACHE zero vector fallback in dictionary so subsequent lookups hit memory cache instantly!
                    self.embeddings_cache[prefixed_text] = [0.0] * 768
                    return [0.0] * 768

        if self.model:
            try:
                # E5 models are trained to output normalized embeddings
                emb = self.model.encode(prefixed_text, normalize_embeddings=True)
                embedding = emb.tolist()
                # Cache it in memory and save to disk
                self.embeddings_cache[prefixed_text] = embedding
                self.save_cache_to_disk()
                return embedding
            except Exception as e:
                print(f"[Retrieval Warning] Local embedding failed: {e}.")
                
        # Fallback to plain text cache lookup (if cached without prefix)
        if cleaned_text in self.embeddings_cache:
            return self.embeddings_cache[cleaned_text]
            
        # Zero fallback
        return [0.0] * 768

    def save_cache_to_disk(self):
        """
        Saves current memory embedding cache to seed directory.
        """
        with open(self.cache_path, "w", encoding="utf-8") as f:
            json.dump(self.embeddings_cache, f, ensure_ascii=False, indent=2)

    def evaluate_metadata_match(self, passport: CompanyPassport, opp: PolicyOpportunity) -> float:
        """
        Returns a score in [0.0, 1.0] indicating geographic and basic profile alignment.
        """
        score = 1.0
        
        # 1. Geography match:
        opp_geo = opp.geography.lower()
        comp_loc = passport.location.value.lower() if hasattr(passport, "location") else ""
        
        # If policy specifies a specific region and company is not there, penalize
        if "toàn quốc" not in opp_geo:
            if comp_loc and comp_loc not in opp_geo and opp_geo not in comp_loc:
                score -= 0.5
                
        # 2. Industry match:
        opp_target = opp.target_companies.lower()
        comp_ind = passport.industry.value.lower() if hasattr(passport, "industry") else ""
        
        # Check if company industry is mentioned in target companies or rules
        if comp_ind:
            # Check simple string containment
            if comp_ind in opp_target or comp_ind in opp.title.lower():
                score += 0.2
            else:
                score -= 0.1
                
        return max(0.0, min(1.0, score))

    def retrieve(self, passport: CompanyPassport, query: str, top_n: int = 3) -> List[dict[str, Any]]:
        """
        Retrieves ranked PolicyOpportunities using BM25 and Vector Search over decree chunks.
        """
        self.load_opportunities_and_documents()
        query_tokens = query.lower().split()
        
        # 1. Compute BM25 scores over decree chunks
        chunk_bm25_scores = self.bm25.get_scores(query_tokens)
        max_bm25 = max(chunk_bm25_scores) if len(chunk_bm25_scores) > 0 and max(chunk_bm25_scores) > 0 else 1.0
        normalized_chunk_bm25 = [score / max_bm25 for score in chunk_bm25_scores]
        
        # 2. Compute Vector scores over decree chunks
        query_emb = self.get_embedding(query, is_query=True)
        chunk_vector_scores = []
        for chunk in self.chunks:
            chunk_emb = self.get_embedding(chunk["text"], is_query=False)
            sim = cosine_similarity(query_emb, chunk_emb)
            chunk_vector_scores.append(sim)
            
        # 3. Aggregate chunk scores for each PolicyOpportunity
        bm25_scores = []
        vector_scores = []
        for opp in self.opportunities:
            linked_docs = opp.source_legal_documents
            # Find chunks belonging to this opportunity's source documents
            matched_indices = [
                idx for idx, chunk in enumerate(self.chunks)
                if chunk["doc_id"] in linked_docs
            ]
            
            if matched_indices:
                # Take the max score among all matched chunks
                opp_bm25 = max([normalized_chunk_bm25[idx] for idx in matched_indices])
                opp_vector = max([chunk_vector_scores[idx] for idx in matched_indices])
            else:
                opp_bm25 = 0.0
                opp_vector = 0.0
                
            bm25_scores.append(opp_bm25)
            vector_scores.append(opp_vector)
            
        # 4. Compute Metadata match scores
        metadata_scores = [self.evaluate_metadata_match(passport, opp) for opp in self.opportunities]
        
        # 5. Fusion ranking: final_score = 0.4*bm25 + 0.4*vector + 0.2*metadata
        fused_results = []
        for i, opp in enumerate(self.opportunities):
            is_relevant = True
            if query.strip():
                opp_text = f"{opp.title} {opp.benefits} {opp.target_companies}".lower()
                opp_emb = self.get_embedding(opp_text, is_query=False)
                direct_sim = cosine_similarity(query_emb, opp_emb)
                is_substring = query.lower() in opp_text
                
                # Filter out opportunities without semantic match or direct substring matches
                if direct_sim < 0.805 and not is_substring:
                    is_relevant = False
                    
            if not is_relevant:
                continue

            final_score = (
                0.4 * bm25_scores[i] +
                0.4 * vector_scores[i] +
                0.2 * metadata_scores[i]
            )
            fused_results.append({
                "opportunity": opp,
                "score": round(final_score, 4),
                "bm25_score": round(bm25_scores[i], 4),
                "vector_score": round(vector_scores[i], 4),
                "metadata_score": round(metadata_scores[i], 4)
            })
            
        # Sort by final score descending
        fused_results.sort(key=lambda x: x["score"], reverse=True)
        return fused_results[:top_n]
