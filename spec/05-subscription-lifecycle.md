# Domain 05: Subscription Lifecycle

## 1) Objective and Business Outcomes
Manage subscription state transitions and commercial changes (upgrades, downgrades, pauses, cancellations, renewals) with accurate proration and policy enforcement.

## 2) Scope
### In Scope
- Subscription creation from plan versions
- Trial management and activation
- Mid-cycle plan/seat changes with proration rules
- Pause/resume and scheduled cancellation
- Renewal and term commitment handling

### Out of Scope
- Payment collection execution (Payments domain)
- Tax calculation and invoice issuance (Taxation/Billing domains)

## 3) Actors and Permissions
- **Customer Admin**: create/change/cancel own subscriptions.
- **Support Agent**: perform assisted changes with reason.
- **Billing System**: executes scheduled lifecycle jobs.

## 4) Data Model
- `subscription`: id, customer_id, status, start_at, current_period_start, current_period_end, billing_anchor, currency.
- `subscription_item`: id, subscription_id, plan_version_id, quantity, unit_price_snapshot.
- `subscription_schedule`: id, subscription_id, action_type, execute_at, payload, status.
- `subscription_change`: id, subscription_id, type(`upgrade`,`downgrade`,`cancel`,`pause`,`resume`), effective_at, proration_mode.
- `subscription_term`: id, subscription_id, term_months, auto_renew, committed_until.
- `proration_preview`: id, subscription_id, change_request_hash, delta_amount_minor, generated_at.

## 5) API Surface (REST + Events)
### REST
- `POST /v1/subscriptions`
- `GET /v1/subscriptions/{subscription_id}`
- `POST /v1/subscriptions/{subscription_id}/preview-change`
- `POST /v1/subscriptions/{subscription_id}/change`
- `POST /v1/subscriptions/{subscription_id}/pause`
- `POST /v1/subscriptions/{subscription_id}/resume`
- `POST /v1/subscriptions/{subscription_id}/cancel`

### Events
- `subscription.created`
- `subscription.activated`
- `subscription.changed`
- `subscription.paused`
- `subscription.canceled`
- `subscription.renewed`

## 6) Core Workflows and State Transitions
1. **Create Subscription**: validate customer + plan eligibility -> set billing anchor -> generate first invoice preview.
2. **Trial to Active**: auto-transition at trial end if payment method valid (or per policy).
3. **Mid-cycle Change**: calculate proration credits/debits -> apply immediately or schedule for next renewal.
4. **Cancellation**: immediate, end-of-period, or end-of-term options.

State model:
- `pending_activation -> trialing -> active -> past_due -> paused -> canceled -> expired`

## 7) Validation Rules and Edge Cases
- Prevent multiple active base plans in one subscription unless multi-plan mode enabled.
- Enforce commitment lock: downgrade restricted before `committed_until` unless override.
- Disallow pause when unresolved metered usage settlement is pending.
- Handle timezone-aware effective dates around DST boundaries.
- Idempotent change requests via idempotency key.

## 8) Security and Compliance
- Authorization checks for cross-customer access.
- Immutable snapshots for commercial terms at change time.
- Audit every lifecycle mutation with actor + reason.

## 9) Reporting and Observability
- Metrics: activation rate, churn rate, upgrade/downgrade mix, pause rate.
- Alerts: scheduler lag for lifecycle jobs, failed renewal transitions.

## 10) Acceptance Criteria and Test Scenarios
### Acceptance Criteria
- Lifecycle transitions follow state machine constraints.
- Proration previews match final billed adjustments.
- Scheduled actions execute exactly once at intended times.

### Test Scenarios
1. Create trial subscription and auto-activate at trial end.
2. Mid-cycle upgrade with immediate proration -> correct debit line appears.
3. End-of-period cancellation -> service remains until period end.
4. Duplicate change request with same idempotency key -> single mutation.

## 11) Domain Dependencies and Sequencing
- Depends on Customer, Plan & Pricing, Discounts, and Usage policies.
- Produces inputs for Payments, Taxation, Billing, Notifications, and Dunning.
