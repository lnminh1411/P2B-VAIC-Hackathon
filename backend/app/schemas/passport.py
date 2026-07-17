from typing import Any, Optional
from pydantic import BaseModel, Field

class FieldProvenance(BaseModel):
    value: Any
    source_type: str = Field(..., description="E.g., PITCH_DECK, BUSINESS_REGISTRATION, WEBSITE, MANUAL_INPUT")
    source_uri: str = Field(..., description="File name or URL source")
    source_location: str = Field(..., description="E.g., page 8, section 3, homepage")
    evidence_quote: str = Field(..., description="Verbatim text quote from the source")
    observed_at: str = Field(..., description="ISO timestamp of extraction/observation")
    confidence: str = Field(..., description="HIGH, MEDIUM, or LOW")
    status: str = Field(..., description="EXTRACTED, USER_PROVIDED, USER_CONFIRMED, CONFLICTED, MISSING")
    conflicts: list[Any] = Field(default_factory=list, description="List of conflicting evidence records")

class CompanyPassport(BaseModel):
    company_name: FieldProvenance
    tax_code: FieldProvenance
    industry: FieldProvenance
    location: FieldProvenance
    employee_count: FieldProvenance
    rd_spend_ratio: FieldProvenance
    revenue: FieldProvenance
    registered_capital: FieldProvenance
    metadata: Optional[dict[str, Any]] = Field(default_factory=dict, description="Dynamic extra attributes")
