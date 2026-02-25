# Domain 02: Plan & Pricing Management

## 1) Objective and Business Outcomes
Provide a versioned product catalog and pricing engine inputs so subscriptions can be sold, changed, and invoiced with predictable commercial terms.

## 2) Scope
### In Scope
- Product and plan catalog management
- Recurring, one-time, and metered price components
- Price versioning with effective dates
- Currency-specific pricing
- Trial settings and minimum commitment options

### Out of Scope
- Usage ingestion (covered by Usage & Metering)
- Invoice generation (covered by Billing & Invoicing)

## 3) Actors and Permissions
- **Catalog Admin**: create/update plans and prices.
- **Finance Admin**: approve pricing changes and activation windows.
- **Support Agent**: read catalog for customer operations.

## 4) Data Model
- `product`: id, code, name, description, active.
- `plan`: id, product_id, code, name, billing_model(`flat`,`per_seat`,`metered`,`hybrid`), active.
- `plan_version`: id, plan_id, version, effective_from, effective_to, change_reason.
- `price_component`: id, plan_version_id, type(`recurring`,`one_time`,`metered`), billing_period(`monthly`,`yearly`), amount_minor, currency.
- `tier_rule`: id, price_component_id, start_unit, end_unit, unit_amount_minor, mode(`graduated`,`volume`).
- `plan_constraint`: id, plan_version_id, min_term_months, notice_period_days, seat_min, seat_max.

## 5) API Surface (REST + Events)
### REST
- `POST /v1/catalog/products`
- `POST /v1/catalog/plans`
- `POST /v1/catalog/plans/{plan_id}/versions`
- `POST /v1/catalog/plan-versions/{version_id}/price-components`
- `POST /v1/catalog/plan-versions/{version_id}/activate`
- `GET /v1/catalog/plans?active=true&currency=USD`

### Events
- `catalog.plan.created`
- `catalog.plan_version.created`
- `catalog.plan_version.activated`
- `catalog.price_component.updated`

## 6) Core Workflows and State Transitions
1. **Create Plan Draft** -> add price components -> validate constraints -> activate version.
2. **New Version Rollout**: new plan version becomes effective for new subscriptions; existing subscriptions follow migration policy.
3. **Sunset Plan**: disallow new signups, keep renewals for grandfathered subscriptions.

State model:
- Plan version: `draft -> approved -> active -> retired`

## 7) Validation Rules and Edge Cases
- Prevent overlapping effective windows for same plan + currency.
- Require at least one recurring component for renewable plans.
- Metered component requires unit definition and aggregation strategy.
- Reject negative amounts unless explicitly marked as credit component.
- Enforce seat_min <= seat_max when both are present.

## 8) Security and Compliance
- Dual-approval workflow for price increases above configured threshold.
- Immutable audit records for activated versions.
- RBAC restrictions for activation endpoint.

## 9) Reporting and Observability
- Metrics: active plan count, version activation frequency, pricing change lead time.
- Alerts: activation failures, overlapping pricing windows, invalid tiers.

## 10) Acceptance Criteria and Test Scenarios
### Acceptance Criteria
- Catalog supports multiple currencies and price versions.
- Activation prevents invalid/overlapping configurations.
- Existing subscriptions can resolve historical price version deterministically.

### Test Scenarios
1. Create plan with monthly + yearly components -> activate -> retrieve active list.
2. Attempt overlapping effective dates -> expect `422`.
3. Add invalid tier range (`end < start`) -> expect validation failure.
4. Sunset plan and verify new subscription checkout blocks it.

## 11) Domain Dependencies and Sequencing
- Depends on Customer Management for account assignment semantics.
- Feeds Discounts, Subscription Lifecycle, Billing, and Analytics domains.
