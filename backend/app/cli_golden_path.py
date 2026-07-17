import os
import json
from app.schemas.passport import CompanyPassport
from app.schemas.policy import PolicyOpportunity
from app.engine.retrieval import HybridRetrievalEngine
from app.engine.rule_evaluator import evaluate_rule_group
from docxtpl import DocxTemplate

def run_golden_path():
    print("==================================================")
    print("         P2B CLI GOLDEN PATH VERTICAL SLICE       ")
    print("==================================================")
    
    current_dir = os.path.dirname(os.path.abspath(__file__))
    seed_dir = os.path.join(current_dir, "seed")
    
    # 1. Load Primary Company Passport (AItech_Vietnam_LLC)
    passports_path = os.path.join(seed_dir, "company_passports.json")
    with open(passports_path, "r", encoding="utf-8") as f:
        passports_data = json.load(f)
    
    company_id = "AItech_Vietnam_LLC"
    passport = CompanyPassport(**passports_data[company_id])
    
    print(f"\n[Step 1] Loaded Company Passport: {company_id}")
    print(f"  - Company Name: {passport.company_name.value} (Source: {passport.company_name.source_type})")
    print(f"  - Location: {passport.location.value} (Source: {passport.location.source_type})")
    print(f"  - Industry: {passport.industry.value} (Source: {passport.industry.source_type})")
    print(f"  - Employees: {passport.employee_count.value} (Source: {passport.employee_count.source_type})")
    print(f"  - R&D spend ratio: {passport.rd_spend_ratio.value} (Source: {passport.rd_spend_ratio.source_type})")
    print(f"  - Registered capital: {passport.registered_capital.value} (Source: {passport.registered_capital.source_type})")
    
    # 2. Hybrid Retrieval for Policies matching the query
    engine = HybridRetrievalEngine(seed_dir)
    query = "chương trình nghiên cứu trí tuệ nhân tạo"
    print(f"\n[Step 2] Executing Hybrid RAG Search for query: '{query}'...")
    results = engine.retrieve(passport, query, top_n=3)
    
    for idx, res in enumerate(results):
        opp = res["opportunity"]
        print(f"  {idx + 1}. Title: {opp.title}")
        print(f"     Score: {res['score']} (BM25: {res['bm25_score']}, Vector: {res['vector_score']}, Meta: {res['metadata_score']})")
        print(f"     Benefits: {opp.benefits[:60]}...")
        
    # Pick the top result
    top_res = results[0]
    matched_opp = top_res["opportunity"]
    print(f"\nSelected top matching Policy Opportunity: '{matched_opp.title}' (ID: {matched_opp.id})")
    
    # 3. Deterministic Eligibility Verification
    print(f"\n[Step 3] Running Deterministic Rule Evaluator against criteria...")
    status, details = evaluate_rule_group(passport, matched_opp.eligibility_rules)
    print(f"  - Overall Eligibility Status: {status}")
    
    # Print criteria details
    for r in details["rules"]:
        print(f"    * Criterion ID: {r['rule_id']}")
        print(f"      Description: {r['description']}")
        print(f"      Status: {r['status']}")
        print(f"      Reason: {r['reason']}")
        if "evidence_quote" in r:
            print(f"      Company Evidence: \"{r['evidence_quote']}\"")
        if "citation" in r and r["citation"]:
            cit = r["citation"]
            print(f"      Policy Citation: {cit['document_id']} {cit['article']} (Quote: \"{cit['quote']}\")")
            
    # 4. Compare with Ground Truth
    gt_path = os.path.join(seed_dir, "ground_truth.json")
    with open(gt_path, "r", encoding="utf-8") as f:
        ground_truth = json.load(f)
        
    expected_status = ground_truth[company_id][matched_opp.id]["status"]
    print(f"\n[Verification] Verifying eligibility result against Ground Truth...")
    print(f"  - Expected Status: {expected_status}")
    print(f"  - Actual Status: {status}")
    
    if expected_status == status:
        print("  => SUCCESS: Eligibility verification matches ground truth exactly!")
    else:
        print("  => FAILURE: Mismatch in eligibility status!")
        return False
        
    # 5. Populate Template
    print(f"\n[Step 4] Drafting Application using template...")
    template_path = os.path.join(seed_dir, "grant_template.docx")
    doc = DocxTemplate(template_path)
    
    # Fill context using values from the passport
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
    output_path = os.path.join(seed_dir, "filled_grant_demo.docx")
    doc.save(output_path)
    print(f"  - Successfully exported filled .docx draft to: {output_path}")
    print("\nGolden Path Vertical Slice Completed Successfully.")
    return True

if __name__ == "__main__":
    run_golden_path()
