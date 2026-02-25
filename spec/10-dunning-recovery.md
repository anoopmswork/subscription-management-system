# Domain 10: Dunning & Recovery

## 1) Objective and Business Outcomes
Recover failed subscription payments with configurable retry strategies, customer communication, and service entitlement actions to reduce involuntary churn.

## 2) Scope
### In Scope
- Dunning policy definition by segment/region/payment method
- Retry schedules and fallback payment method logic
- Grace periods and entitlement actions (restrict/suspend/cancel)
- Promise-to-pay and manual intervention hooks
- Recovery analytics

### Out of Scope
- Payment authorization internals (Payments domain)
- Generic notifications infra (Notifications domain)

## 3) Actors and Permissions
- **Revenue Ops**: configure dunning policies.
- **Dunning Service**: execute retries and transitions.
- **Support Agent**: pause/restart dunning, capture customer commitments.

## 4) Data Model
- `dunning_policy`: id, name, segment, retry_schedule_json, max_attempts, grace_days, terminal_action.
- `dunning_case`: id, customer_id, subscription_id, invoice_id, policy_id, status, started_at, closed_at.
- `dunning_attempt`: id, case_id, attempt_no, scheduled_at, executed_at, outcome, failure_code.
- `entitlement_action`: id, case_id, action(`none`,`limit`,`suspend`,`cancel`), executed_at, reversed_at.
- `promise_to_pay`: id, case_id, promised_date, promised_amount_minor, status.

## 5) API Surface (REST + Events)
### REST
- `POST /v1/dunning/policies`
- `POST /v1/dunning/cases/{case_id}/retry-now`
- `POST /v1/dunning/cases/{case_id}/pause`
- `POST /v1/dunning/cases/{case_id}/resume`
- `POST /v1/dunning/cases/{case_id}/close`

### Events
- `dunning.case.opened`
- `dunning.attempt.scheduled`
- `dunning.attempt.failed`
- `dunning.recovered`
- `dunning.terminal_action.executed`

## 6) Core Workflows and State Transitions
1. **Case Open**: triggered on invoice payment failure beyond immediate retry threshold.
2. **Retry Ladder**: execute attempts on schedule with smart windows and fallback methods.
3. **Customer Communication**: send reminders before each critical action.
4. **Terminal Action**: apply entitlement restriction/suspension/cancel after max attempts/grace expiry.
5. **Recovery Close**: successful payment closes case and restores entitlements.

State model:
- Case: `open -> retrying -> grace -> recovered|write_off|canceled`

## 7) Validation Rules and Edge Cases
- Prevent duplicate dunning case for same invoice unless previous case closed.
- Pause retries during active payment disputes.
- Respect regional regulations on retry cadence and notice periods.
- Ensure entitlement reversal if customer recovers during grace.

## 8) Security and Compliance
- RBAC for manual overrides and write-off actions.
- Full audit trail for policy changes and case overrides.
- No exposure of sensitive payment failure details to unauthorized users.

## 9) Reporting and Observability
- Metrics: recovery rate, days-to-recover, involuntary churn, retry success by attempt index.
- Cohort reports: recovery by segment/payment method/country.
- Alerts: sudden decline in recovery rate, retry scheduler failures.

## 10) Acceptance Criteria and Test Scenarios
### Acceptance Criteria
- Failed invoices automatically enter policy-driven dunning flow.
- Retry schedule executes exactly once per attempt.
- Entitlement actions occur only when policy thresholds are met.

### Test Scenarios
1. Payment fails -> dunning case opens and first retry schedules.
2. Recovery on second retry -> case closes and entitlements stay active.
3. Exhaust retries + grace -> subscription suspended/canceled per policy.
4. Manual pause -> no retries until resumed.

## 11) Domain Dependencies and Sequencing
- Depends on Billing/AR states, Payments outcomes, and Notifications.
- Feeds Subscription status actions and revenue analytics.
