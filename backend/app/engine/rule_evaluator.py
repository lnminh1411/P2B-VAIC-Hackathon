from typing import Any, Tuple, Optional
from datetime import datetime
from app.schemas.passport import CompanyPassport, FieldProvenance
from app.schemas.policy import Rule, RuleGroup, RuleOperator, GroupLogic

def parse_date(date_str: Any) -> Optional[datetime]:
    if not date_str:
        return None
    for fmt in ("%Y-%m-%d", "%Y-%m-%dT%H:%M:%S", "%Y-%m-%dT%H:%M:%S%z", "%Y-%m-%d %H:%M:%S"):
        try:
            return datetime.strptime(str(date_str).strip(), fmt)
        except ValueError:
            continue
    return None

def evaluate_single_rule(passport: CompanyPassport, rule: Rule) -> Tuple[str, dict[str, Any]]:
    """
    Evaluates a single deterministic rule against the CompanyPassport.
    Returns (status, result_details) where status is MET, NOT_MET, or MISSING_INFO.
    """
    field_name = rule.field
    # Check if field exists in CompanyPassport
    if not hasattr(passport, field_name):
        # Check in metadata
        metadata = passport.metadata or {}
        if field_name in metadata:
            # Metadata fields might not be full FieldProvenance objects, wrap them
            val = metadata[field_name]
            field_provenance = FieldProvenance(
                value=val,
                source_type="METADATA",
                source_uri="database_metadata",
                source_location="metadata",
                evidence_quote=f"Metadata field {field_name}",
                observed_at=datetime.utcnow().isoformat(),
                confidence="HIGH",
                status="USER_PROVIDED",
                conflicts=[]
            )
        else:
            return "MISSING_INFO", {
                "rule_id": rule.criterion_id,
                "description": rule.description,
                "status": "MISSING_INFO",
                "reason": f"Trường thông tin '{field_name}' không tồn tại trong Hồ sơ doanh nghiệp.",
                "clarification_question": f"Vui lòng cung cấp thông tin về: {rule.description.lower()}."
            }
    else:
        field_provenance = getattr(passport, field_name)

    # Handlers for field statuses
    if field_provenance.status == "MISSING" or field_provenance.value is None:
        return "MISSING_INFO", {
            "rule_id": rule.criterion_id,
            "description": rule.description,
            "status": "MISSING_INFO",
            "reason": f"Trường thông tin '{field_name}' bị thiếu trong Hồ sơ doanh nghiệp.",
            "clarification_question": f"Vui lòng cung cấp thông tin về: {rule.description.lower()}."
        }

    if field_provenance.status == "CONFLICTED":
        return "MISSING_INFO", {
            "rule_id": rule.criterion_id,
            "description": rule.description,
            "status": "MISSING_INFO",
            "reason": f"Phát hiện mâu thuẫn dữ liệu đối với trường '{field_name}'.",
            "clarification_question": f"Vui lòng xác nhận thông tin chính xác về: {rule.description.lower()}."
        }

    actual_val = field_provenance.value
    expected_val = rule.expected_value
    op = rule.operator

    met = False
    try:
        if op == RuleOperator.EQ:
            met = (actual_val == expected_val)
        elif op == RuleOperator.NEQ:
            met = (actual_val != expected_val)
        elif op == RuleOperator.IN:
            if isinstance(expected_val, list):
                met = (actual_val in expected_val)
            else:
                met = (actual_val == expected_val)
        elif op == RuleOperator.CONTAINS:
            if isinstance(actual_val, list):
                met = (expected_val in actual_val)
            elif isinstance(actual_val, str):
                met = (str(expected_val) in actual_val)
            else:
                met = False
        elif op == RuleOperator.GTE:
            met = (float(actual_val) >= float(expected_val))
        elif op == RuleOperator.LTE:
            met = (float(actual_val) <= float(expected_val))
        elif op == RuleOperator.DATE_BEFORE:
            actual_date = parse_date(actual_val)
            expected_date = parse_date(expected_val)
            if actual_date and expected_date:
                met = (actual_date < expected_date)
            else:
                return "MISSING_INFO", {
                    "rule_id": rule.criterion_id,
                    "description": rule.description,
                    "status": "MISSING_INFO",
                    "reason": f"Không thể so sánh ngày. Định dạng ngày không hợp lệ. Thực tế: {actual_val}, Kì vọng: {expected_val}."
                }
        elif op == RuleOperator.DATE_AFTER:
            actual_date = parse_date(actual_val)
            expected_date = parse_date(expected_val)
            if actual_date and expected_date:
                met = (actual_date > expected_date)
            else:
                return "MISSING_INFO", {
                    "rule_id": rule.criterion_id,
                    "description": rule.description,
                    "status": "MISSING_INFO",
                    "reason": f"Không thể so sánh ngày. Định dạng ngày không hợp lệ. Thực tế: {actual_val}, Kì vọng: {expected_val}."
                }
    except Exception as e:
        return "MISSING_INFO", {
            "rule_id": rule.criterion_id,
            "description": rule.description,
            "status": "MISSING_INFO",
            "reason": f"Lỗi so sánh kiểu dữ liệu: {str(e)}."
        }

    status = "MET" if met else "NOT_MET"
    reason = (
        f"Đạt điều kiện: Giá trị thực tế '{actual_val}' thỏa mãn '{op.value}' so với kì vọng '{expected_val}'."
        if met else
        f"Không đạt điều kiện: Giá trị thực tế '{actual_val}' không thỏa mãn '{op.value}' so với kì vọng '{expected_val}'."
    )

    return status, {
        "rule_id": rule.criterion_id,
        "description": rule.description,
        "status": status,
        "reason": reason,
        "field": field_name,
        "actual_value": actual_val,
        "expected_value": expected_val,
        "operator": op.value,
        "evidence_quote": field_provenance.evidence_quote,
        "source_uri": field_provenance.source_uri,
        "source_location": field_provenance.source_location,
        "citation": rule.citation.model_dump() if rule.citation else None
    }

def evaluate_rule_group(passport: CompanyPassport, group: RuleGroup) -> Tuple[str, dict[str, Any]]:
    """
    Recursively evaluates a RuleGroup using ALL/ANY logic.
    Returns (status, group_details).
    """
    sub_results = []
    statuses = set()

    for rule in group.rules:
        if isinstance(rule, RuleGroup):
            sub_status, sub_detail = evaluate_rule_group(passport, rule)
            sub_results.append(sub_detail)
            statuses.add(sub_status)
        else:
            sub_status, sub_detail = evaluate_single_rule(passport, rule)
            sub_results.append(sub_detail)
            statuses.add(sub_status)

    logic = group.logic

    if logic == GroupLogic.ALL:
        # ALL logic:
        # If any is NOT_MET, the group is NOT_MET
        if "NOT_MET" in statuses:
            final_status = "NOT_MET"
        # If no NOT_MET, but there is MISSING_INFO, the group is MISSING_INFO (cannot confirm MET yet)
        elif "MISSING_INFO" in statuses:
            final_status = "MISSING_INFO"
        # Otherwise, all are MET
        else:
            final_status = "MET"
    else:
        # ANY logic:
        # If any is MET, the group is MET
        if "MET" in statuses:
            final_status = "MET"
        # If no MET, but there is MISSING_INFO, the group is MISSING_INFO
        elif "MISSING_INFO" in statuses:
            final_status = "MISSING_INFO"
        # Otherwise, all are NOT_MET
        else:
            final_status = "NOT_MET"

    return final_status, {
        "criterion_group_id": group.criterion_group_id,
        "logic": logic.value,
        "status": final_status,
        "rules": sub_results
    }
