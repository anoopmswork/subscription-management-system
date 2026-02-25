# Domain 07: Taxation

## 1) Objective and Business Outcomes
Calculate and apply compliant indirect taxes (VAT/GST/Sales Tax) for subscription and usage charges based on jurisdiction, product taxability, and customer profile.

## 2) Scope
### In Scope
- Tax jurisdiction determination by address and nexus rules
- Product/plan tax category mapping
- Exemption handling and reverse-charge logic
- Tax rate lookup and tax line computation
- Tax transaction audit records

### Out of Scope
- Filing/remittance to authorities (external finance process)
- Invoice document rendering (Billing domain)

## 3) Actors and Permissions
- **Tax Admin**: configure nexus, categories, exemptions.
- **Billing Engine**: request tax quote/commit.
- **Finance Auditor**: review tax decisions and audit trails.

## 4) Data Model
- `tax_nexus`: id, legal_entity_id, country, region, effective_from, effective_to.
- `tax_category`: id, code, description, product_types.
- `tax_exemption_certificate`: id, customer_id, jurisdiction, certificate_ref, valid_from, valid_to, status.
- `tax_quote`: id, customer_id, invoice_preview_id, subtotal_minor, tax_total_minor, currency, status.
- `tax_line`: id, tax_quote_id, line_ref, jurisdiction, rate_bps, taxable_amount_minor, tax_amount_minor.
- `tax_transaction`: id, invoice_id, external_ref, committed_at, voided_at.

## 5) API Surface (REST + Events)
### REST
- `POST /v1/tax/quote`
- `POST /v1/tax/commit`
- `POST /v1/tax/void`
- `POST /v1/tax/exemptions`
- `GET /v1/tax/quotes/{tax_quote_id}`

### Events
- `tax.quote.created`
- `tax.quote.failed`
- `tax.transaction.committed`
- `tax.exemption.updated`

## 6) Core Workflows and State Transitions
1. **Tax Quote**: resolve ship/bill address -> map tax categories -> compute per line tax.
2. **Commit Tax Transaction**: on invoice finalization, persist committed tax transaction reference.
3. **Void Tax Transaction**: when invoice is voided/credited according to legal rules.
4. **Exemption Evaluation**: apply valid certificates and reverse-charge where jurisdiction requires.

State model:
- Quote: `draft -> calculated -> committed|expired|failed`

## 7) Validation Rules and Edge Cases
- Require complete taxable address fields before quote.
- Handle mixed-taxability invoices (taxable and exempt lines).
- Recalculate tax on address or line-item changes before finalization.
- Currency rounding rules per jurisdiction (line-level vs invoice-level rounding).
- Backdated invoice finalization uses historical rates at service period date.

## 8) Security and Compliance
- Encrypt tax identifiers and exemption documents.
- Immutable tax decision snapshots for audit.
- Region-specific data retention policies.

## 9) Reporting and Observability
- Metrics: quote latency, quote failure rate, exemption usage rate.
- Audit reports: tax collected by jurisdiction/entity/period.
- Alerts: missing nexus mapping, repeated external tax engine failures.

## 10) Acceptance Criteria and Test Scenarios
### Acceptance Criteria
- Tax calculations are reproducible from stored snapshot inputs.
- Exemptions are honored only within validity windows.
- Tax commit/void operations are idempotent.

### Test Scenarios
1. Taxable customer in VAT region -> quote includes VAT per line.
2. Valid exemption certificate -> expected zero tax lines.
3. Address change before finalization -> quote recalculated.
4. Duplicate tax commit request -> single committed transaction.

## 11) Domain Dependencies and Sequencing
- Depends on Customer (address/tax profile), Plan/Pricing, Discounts, and Usage totals.
- Feeds Billing finalization and Admin financial reporting.
