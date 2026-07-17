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
        self.cache_path = os.path.join(seed_dir, "cached_embeddings.json")
        
        # Load opportunities
        with open(self.opportunities_path, "r", encoding="utf-8") as f:
            self.opps_data = json.load(f)
            self.opportunities = [PolicyOpportunity(**opp) for opp in self.opps_data]
            
        # Load cached embeddings if available (fallback)
        self.embeddings_cache = {}
        if os.path.exists(self.cache_path):
            with open(self.cache_path, "r", encoding="utf-8") as f:
                self.embeddings_cache = json.load(f)
                
        # Initialize local SentenceTransformer for multilingual-e5-base (now cached locally)
        print("Loading local SentenceTransformer (intfloat/multilingual-e5-base)...")
        try:
            self.model = SentenceTransformer('intfloat/multilingual-e5-base')
        except Exception as e:
            print(f"[Warning] Failed to load local SentenceTransformer: {e}. Falling back to cached embeddings only.")
            self.model = None
            
        # Initialize BM25 lexical index over opportunities (title + benefits + target_companies)
        self.corpus = []
        for opp in self.opportunities:
            text = f"{opp.title} {opp.benefits} {opp.target_companies} {opp.geography}"
            # Tokenize by simple splitting (vietnamese words can be split by space for basic BM25)
            self.corpus.append(text.lower().split())
            
        self.bm25 = BM25Okapi(self.corpus)

    def get_embedding(self, text: str, is_query: bool = False) -> List[float]:
        """
        Gets text embedding using local SentenceTransformer. 
        Prepends 'query: ' or 'passage: ' as required by multilingual-e5 models.
        """
        cleaned_text = text.strip()
        prefix = "query: " if is_query else "passage: "
        prefixed_text = prefix + cleaned_text
        
        # Check cache first
        if prefixed_text in self.embeddings_cache:
            return self.embeddings_cache[prefixed_text]
            
        if self.model:
            try:
                # E5 models are trained to output normalized embeddings
                emb = self.model.encode(prefixed_text, normalize_embeddings=True)
                embedding = emb.tolist()
                # Cache it in memory
                self.embeddings_cache[prefixed_text] = embedding
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
        Retrieves ranked PolicyOpportunities using BM25, Vector Search, and Metadata Filtering.
        """
        query_tokens = query.lower().split()
        
        # 1. Compute BM25 scores
        bm25_scores = self.bm25.get_scores(query_tokens)
        max_bm25 = max(bm25_scores) if len(bm25_scores) > 0 and max(bm25_scores) > 0 else 1.0
        normalized_bm25 = [score / max_bm25 for score in bm25_scores]
        
        # 2. Compute Vector scores
        query_emb = self.get_embedding(query, is_query=True)
        vector_scores = []
        for opp in self.opportunities:
            opp_text = f"{opp.title} {opp.benefits} {opp.target_companies}"
            opp_emb = self.get_embedding(opp_text, is_query=False)
            sim = cosine_similarity(query_emb, opp_emb)
            vector_scores.append(sim)
            
        # 3. Compute Metadata match scores
        metadata_scores = [self.evaluate_metadata_match(passport, opp) for opp in self.opportunities]
        
        # 4. Fusion ranking: final_score = 0.4*bm25 + 0.4*vector + 0.2*metadata
        fused_results = []
        for i, opp in enumerate(self.opportunities):
            final_score = (
                0.4 * normalized_bm25[i] +
                0.4 * vector_scores[i] +
                0.2 * metadata_scores[i]
            )
            fused_results.append({
                "opportunity": opp,
                "score": round(final_score, 4),
                "bm25_score": round(normalized_bm25[i], 4),
                "vector_score": round(vector_scores[i], 4),
                "metadata_score": round(metadata_scores[i], 4)
            })
            
        # Sort by final score descending
        fused_results.sort(key=lambda x: x["score"], reverse=True)
        return fused_results[:top_n]
