from enum import Enum
from typing import Any, Optional, Union
from pydantic import BaseModel, Field

class RuleOperator(str, Enum):
    EQ = "EQ"
    NEQ = "NEQ"
    IN = "IN"
    CONTAINS = "CONTAINS"
    GTE = "GTE"
    LTE = "LTE"
    DATE_BEFORE = "DATE_BEFORE"
    DATE_AFTER = "DATE_AFTER"

class GroupLogic(str, Enum):
    ALL = "ALL"
    ANY = "ANY"

class DocumentStatus(str, Enum):
    CURRENT = "CURRENT"
    EXPIRED = "EXPIRED"
    SUPERSEDED = "SUPERSEDED"

class Citation(BaseModel):
    document_id: str
    article: str
    page: Optional[int] = None
    quote: str
    source_url: str

class Rule(BaseModel):
    criterion_id: str
    description: str
    field: str
    operator: RuleOperator
    expected_value: Any
    required: bool
    citation: Citation

class RuleGroup(BaseModel):
    criterion_group_id: str
    logic: GroupLogic
    rules: list[Union[Rule, "RuleGroup"]]

# Rebuild model for recursive Union definition in Pydantic v2
RuleGroup.model_rebuild()

class PolicyOpportunity(BaseModel):
    id: str
    title: str
    benefits: str
    target_companies: str = Field(..., description="Target company type narrative")
    geography: str
    deadline: Optional[str] = None
    required_documents: list[str] = Field(default_factory=list)
    eligibility_rules: RuleGroup
    source_legal_documents: list[str] = Field(default_factory=list)

class LegalDocument(BaseModel):
    id: str
    title: str
    issuing_body: str
    source_url: str
    issued_at: str
    effective_from: str
    effective_to: Optional[str] = None
    last_verified_at: str
    status: DocumentStatus
    content_hash: str
    chunks: list[str] = Field(default_factory=list)
