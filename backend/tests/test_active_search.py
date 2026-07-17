import unittest
import json
import os
import sqlite3
from app.engine.db import get_db_connection, init_db
from app.pipeline.search_crawler import search_and_cache_decrees
from app.engine.retrieval import HybridRetrievalEngine
from app.schemas.passport import CompanyPassport

class TestActiveSearchAndCache(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        init_db()

    def test_search_and_cache_inserts_to_db(self):
        # 1. Run search crawler for "trí tuệ nhân tạo"
        search_and_cache_decrees("trí tuệ nhân tạo")
        
        # 2. Verify entries exist in DB
        conn = get_db_connection()
        cursor = conn.cursor()
        
        # Check policy_opportunities
        cursor.execute("SELECT data_json FROM policy_opportunities WHERE id = 'opp_qd_188_2025_qd_ttg'")
        opp_row = cursor.fetchone()
        self.assertIsNotNone(opp_row)
        opp_data = json.loads(opp_row["data_json"])
        self.assertEqual(opp_data["id"], "opp_qd_188_2025_qd_ttg")
        
        # Check legal_documents
        cursor.execute("SELECT chunks_json FROM legal_documents WHERE id = 'qd_188_2025_qd_ttg'")
        doc_row = cursor.fetchone()
        self.assertIsNotNone(doc_row)
        chunks = json.loads(doc_row["chunks_json"])
        self.assertTrue(len(chunks) > 0)
        
        conn.close()

    def test_retrieval_engine_loads_dynamic_cache(self):
        # 1. Run crawler to seed "xanh"
        search_and_cache_decrees("xanh")
        
        # 2. Run retrieval engine
        seed_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "seed")
        engine = HybridRetrievalEngine(seed_dir)
        
        # Verify the dynamically crawled opportunity is in opportunities list
        found = any(opp.id == "opp_qd_210_2026_qd_ttg" for opp in engine.opportunities)
        self.assertTrue(found, "Should load crawled opportunity from SQLite")
        
        # Verify chunks contain the document
        chunks_found = any(chunk["doc_id"] == "qd_210_2026_qd_ttg" for chunk in engine.chunks)
        self.assertTrue(chunks_found, "Should load crawled document chunks from SQLite")

if __name__ == "__main__":
    unittest.main()
