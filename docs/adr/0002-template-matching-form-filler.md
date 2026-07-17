# Template Matching for P2B Form Filler

## Context and Decision

Automating arbitrary government web portals with Playwright at runtime is highly fragile and prone to breaking during live hackathon demos due to CAPTCHAs, dynamic DOM changes, and portal latency. For P2B, we decided to target business-specified `.docx` application templates and a structured mock web form for the demo. The agent parses placeholders in the templates, evaluates the Company Passport against eligibility rules, and generates filled registration forms alongside a checklist of missing documents, which is more reliable and matches the P2B core value proposition.
