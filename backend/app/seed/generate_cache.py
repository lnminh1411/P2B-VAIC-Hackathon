import os
import json
from sentence_transformers import SentenceTransformer

def generate():
    print("Loading local SentenceTransformer (intfloat/multilingual-e5-base)...")
    model = SentenceTransformer('intfloat/multilingual-e5-base')

    # Define texts and queries (prefixed as required by e5)
    data_to_embed = [
        # Queries (prefixed with 'query: ')
        ("query: AItech Vietnam LLC", "AItech Vietnam LLC", True),
        ("query: FDI SemiVina Corp", "FDI SemiVina Corp", True),
        ("query: SolarGreen Tech JSC", "SolarGreen Tech JSC", True),
        ("query: chính sách ưu đãi AI", "chính sách ưu đãi AI", True),
        ("query: incentives for semiconductor", "incentives for semiconductor", True),
        ("query: green innovation grants", "green innovation grants", True),
        ("query: chính sách hỗ trợ doanh nghiệp nhỏ và vừa", "chính sách hỗ trợ doanh nghiệp nhỏ và vừa", True),
        ("query: ưu đãi thuế công nghệ cao hte", "ưu đãi thuế công nghệ cao hte", True),
        ("query: ưu đãi nic hòa lạc", "ưu đãi nic hòa lạc", True),
        ("query: hỗ trợ R&D bán dẫn fdi", "hỗ trợ R&D bán dẫn fdi", True),
        ("query: quỹ phát triển công nghệ xanh", "quỹ phát triển công nghệ xanh", True),
        
        # Policies (prefixed with 'passage: ')
        ("passage: Chương trình Giảm Thuế Thu Nhập Doanh Nghiệp Cho SME Giảm thuế suất thuế thu nhập doanh nghiệp (CIT) xuống 15-17% (so với mức phổ thông 20%). Doanh nghiệp nhỏ và vừa tại Việt Nam", "Chương trình Giảm Thuế Thu Nhập Doanh Nghiệp Cho SME Giảm thuế suất thuế thu nhập doanh nghiệp (CIT) xuống 15-17% (so với mức phổ thông 20%). Doanh nghiệp nhỏ và vừa tại Việt Nam", False),
        ("passage: Ưu Đãi Thuế Cho Doanh Nghiệp Công Nghệ Cao (HTE) Miễn thuế CIT trong 4 năm đầu, giảm 50% trong 9 năm tiếp theo, thuế suất ưu đãi 10% trong 15 năm. Doanh nghiệp sản xuất sản phẩm công nghệ cao hoặc ứng dụng công nghệ cao", "Ưu Đãi Thuế Cho Doanh Nghiệp Công Nghệ Cao (HTE) Miễn thuế CIT trong 4 năm đầu, giảm 50% trong 9 năm tiếp theo, thuế suất ưu đãi 10% trong 15 năm. Doanh nghiệp sản xuất sản phẩm công nghệ cao hoặc ứng dụng công nghệ cao", False),
        ("passage: Chương Trình Ưu Đãi Đặc Biệt Cho Thành Viên NIC Hòa Lạc Miễn tiền thuê đất toàn bộ thời hạn thuê, hỗ trợ thủ tục hải quan nhanh, ưu đãi thuế nhập khẩu thiết bị R&D. Doanh nghiệp công nghệ đổi mới sáng tạo đặt trụ sở tại NIC Hòa Lạc", "Chương Trình Ưu Đãi Đặc Biệt Cho Thành Viên NIC Hòa Lạc Miễn tiền thuê đất toàn bộ thời hạn thuê, hỗ trợ thủ tục hải quan nhanh, ưu đãi thuế nhập khẩu thiết bị R&D. Doanh nghiệp công nghệ đổi mới sáng tạo đặt trụ sở tại NIC Hòa Lạc", False),
        ("passage: Quỹ Tài Trợ Phát Triển Công Nghệ Xanh và Năng Lượng Tái Tạo Tài trợ vốn không hoàn lại lên đến 1 tỷ VND cho dự án nghiên cứu và thương mại hóa sản phẩm xanh. Doanh nghiệp công nghệ xanh, tiết kiệm năng lượng tại Việt Nam", "Quỹ Tài Trợ Phát Triển Công Nghệ Xanh và Năng Lượng Tái Tạo Tài trợ vốn không hoàn lại lên đến 1 tỷ VND cho dự án nghiên cứu và thương mại hóa sản phẩm xanh. Doanh nghiệp công nghệ xanh, tiết kiệm năng lượng tại Việt Nam", False),
        ("passage: Chính Sách Hỗ Trợ Đặc Biệt Dự Án FDI Bán Dẫn và R&D Thuế suất ưu đãi đặc biệt 5% CIT trong 30 năm, miễn thuế nhập khẩu tài sản cố định tạo dự án R&D. Tập đoàn công nghệ nước ngoài đầu tư nhà máy bán dẫn lớn", "Chính Sách Hỗ Trợ Đặc Biệt Dự Án FDI Bán Dẫn và R&D Thuế suất ưu đãi đặc biệt 5% CIT trong 30 năm, miễn thuế nhập khẩu tài sản cố định tạo dự án R&D. Tập đoàn công nghệ nước ngoài đầu tư nhà máy bán dẫn lớn", False),
        ("passage: Chương Trình Khoa Học Công Nghệ Quốc Gia Về Trí Tuệ Nhân Tạo Hỗ trợ 100% kinh phí đề tài nghiên cứu AI, ưu tiên kết nối đối tác công nghệ quốc tế. Doanh nghiệp nghiên cứu và phát triển giải pháp AI tại Việt Nam", "Chương Trình Khoa Học Công Nghệ Quốc Gia Về Trí Tuệ Nhân Tạo Hỗ trợ 100% kinh phí đề tài nghiên cứu AI, ưu tiên kết nối đối tác công nghệ quốc tế. Doanh nghiệp nghiên cứu và phát triển giải pháp AI tại Việt Nam", False)
    ]

    cache = {}
    print(f"Generating embeddings for {len(data_to_embed)} items...")
    for idx, (prefixed_text, raw_text, is_query) in enumerate(data_to_embed):
        print(f"Embedding item {idx + 1}/{len(data_to_embed)} ({'query' if is_query else 'passage'})")
        emb = model.encode(prefixed_text, normalize_embeddings=True)
        cache[prefixed_text] = emb.tolist()

    # Write cache
    output_path = os.path.join(os.path.dirname(__file__), "cached_embeddings.json")
    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(cache, f, ensure_ascii=False, indent=2)
    print(f"Saved cached embeddings to: {output_path}")

if __name__ == "__main__":
    generate()
