import os
import json
import hashlib

def create_legal_corpus(output_dir: str):
    documents = [
        # --- SME SUPPORT GROUP ---
        {
            "id": "law_sme_support_2017",
            "title": "Luật Hỗ trợ Doanh nghiệp nhỏ và vừa năm 2017 (Luật số 04/2017/QH14)",
            "issuing_body": "Quốc hội Việt Nam",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=04/2017/QH14",
            "issued_at": "2017-06-12",
            "effective_from": "2018-01-01",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 4. Phân loại doanh nghiệp nhỏ và vừa: Doanh nghiệp nhỏ và vừa bao gồm doanh nghiệp siêu nhỏ, doanh nghiệp nhỏ và doanh nghiệp vừa, có số lao động tham gia bảo hiểm xã hội bình quân năm không quá 200 người và đáp ứng một trong hai tiêu chí sau đây: Tổng nguồn vốn không quá 100 tỷ đồng; Tổng doanh thu của năm trước liền kề không quá 300 tỷ đồng.",
                "Điều 10. Hỗ trợ về thuế, kế toán: Doanh nghiệp nhỏ và vừa được áp dụng có thời hạn mức thuế suất thuế thu nhập doanh nghiệp thấp hơn mức thuế suất thông thường áp dụng cho doanh nghiệp theo quy định của pháp luật về thuế thu nhập doanh nghiệp.",
                "Điều 16. Hỗ trợ doanh nghiệp nhỏ và vừa khởi nghiệp sáng tạo: Doanh nghiệp nhỏ và vừa khởi nghiệp sáng tạo được hỗ trợ chi phí sử dụng dịch vụ công nghệ, đào tạo chuyên sâu, sử dụng không gian làm việc chung, và tham gia các cuộc thi về khởi nghiệp."
            ]
        },
        {
            "id": "decree_80_2021_nd_cp",
            "title": "Nghị định số 80/2021/NĐ-CP hướng dẫn Luật Hỗ trợ Doanh nghiệp nhỏ và vừa",
            "issuing_body": "Chính phủ Việt Nam",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=80/2021/ND-CP",
            "issued_at": "2021-08-26",
            "effective_from": "2021-10-15",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 5. Hỗ trợ công nghệ: Hỗ trợ tối đa 50% giá trị hợp đồng tư vấn giải pháp chuyển đổi số cho doanh nghiệp nhưng không quá 50 triệu đồng/năm đối với doanh nghiệp nhỏ.",
                "Điều 11. Hỗ trợ phát triển nguồn nhân lực: Miễn học phí cho học viên của doanh nghiệp nhỏ và vừa khi tham gia các khóa đào tạo khởi sự kinh doanh và quản trị doanh nghiệp sử dụng ngân sách nhà nước."
            ]
        },
        {
            "id": "decree_39_2018_nd_cp",
            "title": "Nghị định số 39/2018/NĐ-CP chi tiết Luật Hỗ trợ Doanh nghiệp nhỏ và vừa",
            "issuing_body": "Chính phủ Việt Nam",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=39/2018/ND-CP",
            "issued_at": "2018-03-11",
            "effective_from": "2018-03-11",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 12. Hỗ trợ tư vấn pháp lý: Thiết lập mạng lưới tư vấn viên pháp luật để hỗ trợ pháp lý, giải quyết tranh chấp kinh doanh cho doanh nghiệp nhỏ và vừa.",
                "Điều 14. Hỗ trợ thông tin: Bộ Kế hoạch và Đầu tư xây dựng Cổng thông tin quốc gia hỗ trợ doanh nghiệp nhỏ và vừa để cung cấp miễn phí thông tin về cơ chế, chính sách, kế hoạch, chương trình hỗ trợ."
            ]
        },
        {
            "id": "circular_06_2022_tt_bkhdt",
            "title": "Thông tư số 06/2022/TT-BKHĐT hướng dẫn một số điều về hỗ trợ doanh nghiệp nhỏ và vừa",
            "issuing_body": "Bộ Kế hoạch và Đầu tư",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=06/2022/TT-BKHDT",
            "issued_at": "2022-05-10",
            "effective_from": "2022-06-25",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 4. Quy trình hỗ trợ: Doanh nghiệp chuẩn bị hồ sơ đề xuất nhu cầu hỗ trợ nộp trực tiếp hoặc trực tuyến qua Cổng thông tin hỗ trợ doanh nghiệp nhỏ và vừa.",
                "Điều 8. Kinh phí hỗ trợ: Cơ quan hỗ trợ SME thực hiện thanh toán chi phí hỗ trợ trực tiếp cho đơn vị cung cấp dịch vụ hoặc hoàn trả cho doanh nghiệp theo hợp đồng đã thực hiện."
            ]
        },

        # --- HIGH-TECH & R&D GROUP ---
        {
            "id": "law_high_tech_2008",
            "title": "Luật Công nghệ cao năm 2008 (Luật số 21/2008/QH12)",
            "issuing_body": "Quốc hội Việt Nam",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=21/2008/QH12",
            "issued_at": "2008-11-13",
            "effective_from": "2009-07-01",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 18. Doanh nghiệp công nghệ cao: Là doanh nghiệp sản xuất sản phẩm công nghệ cao, ứng dụng công nghệ cao có hoạt động nghiên cứu phát triển, quy mô doanh thu và nhân lực nghiên cứu đạt chuẩn theo quy định của Thủ tướng Chính phủ.",
                "Điều 19. Ưu đãi đối với doanh nghiệp công nghệ cao: Được hưởng mức ưu đãi cao nhất theo quy định của pháp luật về thuế thu nhập doanh nghiệp, thuế giá trị gia tăng, thuế xuất nhập khẩu và đất đai."
            ]
        },
        {
            "id": "decision_10_2021_qd_ttg",
            "title": "Quyết định số 10/2021/QĐ-TTg quy định tiêu chí xác định doanh nghiệp công nghệ cao",
            "issuing_body": "Thủ tướng Chính phủ",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=10/2021/QD-TTg",
            "issued_at": "2021-03-16",
            "effective_from": "2021-04-30",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 3. Tiêu chí xác định doanh nghiệp công nghệ cao: Doanh nghiệp công nghiệp phải đáp ứng các tiêu chí sau: Tỷ lệ chi cho hoạt động nghiên cứu và phát triển (R&D) của doanh nghiệp trên tổng doanh thu thuần hàng năm tối thiểu đạt 1% đối với doanh nghiệp có doanh thu từ 100 tỷ đồng trở lên.",
                "Điều 3. Số lượng lao động có trình độ chuyên môn trực tiếp thực hiện nghiên cứu và phát triển đạt tối thiểu 1% đến 2.5% tổng số lao động tùy thuộc vào quy mô doanh nghiệp."
            ]
        },
        {
            "id": "decision_38_2020_qd_ttg",
            "title": "Quyết định số 38/2020/QĐ-TTg ban hành danh mục công nghệ cao được ưu tiên đầu tư phát triển",
            "issuing_body": "Thủ tướng Chính phủ",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=38/2020/QD-TTg",
            "issued_at": "2020-12-30",
            "effective_from": "2021-02-15",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 1. Danh mục công nghệ cao: Công nghệ trí tuệ nhân tạo; Công nghệ Internet vạn vật; Công nghệ sinh học thế hệ mới; Công nghệ sản xuất linh kiện bán dẫn và vi mạch điện tử.",
                "Điều 2. Tổ chức thực hiện: Bộ Khoa học và Công nghệ chủ trì phối hợp các bộ ngành định kỳ rà soát, đề xuất sửa đổi danh mục công nghệ cao ưu tiên đầu tư phù hợp xu hướng phát triển."
            ]
        },
        {
            "id": "decree_13_2019_nd_cp",
            "title": "Nghị định số 13/2019/NĐ-CP về phát triển doanh nghiệp khoa học và công nghệ",
            "issuing_body": "Chính phủ Việt Nam",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=13/2019/ND-CP",
            "issued_at": "2019-02-01",
            "effective_from": "2019-03-20",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 12. Miễn, giảm thuế thu nhập doanh nghiệp: Doanh nghiệp khoa học và công nghệ được miễn thuế thu nhập doanh nghiệp trong 4 năm và giảm 50% số thuế phải nộp trong 9 năm tiếp theo kể từ năm đầu tiên có thu nhập chịu thuế.",
                "Điều 15. Hỗ trợ tín dụng: Các dự án sản xuất thử nghiệm sản phẩm khoa học công nghệ được hỗ trợ vay vốn ưu đãi từ Quỹ Đổi mới công nghệ quốc gia."
            ]
        },

        # --- NATIONAL INNOVATION CENTER (NIC) GROUP ---
        {
            "id": "decree_94_2020_nd_cp",
            "title": "Nghị định số 94/2020/NĐ-CP quy định cơ chế, chính sách ưu đãi đối với Trung tâm Đổi mới sáng tạo Quốc gia",
            "issuing_body": "Chính phủ Việt Nam",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=94/2020/ND-CP",
            "issued_at": "2020-08-21",
            "effective_from": "2020-10-05",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 8. Ưu đãi đối với doanh nghiệp đổi mới sáng tạo khởi nghiệp hoạt động tại NIC: Miễn tiền thuê đất, thuê mặt nước trong toàn bộ thời hạn thuê; Giảm 50% phí dịch vụ sử dụng hạ tầng kỹ thuật tại cơ sở Hòa Lạc; Áp dụng thuế suất thuế thu nhập doanh nghiệp 10% trong 15 năm.",
                "Điều 12. Hỗ trợ về thủ tục xuất nhập khẩu, hải quan: Ưu tiên áp dụng chế độ thông quan nhanh cho các thiết bị công nghệ, vật tư phục vụ trực tiếp nghiên cứu và phát triển của doanh nghiệp đặt tại NIC Hòa Lạc."
            ]
        },
        {
            "id": "decree_35_2022_nd_cp",
            "title": "Nghị định số 35/2022/NĐ-CP về quản lý khu công nghiệp và khu kinh tế",
            "issuing_body": "Chính phủ Việt Nam",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=35/2022/ND-CP",
            "issued_at": "2022-05-28",
            "effective_from": "2022-07-15",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 28. Ưu đãi đầu tư khu công nghiệp: Các dự án đầu tư phát triển hạ tầng và dự án công nghệ cao sản xuất tại khu công nghiệp được hưởng ưu đãi thuế CIT và thuế nhập khẩu theo khung pháp lý ưu đãi đặc biệt.",
                "Điều 33. Thủ tục một cửa: Ban quản lý khu công nghiệp thực hiện thẩm quyền cấp, điều chỉnh giấy chứng nhận đăng ký đầu tư đối với các dự án thứ cấp một cách nhanh chóng."
            ]
        },
        {
            "id": "decision_1269_qd_ttg",
            "title": "Quyết định số 1269/QĐ-TTg thành lập Trung tâm Đổi mới sáng tạo Quốc gia",
            "issuing_body": "Thủ tướng Chính phủ",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=1269/QD-TTg",
            "issued_at": "2019-10-02",
            "effective_from": "2019-10-02",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 1. Thành lập NIC: Thành lập Trung tâm Đổi mới sáng tạo Quốc gia trực thuộc Bộ Kế hoạch và Đầu tư nhằm thúc đẩy hoạt động đổi mới sáng tạo, chuyển giao công nghệ, hỗ trợ doanh nghiệp khởi nghiệp sáng tạo.",
                "Điều 2. Nhiệm vụ của Trung tâm: Xây dựng hệ sinh thái đổi mới sáng tạo; Tổ chức đào tạo nguồn nhân lực số; Thu hút vốn đầu tư mạo hiểm quốc tế."
            ]
        },

        # --- GREEN ENERGY & SUSTAINABILITY GROUP ---
        {
            "id": "decision_1658_qd_ttg",
            "title": "Quyết định số 1658/QĐ-TTg phê duyệt Chiến lược quốc gia về tăng trưởng xanh giai đoạn 2021-2030",
            "issuing_body": "Thủ tướng Chính phủ",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=1658/QD-TTg",
            "issued_at": "2021-10-01",
            "effective_from": "2021-10-01",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Mục II.3. Giải pháp phát triển công nghệ xanh: Hỗ trợ tài chính từ Quỹ tăng trưởng xanh cho các dự án kinh tế tuần hoàn, năng lượng tái tạo, tiết kiệm năng lượng hiệu quả cao.",
                "Mục II.3. Hỗ trợ tài trợ: Doanh nghiệp công nghệ xanh có mức vốn tự có từ 20 tỷ đồng trở lên được xem xét hỗ trợ các khoản tài trợ không hoàn lại phục vụ ứng dụng công nghệ tiết kiệm năng lượng."
            ]
        },
        {
            "id": "law_environment_2020",
            "title": "Luật Bảo vệ môi trường năm 2020 (Luật số 72/2020/QH14)",
            "issuing_body": "Quốc hội Việt Nam",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=72/2020/QH14",
            "issued_at": "2020-11-17",
            "effective_from": "2022-01-01",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 138. Phát triển kinh tế tuần hoàn: Nhà nước khuyến khích, ưu đãi cho tổ chức, cá nhân thực hiện hoạt động nghiên cứu khoa học, ứng dụng công nghệ, sản xuất sản phẩm thân thiện môi trường.",
                "Điều 141. Tín dụng xanh, trái phiếu xanh: Các dự án đầu tư bảo vệ môi trường, tiết kiệm năng lượng được ưu tiên cấp tín dụng xanh với lãi suất ưu đãi."
            ]
        },
        {
            "id": "decree_08_2022_nd_cp",
            "title": "Nghị định số 08/2022/NĐ-CP quy định chi tiết một số điều của Luật Bảo vệ môi trường",
            "issuing_body": "Chính phủ Việt Nam",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=08/2022/ND-CP",
            "issued_at": "2022-01-10",
            "effective_from": "2022-01-10",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 131. Ưu đãi về đất đai và tài chính: Các dự án xử lý chất thải tập trung, sản xuất năng lượng sạch được miễn, giảm tiền sử dụng đất và hỗ trợ vốn đầu tư hạ tầng.",
                "Điều 135. Hỗ trợ sản phẩm thân thiện môi trường: Các sản phẩm được gắn nhãn sinh thái Việt Nam được ưu tiên mua sắm công bằng ngân sách nhà nước."
            ]
        },
        {
            "id": "decision_500_qd_ttg",
            "title": "Quyết định số 500/QĐ-TTg phê duyệt Quy hoạch phát triển điện lực quốc gia (Quy hoạch điện VIII)",
            "issuing_body": "Thủ tướng Chính phủ",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=500/QD-TTg",
            "issued_at": "2023-05-15",
            "effective_from": "2023-05-15",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Mục III. Định hướng phát triển: Ưu tiên phát triển mạnh mẽ các nguồn năng lượng tái tạo (điện gió, điện mặt trời, điện sinh khối) phục vụ sản xuất và tiêu dùng nội địa.",
                "Mục IV. Giải pháp thực hiện: Khuyến khích mô hình điện mặt trời mái nhà tự sản tự tiêu phục vụ cho các doanh nghiệp sản xuất công nghiệp và dân dụng."
            ]
        },

        # --- SEMICONDUCTOR & FOREIGN INVESTMENT GROUP ---
        {
            "id": "investment_law_2020",
            "title": "Luật Đầu tư năm 2020 (Luật số 61/2020/QH14)",
            "issuing_body": "Quốc hội Việt Nam",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=61/2020/QH14",
            "issued_at": "2020-06-17",
            "effective_from": "2021-01-01",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 19. Ngành, nghề ưu đãi đầu tư: Công nghệ thông tin, sản xuất sản phẩm phần mềm, linh kiện điện tử, bán dẫn, sản phẩm công nghệ số trọng điểm là ngành nghề đặc biệt ưu đãi đầu tư.",
                "Điều 20. Ưu đãi đầu tư đặc biệt: Áp dụng thuế suất CIT ưu đãi đặc biệt tối thiểu 5% trong thời hạn tối đa 30 năm đối với các dự án thành lập mới trung tâm nghiên cứu và phát triển có tổng vốn đầu tư từ 3.000 tỷ đồng trở lên."
            ]
        },
        {
            "id": "decree_31_2021_nd_cp",
            "title": "Nghị định số 31/2021/NĐ-CP hướng dẫn chi tiết thi hành Luật Đầu tư",
            "issuing_body": "Chính phủ Việt Nam",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=31/2021/ND-CP",
            "issued_at": "2021-03-26",
            "effective_from": "2021-03-26",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 22. Danh mục ngành nghề ưu đãi: Quy định chi tiết các lĩnh vực công nghệ cao, linh kiện điện tử, vi mạch bán dẫn và sản phẩm công nghệ hỗ trợ được áp dụng mức ưu đãi đầu tư.",
                "Điều 30. Quy trình đăng ký ưu đãi: Doanh nghiệp tự xác định điều kiện ưu đãi đầu tư và thực hiện thủ tục tại cơ quan thuế, hải quan không cần xin xác nhận riêng."
            ]
        },
        {
            "id": "decision_1018_qd_ttg",
            "title": "Quyết định số 1018/QĐ-TTg ban hành Chiến lược phát triển công nghiệp bán dẫn Việt Nam đến năm 2030",
            "issuing_body": "Thủ tướng Chính phủ",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=1018/QD-TTg",
            "issued_at": "2024-09-21",
            "effective_from": "2024-09-21",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Mục II. Mục tiêu đến năm 2030: Thu hút các tập đoàn bán dẫn lớn xây dựng nhà máy đóng gói, kiểm thử và sản xuất chip bán dẫn tại Việt Nam; Đào tạo tối thiểu 50.000 kỹ sư bán dẫn.",
                "Mục III. Giải pháp đột phá: Hỗ trợ tài chính đặc biệt cho dự án R&D thiết kế chip và chế tạo linh kiện bán dẫn từ Quỹ phát triển khoa học công nghệ quốc gia."
            ]
        },
        {
            "id": "decision_29_2021_qd_ttg",
            "title": "Quyết định số 29/2021/QĐ-TTg quy định về ưu đãi đầu tư đặc biệt",
            "issuing_body": "Thủ tướng Chính phủ",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=29/2021/QD-TTg",
            "issued_at": "2021-10-06",
            "effective_from": "2021-10-06",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 5. Mức ưu đãi thuế suất: Thuế suất thuế TNDN ưu đãi 9% trong 30 năm, 7% trong 33 năm hoặc 5% trong 37 năm áp dụng tùy thuộc mức độ đóng góp khoa học công nghệ của dự án đầu tư đặc biệt.",
                "Điều 6. Miễn giảm tiền thuê đất: Miễn tiền thuê đất, thuê mặt nước lên tới 18-22 năm đối với dự án nghiên cứu và sản xuất công nghệ cao quy mô lớn."
            ]
        },

        # --- ARTIFICIAL INTELLIGENCE (AI) GROUP ---
        {
            "id": "decision_127_qd_ttg",
            "title": "Quyết định số 127/QĐ-TTg về Chiến lược quốc gia về nghiên cứu, phát triển và ứng dụng Trí tuệ nhân tạo đến năm 2030",
            "issuing_body": "Thủ tướng Chính phủ",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=127/QD-TTg",
            "issued_at": "2021-01-26",
            "effective_from": "2021-01-26",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Mục III.1. Nghiên cứu và phát triển sản phẩm AI trọng điểm: Tài trợ từ 50% đến 100% kinh phí cho doanh nghiệp chủ trì các đề tài khoa học công nghệ cấp quốc gia trong lĩnh vực trí tuệ nhân tạo với chi phí R&D tối thiểu đạt 5% doanh thu.",
                "Mục III.2. Phát triển nguồn nhân lực AI: Hỗ trợ kết nối chuyên gia AI quốc tế, đào tạo chuyên sâu và chuyển giao công nghệ cho doanh nghiệp khởi nghiệp trong nước."
            ]
        },
        {
            "id": "decision_749_qd_ttg",
            "title": "Quyết định số 749/QĐ-TTg phê duyệt Chương trình Chuyển đổi số quốc gia đến năm 2025",
            "issuing_body": "Thủ tướng Chính phủ",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=749/QD-TTg",
            "issued_at": "2020-06-03",
            "effective_from": "2020-06-03",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Mục II. Mục tiêu cơ bản: Phát triển chính phủ số, kinh tế số và xã hội số, phổ cập danh tính điện tử và thúc đẩy thanh toán không dùng tiền mặt.",
                "Mục III. Nhiệm vụ giải pháp: Đẩy mạnh phát triển hạ tầng băng rộng cáp quang, mạng di động 5G, phát triển các nền tảng điện toán đám mây Make in Vietnam."
            ]
        },
        {
            "id": "decree_13_2023_nd_cp",
            "title": "Nghị định số 13/2023/NĐ-CP về bảo vệ dữ liệu cá nhân",
            "issuing_body": "Chính phủ Việt Nam",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=13/2023/ND-CP",
            "issued_at": "2023-04-17",
            "effective_from": "2023-07-01",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Điều 9. Quyền của chủ thể dữ liệu: Chủ thể dữ liệu có quyền đồng ý, rút lại sự đồng ý, yêu cầu xóa dữ liệu và được thông báo về các hoạt động xử lý dữ liệu cá nhân.",
                "Điều 24. Đánh giá tác động xử lý dữ liệu cá nhân: Doanh nghiệp xử lý dữ liệu cá nhân phải lập và lưu giữ Hồ sơ đánh giá tác động xử lý dữ liệu cá nhân gửi về Bộ Công an."
            ]
        },
        {
            "id": "decision_2289_qd_ttg",
            "title": "Quyết định số 2289/QĐ-TTg ban hành Chiến lược quốc gia về Cách mạng công nghiệp lần thứ tư đến năm 2030",
            "issuing_body": "Thủ tướng Chính phủ",
            "source_url": "https://vbpl.vn/pages/portal.aspx?SearchTerm=2289/QD-TTg",
            "issued_at": "2020-12-31",
            "effective_from": "2020-12-31",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Mục III. Định hướng chiến lược: Hoàn thiện thể chế thúc đẩy đổi mới sáng tạo, phát triển hạ tầng số kết nối vạn vật, đầu tư mạnh mẽ vào các công nghệ tương lai như AI, Big Data, Blockchain.",
                "Mục IV. Tổ chức thực hiện: Giao Bộ Thông tin và Truyền thông chủ trì xây dựng kế hoạch hành động về hạ tầng mạng, an toàn thông tin phục vụ CMCN 4.0."
            ]
        }
    ]

    # Compute content hashes
    for doc in documents:
        content_str = " ".join(doc["chunks"])
        doc["content_hash"] = hashlib.sha256(content_str.encode("utf-8")).hexdigest()

    output_path = os.path.join(output_dir, "legal_corpus.json")
    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(documents, f, ensure_ascii=False, indent=2)
    print(f"Generated legal corpus JSON with {len(documents)} documents at: {output_path}")

if __name__ == "__main__":
    current_dir = os.path.dirname(os.path.abspath(__file__))
    create_legal_corpus(current_dir)
