# Domain 04: Usage & Metering

## 1) Objective and Business Outcomes
Capture, validate, aggregate, and bill customer usage events accurately for metered and hybrid pricing models.

## 2) Scope
### In Scope
- Usage event ingestion APIs
- Idempotent event storage and deduplication
- Aggregation by metric and billing window
- Late-arriving event handling and correction policies
- Usage previews for customer transparency

### Out of Scope
- Pricing definition (Plan & Pricing domain)
- Invoice issuance (Billing & Invoicing domain)

## 3) Actors and Permissions
- **System Integrator**: pushes usage events.
- **Billing Engine**: reads aggregates for invoice computation.
- **Support/Finance**: applies approved manual adjustments.

## 4) Data Model
- `usage_metric`: id, code, name, unit, aggregation_method(`sum`,`max`,`distinct_count`).
- `usage_event`: id, idempotency_key, customer_id, subscription_id, metric_id, quantity, event_time, received_at, source.
- `usage_event_dedupe`: idempotency_key, fingerprint, first_seen_at.
- `usage_aggregate`: id, subscription_id, metric_id, period_start, period_end, quantity, version.
- `usage_adjustment`: id, aggregate_id, delta_quantity, reason, approved_by, created_at.
- `usage_cutoff_policy`: metric_id, close_after_hours, late_event_mode(`next_cycle`,`credit_note`,`reject`).

## 5) API Surface (REST + Events)
### REST
- `POST /v1/usage/events`
- `POST /v1/usage/events/batch`
- `GET /v1/subscriptions/{subscription_id}/usage?from=&to=`
- `POST /v1/usage/aggregates/recompute`
- `POST /v1/usage/adjustments`

### Events
- `metering.usage_event.ingested`
- `metering.aggregate.updated`
- `metering.adjustment.created`

## 6) Core Workflows and State Transitions
1. **Ingest Usage Event**: validate schema -> dedupe via idempotency key -> persist raw event -> emit ingestion event.
2. **Aggregate Pipeline**: roll up events into billing periods based on subscription anchors/timezone policy.
3. **Window Closure**: lock aggregates at cutoff; late events follow configured late-event mode.
4. **Adjustment Flow**: finance-approved delta updates with immutable reason codes.

State model:
- Aggregate: `open -> closing -> closed -> adjusted`

## 7) Validation Rules and Edge Cases
- Reject negative usage unless metric allows reversals.
- Reject events without metric definition or inactive subscription binding.
- Handle out-of-order events and clock skew.
- Enforce maximum batch size and payload size.
- Support partial batch acceptance with per-record error details.

## 8) Security and Compliance
- Signed ingestion keys per integration source.
- Replay protection with idempotency window + nonce policies.
- Full audit trail on manual adjustments and recomputes.

## 9) Reporting and Observability
- Metrics: event ingest throughput, dedupe ratio, late event ratio, aggregation lag.
- SLOs: aggregation freshness (< 15 min for near-real-time metrics).
- Tracing: correlation from usage event -> aggregate -> invoice line item.

## 10) Acceptance Criteria and Test Scenarios
### Acceptance Criteria
- Duplicate events do not double-bill.
- Aggregates are deterministic for a closed billing window.
- Late events are processed according to policy without silent loss.

### Test Scenarios
1. Send same event twice with same idempotency key -> one accepted, one deduped.
2. Send out-of-order events within open window -> aggregate total correct.
3. Submit late event after close with `next_cycle` policy -> next cycle adjustment created.
4. Batch with one malformed record -> partial success and detailed error list.

## 11) Domain Dependencies and Sequencing
- Depends on Plan & Pricing for metered component definitions and tiers.
- Feeds Subscription Lifecycle (entitlement checks), Billing, and Analytics.
