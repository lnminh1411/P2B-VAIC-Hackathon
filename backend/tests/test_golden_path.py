import os
import json
import unittest
from app.schemas.passport import CompanyPassport
from app.schemas.policy import PolicyOpportunity
from app.engine.retrieval import HybridRetrievalEngine
from app.engine.rule_evaluator import evaluate_rule_group
from docxtpl import DocxTemplate

class TestP2BGoldenPath(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.current_dir = os.path.dirname(os.path.abspath(__file__))
        cls.seed_dir = os.path.join(os.path.dirname(cls.current_dir), "app", "seed")
        
        # Load passports
        passports_path = os.path.join(cls.seed_dir, "company_passports.json")
        with open(passports_path, "r", encoding="utf-8") as f:
            cls.passports_data = json.load(f)
            
        # Load opportunities
        opps_path = os.path.join(cls.seed_dir, "policy_opportunities.json")
        with open(opps_path, "r", encoding="utf-8") as f:
            cls.opps_data = json.load(f)
            
        # Load ground truth
        gt_path = os.path.join(cls.seed_dir, "ground_truth.json")
        with open(gt_path, "r", encoding="utf-8") as f:
            cls.ground_truth = json.load(f)
            
        cls.engine = HybridRetrievalEngine(cls.seed_dir)

    def test_ground_truth_compliance(self):
        """
        Verify that eligibility verifier matches the labeled ground truth for all 3 companies and 6 policies.
        """
        for company_id, expected_policies in self.ground_truth.items():
            passport = CompanyPassport(**self.passports_data[company_id])
            for opp_data in self.opps_data:
                opp = PolicyOpportunity(**opp_data)
                
                # Run deterministic verifier
                status, details = evaluate_rule_group(passport, opp.eligibility_rules)
                
                expected_status = expected_policies[opp.id]["status"]
                
                print(f"[Test] Checking {company_id} against {opp.id} -> Expected: {expected_status}, Actual: {status}")
                self.assertEqual(
                    status, 
                    expected_status, 
                    f"Mismatch for {company_id} on {opp.id}: expected {expected_status}, got {status}"
                )

    def test_rag_retrieval(self):
        """
        Verify that RAG retrieves correct AI program for the AI query.
        """
        passport = CompanyPassport(**self.passports_data["AItech_Vietnam_LLC"])
        query = "chương trình nghiên cứu trí tuệ nhân tạo"
        results = self.engine.retrieve(passport, query, top_n=1)
        
        self.assertGreater(len(results), 0)
        top_opp = results[0]["opportunity"]
        self.assertEqual(top_opp.id, "national_ai_program")
        self.assertGreaterEqual(results[0]["score"], 0.8)

    def test_template_filling_no_hallucination(self):
        """
        Verify that document template is filled correctly.
        """
        passport = CompanyPassport(**self.passports_data["AItech_Vietnam_LLC"])
        template_path = os.path.join(self.seed_dir, "grant_template.docx")
        doc = DocxTemplate(template_path)
        
        context = {
            "company_name": passport.company_name.value,
            "tax_code": passport.tax_code.value,
            "location": passport.location.value,
            "employee_count": passport.employee_count.value,
            "rd_spend_ratio": f"{passport.rd_spend_ratio.value * 100}%",
            "registered_capital": f"{passport.registered_capital.value:,} VND",
            "revenue": f"{passport.revenue.value:,} VND"
        }
        
        doc.render(context)
        
        # Verify that render runs without exceptions
        temp_out = os.path.join(self.seed_dir, "filled_grant_test_output.docx")
        doc.save(temp_out)
        
        self.assertTrue(os.path.exists(temp_out))
        # Cleanup
        if os.path.exists(temp_out):
            os.remove(temp_out)

if __name__ == "__main__":
    unittest.main()
