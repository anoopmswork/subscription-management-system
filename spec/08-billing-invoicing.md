# Domain 08: Billing & Invoicing

## 1) Objective and Business Outcomes
Generate accurate, compliant invoices and credit notes from subscription, usage, discount, and tax inputs while maintaining immutable financial records.

## 2) Scope
### In Scope
- Billing cycle orchestration and invoice generation
- Invoice previews and finalization
- Line item construction (recurring, usage, adjustments)
- Credit notes and void flows
- Accounts receivable status tracking

### Out of Scope
- Payment execution (Payments domain)
- Dunning communication strategy (Dunning/Notifications domains)

## 3) Actors and Permissions
- **Billing Engine**: scheduled invoice generation/finalization.
- **Finance Admin**: issue credit notes/voids with controls.
- **Customer Billing User**: view/download invoices.

## 4) Data Model
- `invoice`: id, customer_id, subscription_id, number, status, currency, subtotal_minor, discount_minor, tax_minor, total_minor, due_date.
- `invoice_line_item`: id, invoice_id, type, description, quantity, unit_amount_minor, amount_minor, service_period_start, service_period_end.
- `invoice_adjustment`: id, invoice_id, source(`proration`,`manual`,`usage_correction`), amount_minor, reason.
- `credit_note`: id, invoice_id, number, amount_minor, reason, status.
- `billing_run`: id, period_start, period_end, status, started_at, completed_at, item_count.
- `ar_balance`: customer_id, open_amount_minor, overdue_amount_minor, currency, updated_at.

## 5) API Surface (REST + Events)
### REST
- `POST /v1/billing/runs`
- `POST /v1/invoices/preview`
- `POST /v1/invoices/{invoice_id}/finalize`
- `GET /v1/invoices/{invoice_id}`
- `POST /v1/invoices/{invoice_id}/void`
- `POST /v1/invoices/{invoice_id}/credit-notes`

### Events
- `billing.run.started`
- `billing.invoice.created`
- `billing.invoice.finalized`
- `billing.invoice.voided`
- `billing.credit_note.created`

## 6) Core Workflows and State Transitions
1. **Invoice Preview**: assemble recurring + usage + discounts + tax quote for customer confirmation.
2. **Invoice Finalization**: lock line items/taxes/totals, assign legal invoice number, emit event.
3. **Adjustments**: post-finalization changes handled via credit/debit notes, not line mutation.
4. **AR Updates**: update customer outstanding and aging buckets.

State model:
- Invoice: `draft -> open -> paid|partially_paid|overdue|void`

## 7) Validation Rules and Edge Cases
- No line-item mutation after finalization.
- Enforce unique invoice numbering per legal entity/series.
- Support minimum invoice amount thresholds (defer to next cycle).
- Handle zero-total invoice policy (issue or suppress per configuration).
- Split invoices per tax/legal entity where required.

## 8) Security and Compliance
- Immutable finalized invoice and credit-note snapshots.
- Access controls for document download and finance actions.
- Audit logs for void/credit operations with approval trail.

## 9) Reporting and Observability
- Metrics: invoice generation success rate, finalize latency, overdue ratio.
- Reports: AR aging, billed vs collected, credit-note trends.
- Alerts: billing run failures, invoice numbering gaps.

## 10) Acceptance Criteria and Test Scenarios
### Acceptance Criteria
- Invoice totals equal deterministic sum of components.
- Finalized invoice is immutable and legally numbered.
- Credit notes correctly reduce AR and preserve audit chain.

### Test Scenarios
1. Generate monthly invoice with recurring + usage + discount + tax.
2. Finalize invoice and verify immutable lock.
3. Create partial credit note and verify AR delta.
4. Retry failed billing run item idempotently without duplicates.

## 11) Domain Dependencies and Sequencing
- Depends on Subscription Lifecycle, Usage & Metering, Discounts, and Taxation.
- Feeds Payments, Dunning, Notifications, Analytics, and Integrations.
