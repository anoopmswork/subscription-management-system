# Domain 06: Payments

## 1) Objective and Business Outcomes
Collect funds reliably across payment providers with secure tokenized methods, resilient retries, and ledger-grade reconciliation.

## 2) Scope
### In Scope
- Payment method vault references (tokenized)
- Payment intent creation and capture
- Multi-provider routing and failover policy
- Webhook processing and reconciliation
- Refunds and payment reversals

### Out of Scope
- Dunning strategy orchestration (Dunning domain)
- Invoice generation logic (Billing domain)

## 3) Actors and Permissions
- **Customer Billing Admin**: add/update payment methods.
- **Payment Service**: execute payment attempts.
- **Finance Ops**: run reconciliations, issue refunds.

## 4) Data Model
- `payment_method`: id, customer_id, provider, provider_token, type, brand, last4, expiry_month, expiry_year, is_default.
- `payment_intent`: id, invoice_id, customer_id, amount_minor, currency, status, attempt_count.
- `payment_attempt`: id, intent_id, provider, request_id, status, failure_code, processed_at.
- `payment_transaction`: id, attempt_id, provider_txn_id, auth_amount_minor, captured_amount_minor, settled_at.
- `refund`: id, payment_transaction_id, amount_minor, reason, status.
- `reconciliation_record`: id, provider, statement_date, expected_amount_minor, settled_amount_minor, variance_minor.

## 5) API Surface (REST + Events)
### REST
- `POST /v1/customers/{customer_id}/payment-methods`
- `PATCH /v1/payment-methods/{payment_method_id}/default`
- `POST /v1/payments/intents`
- `POST /v1/payments/intents/{intent_id}/confirm`
- `POST /v1/payments/{transaction_id}/refund`
- `POST /v1/payments/webhooks/{provider}`

### Events
- `payment.method.added`
- `payment.intent.created`
- `payment.attempt.failed`
- `payment.succeeded`
- `payment.refund.succeeded`

## 6) Core Workflows and State Transitions
1. **Payment Method Add**: collect method via provider SDK -> store provider token + metadata only.
2. **Charge Invoice**: create intent -> confirm with provider -> update invoice/payment state.
3. **Auth/Capture**: for supported methods, authorize first then capture on finalization.
4. **Webhook Reconciliation**: process asynchronous status updates idempotently.

State model:
- Intent: `created -> requires_action -> processing -> succeeded|failed|canceled`

## 7) Validation Rules and Edge Cases
- Enforce currency compatibility per provider.
- Prevent duplicate webhook processing with event-id dedupe.
- Handle partial captures and partial refunds.
- Support strong customer authentication flows (`requires_action`).
- Retry only on retryable failure codes; stop on hard declines.

## 8) Security and Compliance
- PCI scope reduction: never store PAN/CVV, token references only.
- Signed webhook verification per provider.
- Secret rotation and key-scoped service accounts.
- Sensitive fields redacted in logs and traces.

## 9) Reporting and Observability
- Metrics: authorization rate, capture success rate, failure-code distribution, refund ratio.
- Reconciliation dashboard: provider settlement variances.
- Alerts: webhook verification failures, spike in hard declines.

## 10) Acceptance Criteria and Test Scenarios
### Acceptance Criteria
- Payment attempts are idempotent and safely retryable.
- Webhook events update local state exactly once.
- Refund workflow supports full and partial cases.

### Test Scenarios
1. Successful card payment updates invoice to `paid`.
2. Soft decline triggers retryable failure path.
3. Duplicate webhook event does not duplicate transaction updates.
4. Partial refund updates payment and invoice balance accurately.

## 11) Domain Dependencies and Sequencing
- Depends on Customer and Subscription/Billing signals.
- Feeds Dunning, Notifications, Admin Analytics, and Integrations.
