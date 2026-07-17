import os
import io
import uuid
import unittest
from fastapi.testclient import TestClient
from app.main import app
from app.engine.db import get_db_connection

class TestIntegrationP2B(unittest.TestCase):
    def setUp(self):
        self.client = TestClient(app)
        # Verify db is initialized (already seeded in previous steps)
        
    def test_cors_headers(self):
        """Verify CORS whitelist enforcement"""
        # Test whitelisted origin
        response = self.client.options(
            "/api/v1/policies",
            headers={
                "Origin": "http://localhost:5173",
                "Access-Control-Request-Method": "GET"
            }
        )
        self.assertEqual(response.headers.get("access-control-allow-origin"), "http://localhost:5173")
        
        # Test unwhitelisted origin
        response = self.client.options(
            "/api/v1/policies",
            headers={
                "Origin": "https://hackersite.com",
                "Access-Control-Request-Method": "GET"
            }
        )
        self.assertNotEqual(response.headers.get("access-control-allow-origin"), "https://hackersite.com")

    def test_auth_full_lifecycle(self):
        """Test signup, login, get profile, change password, and logout"""
        email = f"test_{uuid.uuid4().hex[:6]}@p2b.vn"
        password = "SecurePassword123"
        
        # 1. Signup
        signup_res = self.client.post("/api/v1/auth/signup", json={
            "email": email,
            "password": password,
            "user_type": "COMPANY_MANAGER"
        })
        self.assertEqual(signup_res.status_code, 200)
        
        # 2. Login
        login_res = self.client.post("/api/v1/auth/login", json={
            "email": email,
            "password": password
        })
        self.assertEqual(login_res.status_code, 200)
        token = login_res.json()["token"]
        auth_header = {"Authorization": f"Bearer {token}"}
        
        # 3. Access profile
        me_res = self.client.get("/api/v1/users/me", headers=auth_header)
        self.assertEqual(me_res.status_code, 200)
        self.assertEqual(me_res.json()["user"]["email"], email)
        self.assertEqual(me_res.json()["user"]["user_type"], "COMPANY_MANAGER")
        
        # 4. Change Password
        change_res = self.client.put("/api/v1/users/change-password", headers=auth_header, json={
            "old_password": password,
            "new_password": "NewSecurePassword456"
        })
        self.assertEqual(change_res.status_code, 200)
        
        # 5. Verify old password fails, new password succeeds
        login_fail = self.client.post("/api/v1/auth/login", json={
            "email": email,
            "password": password
        })
        self.assertEqual(login_fail.status_code, 400)
        
        login_ok = self.client.post("/api/v1/auth/login", json={
            "email": email,
            "password": "NewSecurePassword456"
        })
        self.assertEqual(login_ok.status_code, 200)
        new_token = login_ok.json()["token"]
        new_auth = {"Authorization": f"Bearer {new_token}"}
        
        # 6. Logout
        logout_res = self.client.post("/api/v1/auth/logout", headers=new_auth)
        self.assertEqual(logout_res.status_code, 200)
        
        # 7. Check session invalidated
        me_fail = self.client.get("/api/v1/users/me", headers=new_auth)
        self.assertEqual(me_fail.status_code, 401)

    def test_avatar_upload_and_delete(self):
        """Test avatar uploading and account deletion"""
        email = f"test_{uuid.uuid4().hex[:6]}@p2b.vn"
        password = "Password123"
        
        # Signup & Login
        self.client.post("/api/v1/auth/signup", json={"email": email, "password": password, "user_type": "INDIVIDUAL"})
        tok = self.client.post("/api/v1/auth/login", json={"email": email, "password": password}).json()["token"]
        auth = {"Authorization": f"Bearer {tok}"}
        
        # Upload mock image
        file_data = {"file": ("avatar.png", io.BytesIO(b"fakepngdata"), "image/png")}
        upload_res = self.client.post("/api/v1/users/avatar", headers={"Authorization": f"Bearer {tok}"}, files=file_data)
        self.assertEqual(upload_res.status_code, 200)
        self.assertIn("avatar_url", upload_res.json())
        
        # Verify me endpoint returns avatar path
        me_res = self.client.get("/api/v1/users/me", headers=auth)
        self.assertTrue(me_res.json()["user"]["avatar_path"].startswith("/static/avatars/"))
        
        # Delete user account
        del_res = self.client.delete("/api/v1/users", headers=auth)
        self.assertEqual(del_res.status_code, 200)
        
        # Confirm user deletion
        login_fail = self.client.post("/api/v1/auth/login", json={"email": email, "password": password})
        self.assertEqual(login_fail.status_code, 400)

    def test_gated_reviews_enforcement(self):
        """Verify gated review blocks un-MET approvals and permits MET ones"""
        # Login pre-seeded aitech user
        login_res = self.client.post("/api/v1/auth/login", json={
            "email": "aitech@p2b.vn",
            "password": "Password123"
        })
        self.assertEqual(login_res.status_code, 200)
        auth = {"Authorization": f"Bearer {login_res.json()['token']}"}
        
        # 1. Create a draft for national_ai_program (MET status)
        draft_res_met = self.client.post("/api/v1/drafts", headers=auth, json={"policy_id": "national_ai_program"})
        self.assertEqual(draft_res_met.status_code, 200)
        draft_id_met = draft_res_met.json()["draft_id"]
        
        # Approve MET draft - should pass gating
        app_res_met = self.client.put(f"/api/v1/drafts/{draft_id_met}/status", headers=auth, json={
            "status": "APPROVED",
            "reviewer_comments": "Looks perfect!"
        })
        self.assertEqual(app_res_met.status_code, 200)
        self.assertEqual(app_res_met.json()["status"], "GENERATED")
        
        # Download document should succeed
        dl_res = self.client.get(f"/api/v1/drafts/{draft_id_met}/download", headers=auth)
        self.assertEqual(dl_res.status_code, 200)
        
        # 2. Create draft for green_innovation_grant (NOT_MET status for AItech)
        draft_res_fail = self.client.post("/api/v1/drafts", headers=auth, json={"policy_id": "green_innovation_grant"})
        self.assertEqual(draft_res_fail.status_code, 200)
        draft_id_fail = draft_res_fail.json()["draft_id"]
        
        # Approve NOT_MET draft - should fail gating with 400
        app_res_fail = self.client.put(f"/api/v1/drafts/{draft_id_fail}/status", headers=auth, json={
            "status": "APPROVED",
            "reviewer_comments": "Should fail!"
        })
        self.assertEqual(app_res_fail.status_code, 400)
        self.assertIn("Cannot approve draft", app_res_fail.json()["detail"])

    def test_golden_path_consecutive_runs(self):
        """Run full golden path of registration, policy search, eligibility check, gated approval 5 times consecutively"""
        for i in range(5):
            email = f"golden_{i}_{uuid.uuid4().hex[:4]}@p2b.vn"
            password = "Password123"
            
            # 1. Signup
            self.client.post("/api/v1/auth/signup", json={"email": email, "password": password, "user_type": "COMPANY_MANAGER"})
            # 2. Login
            tok = self.client.post("/api/v1/auth/login", json={"email": email, "password": password}).json()["token"]
            auth = {"Authorization": f"Bearer {tok}"}
            
            # Fetch company ID
            company_id = self.client.get("/api/v1/users/me", headers=auth).json()["user"]["company_id"]
            
            # 3. Update company passport values so it becomes eligible (MET) for National AI Program
            # Set location to NIC Hoa Lac, employee_count > 10, etc.
            passport_res = self.client.get(f"/api/v1/passports/{company_id}", headers=auth)
            passport_data = passport_res.json()["data"]
            
            passport_data["company_name"]["value"] = f"Golden Co {i}"
            passport_data["company_name"]["status"] = "USER_CONFIRMED"
            passport_data["location"]["value"] = "NIC Hoa Lac"
            passport_data["location"]["status"] = "USER_CONFIRMED"
            passport_data["employee_count"]["value"] = 50
            passport_data["employee_count"]["status"] = "USER_CONFIRMED"
            passport_data["rd_spend_ratio"]["value"] = 0.05
            passport_data["rd_spend_ratio"]["status"] = "USER_CONFIRMED"
            passport_data["revenue"]["value"] = 15000000000
            passport_data["revenue"]["status"] = "USER_CONFIRMED"
            passport_data["registered_capital"]["value"] = 5000000000
            passport_data["registered_capital"]["status"] = "USER_CONFIRMED"
            passport_data["industry"]["value"] = "Artificial Intelligence"
            passport_data["industry"]["status"] = "USER_CONFIRMED"
            
            # Save updated passport
            self.client.put(f"/api/v1/passports/{company_id}", headers=auth, json={"passport_data": passport_data})
            
            # 4. Search policies
            policies = self.client.get("/api/v1/policies", headers=auth).json()
            self.assertGreater(len(policies), 0)
            
            # 5. Check eligibility for national_ai_program
            el_res = self.client.post("/api/v1/eligibility", headers=auth, json={"policy_id": "national_ai_program"})
            self.assertEqual(el_res.json()["status"], "MET")
            
            # 6. Create Draft
            draft_res = self.client.post("/api/v1/drafts", headers=auth, json={"policy_id": "national_ai_program"})
            draft_id = draft_res.json()["draft_id"]
            
            # 7. Approve Draft (Gated approval passes since MET)
            app_res = self.client.put(f"/api/v1/drafts/{draft_id}/status", headers=auth, json={
                "status": "APPROVED",
                "reviewer_comments": f"Golden path iteration {i}"
            })
            self.assertEqual(app_res.status_code, 200)
            self.assertEqual(app_res.json()["status"], "GENERATED")
            
            # 8. Download document
            dl_res = self.client.get(f"/api/v1/drafts/{draft_id}/download", headers=auth)
            self.assertEqual(dl_res.status_code, 200)
            
            # 9. Cleanup - delete user
            self.client.delete("/api/v1/users", headers=auth)

if __name__ == "__main__":
    unittest.main()
