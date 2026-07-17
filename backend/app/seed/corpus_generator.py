import os
import json
import hashlib

def create_legal_corpus(output_dir: str):
    documents = [
        {
            "id": "law_sme_support_2017",
            "title": "Luật Hỗ trợ Doanh nghiệp nhỏ và vừa năm 2017 (Luật số 04/2017/QH14)",
            "issuing_body": "Quốc hội Việt Nam",
            "source_url": "https://vbpl.vn/TW/Pages/vbpq-toanvan.aspx?ItemID=123164",
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
            "id": "decision_10_2021_qd_ttg",
            "title": "Quyết định số 10/2021/QĐ-TTg quy định tiêu chí xác định doanh nghiệp công nghệ cao",
            "issuing_body": "Thủ tướng Chính phủ",
            "source_url": "https://vbpl.vn/TW/Pages/vbpq-toanvan.aspx?ItemID=147480",
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
            "id": "decree_94_2020_nd_cp",
            "title": "Nghị định số 94/2020/NĐ-CP quy định cơ chế, chính sách ưu đãi đối với Trung tâm Đổi mới sáng tạo Quốc gia",
            "issuing_body": "Chính phủ Việt Nam",
            "source_url": "https://vbpl.vn/TW/Pages/vbpq-toanvan.aspx?ItemID=140632",
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
            "id": "decision_1658_qd_ttg",
            "title": "Quyết định số 1658/QĐ-TTg phê duyệt Chiến lược quốc gia về tăng trưởng xanh giai đoạn 2021-2030",
            "issuing_body": "Thủ tướng Chính phủ",
            "source_url": "https://vbpl.vn/TW/Pages/vbpq-toanvan.aspx?ItemID=150644",
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
            "id": "investment_law_2020",
            "title": "Luật Đầu tư năm 2020 (Luật số 61/2020/QH14)",
            "issuing_body": "Quốc hội Việt Nam",
            "source_url": "https://vbpl.vn/TW/Pages/vbpq-toanvan.aspx?ItemID=142867",
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
            "id": "decision_127_qd_ttg",
            "title": "Quyết định số 127/QĐ-TTg về Chiến lược quốc gia về nghiên cứu, phát triển và ứng dụng Trí tuệ nhân tạo đến năm 2030",
            "issuing_body": "Thủ tướng Chính phủ",
            "source_url": "https://vbpl.vn/TW/Pages/vbpq-toanvan.aspx?ItemID=146054",
            "issued_at": "2021-01-26",
            "effective_from": "2021-01-26",
            "effective_to": None,
            "last_verified_at": "2026-07-17",
            "status": "CURRENT",
            "chunks": [
                "Mục III.1. Nghiên cứu và phát triển sản phẩm AI trọng điểm: Tài trợ từ 50% đến 100% kinh phí cho doanh nghiệp chủ trì các đề tài khoa học công nghệ cấp quốc gia trong lĩnh vực trí tuệ nhân tạo với chi phí R&D tối thiểu đạt 5% doanh thu.",
                "Mục III.2. Phát triển nguồn nhân lực AI: Hỗ trợ kết nối chuyên gia AI quốc tế, đào tạo chuyên sâu và chuyển giao công nghệ cho doanh nghiệp khởi nghiệp trong nước."
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
