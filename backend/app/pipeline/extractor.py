import os
import re
import json
import numpy as np
from typing import List, Dict, Any, Tuple
from pydantic import BaseModel, Field
from google import genai
from google.genai import types
from sentence_transformers import SentenceTransformer

# Load local E5 model for the document sorting agent
EMBED_MODEL_NAME = "intfloat/multilingual-e5-base"
_embed_model = None

def get_embed_model():
    global _embed_model
    if _embed_model is None:
        print(f"Loading embeddings model for document sorting: {EMBED_MODEL_NAME}")
        # Run offline-first, trust that model is cached
        _embed_model = SentenceTransformer(EMBED_MODEL_NAME)
    return _embed_model

class ExtractedPassport(BaseModel):
    company_name: str = Field(description="Tên chính thức của doanh nghiệp")
    tax_code: str = Field(description="Mã số thuế hoặc mã số doanh nghiệp")
    industry: str = Field(description="Lĩnh vực hoạt động chính (ví dụ: Semiconductor, Artificial Intelligence, Green Energy / Innovation)")
    location: str = Field(description="Địa chỉ trụ sở chính (ví dụ: NIC Hoa Lac, Da Nang High-Tech Park, Ho Chi Minh City)")
    employee_count: int = Field(description="Tổng số lượng lao động/nhân sự")
    rd_spend_ratio: float = Field(description="Tỷ lệ chi phí R&D trên tổng doanh thu (giá trị thực từ 0.0 đến 1.0, ví dụ: 0.03)")
    revenue: int = Field(description="Doanh thu hàng năm tính bằng VND")
    registered_capital: int = Field(description="Vốn điều lệ đăng ký tính bằng VND")
    evidence_quotes: Dict[str, str] = Field(description="Trích dẫn nguyên văn từ tài liệu chứng minh cho từng trường thông tin")
    page_locations: Dict[str, str] = Field(description="Trang hoặc mục chứa thông tin trích dẫn")

class ExtractedPersonal(BaseModel):
    full_name: str = Field(description="Họ và tên cá nhân")
    birth_year: int = Field(description="Năm sinh")
    location: str = Field(description="Tỉnh/Thành phố sinh sống")
    occupation: str = Field(description="Nghề nghiệp hoặc chuyên môn chính (ví dụ: Semiconductor Engineer, AI Researcher)")
    degree: str = Field(description="Bằng cấp cao nhất (ví dụ: Bachelor, Master, PhD)")
    monthly_income: int = Field(description="Thu nhập trung bình hàng tháng tính bằng VND")
    evidence_quotes: Dict[str, str] = Field(description="Trích dẫn nguyên văn hỗ trợ cho các trường")
    page_locations: Dict[str, str] = Field(description="Trang hoặc mục chứa thông tin")

def extract_year_from_text(text: str) -> int:
    """Finds the most recent year (e.g. 2024, 2025) mentioned in the document text."""
    years = re.findall(r'\b(202\d|201\d)\b', text)
    if not years:
        return 0
    return max(int(y) for y in years)

def rank_documents_for_field(field_query: str, documents: List[Dict[str, Any]]) -> Dict[str, Any]:
    """
    Ranks documents based on semantic relevance to the query and date recency,
    returning the top document.
    Each document dict should contain: 'name', 'text' (markdown/plain text), 'path'.
    """
    if not documents:
        raise ValueError("No documents uploaded to sort.")
        
    model = get_embed_model()
    # Embed the search query
    query_vector = model.encode(f"query: {field_query}", normalize_embeddings=True)
    
    scored_docs = []
    for doc in documents:
        text = doc['text']
        # Extract recency
        year = extract_year_from_text(text)
        year_score = (year - 2018) / 8.0 if year > 2018 else 0.0  # Normalized weight
        
        # Embed first 2000 chars of document for relevance check
        sample_text = text[:2000]
        doc_vector = model.encode(f"passage: {sample_text}", normalize_embeddings=True)
        
        relevance_score = float(np.dot(query_vector, doc_vector))
        
        # Combined score: 60% relevance, 40% recency
        combined_score = 0.6 * relevance_score + 0.4 * year_score
        
        scored_docs.append({
            "doc": doc,
            "relevance": relevance_score,
            "year": year,
            "score": combined_score
        })
        
    # Sort descending
    scored_docs.sort(key=lambda x: x["score"], reverse=True)
    print(f"[Agent Sort] Ranking for query '{field_query}':")
    for s in scored_docs[:3]:
        print(f"  - {s['doc']['name']} (Year: {s['year']}, Relevance: {s['relevance']:.3f}, Score: {s['score']:.3f})")
        
    return scored_docs[0]["doc"]

def call_gemini_extraction(file_path: str, mime_type: str, user_type: str = "COMPANY_MANAGER") -> Dict[str, Any]:
    """
    Calls the Gemini API (gemini-3.1-flash-lite) to perform structured extraction on a file.
    Works with both PDF bytes and converted Markdown text.
    """
    api_key = os.environ.get("GEMINI_API_KEY")
    if not api_key:
        raise ValueError("GEMINI_API_KEY environment variable is missing. Live extraction requires an API key.")
        
    client = genai.Client(api_key=api_key)
    
    # 1. Prepare contents
    _, ext = os.path.splitext(file_path.lower())
    if ext == '.pdf':
        with open(file_path, 'rb') as f:
            file_bytes = f.read()
        parts = [
            types.Part.from_bytes(data=file_bytes, mime_type=mime_type),
            "Vui lòng trích xuất thông tin chi tiết từ tài liệu đính kèm này."
        ]
    else:
        # Non-PDF documents converted to Markdown text
        with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
            text_content = f.read()
        parts = [
            f"Nội dung tài liệu:\n\n{text_content}\n\nVui lòng trích xuất thông tin chi tiết từ tài liệu trên."
        ]
        
    schema = ExtractedPassport if user_type == "COMPANY_MANAGER" else ExtractedPersonal
    prompt = (
        "Trích xuất tất cả các trường thông tin theo đúng định dạng schema yêu cầu. "
        "Yêu cầu cung cấp trích dẫn nguyên văn (evidence_quotes) và trang/mục tương ứng (page_locations). "
        "Nếu tài liệu thiếu hoặc không đề cập trường nào, hãy điền giá trị mặc định (rỗng hoặc 0) nhưng phải trung thực."
    )
    
    parts.append(prompt)
    
    # 2. Call Gemini
    model_name = os.environ.get("GEMINI_EXTRACT_MODEL", "gemini-3.1-flash-lite")
    print(f"Calling Gemini model {model_name} for user type {user_type}...")
    response = client.models.generate_content(
        model=model_name,
        contents=parts,
        config=types.GenerateContentConfig(
            response_mime_type="application/json",
            response_schema=schema,
            temperature=0.1
        )
    )
    
    return json.loads(response.text)
