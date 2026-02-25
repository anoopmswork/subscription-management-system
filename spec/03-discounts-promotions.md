# Domain 03: Discounts & Promotions

## 1) Objective and Business Outcomes
Enable controlled discounting and campaign promotions to improve acquisition and retention while preserving billing correctness and auditability.

## 2) Scope
### In Scope
- Coupon and promotion code lifecycle
- Percentage, fixed amount, and trial-extension offers
- Eligibility rules (plan, segment, region, channel)
- Duration rules (one-time, repeating N cycles, forever)
- Stacking and precedence rules

### Out of Scope
- Price catalog creation (Plan & Pricing domain)
- Manual invoice adjustments after issuance (Billing domain)

## 3) Actors and Permissions
- **Growth Admin**: create campaigns, define rules, publish codes.
- **Finance Admin**: approve high-impact promotions.
- **Support Agent**: apply or revoke promo with reason and audit trail.

## 4) Data Model
- `promotion`: id, name, type(`coupon`,`campaign`), status, start_at, end_at, max_redemptions.
- `promotion_code`: id, promotion_id, code, status, redemption_limit_per_customer.
- `discount_rule`: id, promotion_id, value_type(`percent`,`fixed`,`trial_days`), value, applies_to(`subscription`,`line_item`,`usage`).
- `eligibility_rule`: id, promotion_id, plan_ids, customer_segments, countries, channels.
- `promotion_redemption`: id, promotion_id, customer_id, subscription_id, invoice_id, redeemed_at, revoked_at.
- `stacking_policy`: id, promotion_id, stack_group, priority, combinable.

## 5) API Surface (REST + Events)
### REST
- `POST /v1/promotions`
- `POST /v1/promotions/{promotion_id}/codes`
- `POST /v1/promotions/{promotion_id}/publish`
- `POST /v1/promotions/validate`
- `POST /v1/subscriptions/{subscription_id}/apply-promotion`
- `POST /v1/subscriptions/{subscription_id}/revoke-promotion`

### Events
- `discount.promotion.created`
- `discount.promotion.published`
- `discount.redemption.created`
- `discount.redemption.revoked`

## 6) Core Workflows and State Transitions
1. **Create Promotion Draft** -> configure discount + eligibility + stacking policy -> publish.
2. **Redeem Code at Checkout** -> validate eligibility + date window + usage caps -> reserve redemption -> apply discount snapshot.
3. **Recurring Billing Application**: apply repeating discounts per cycle until duration exhaustion.
4. **Revocation**: allow admin revocation for abuse/fraud with immutable audit reason.

State model:
- Promotion: `draft -> scheduled -> active -> expired|disabled`

## 7) Validation Rules and Edge Cases
- Prevent code collisions (case-insensitive).
- Enforce max percent <= 100.
- Fixed discounts cannot reduce taxable base below zero unless policy allows carry-forward credits.
- Handle concurrent redemption race using row-level locks/idempotency.
- Preserve original discount snapshot on invoices after promotion is edited.

## 8) Security and Compliance
- Rate-limit validation endpoint to reduce brute-force code guessing.
- Mask promotion-code lookups in logs when marked confidential.
- Audit all manual applies/revocations with operator identity.

## 9) Reporting and Observability
- Metrics: redemption rate, promo-attributed MRR, abuse detection signals.
- Dashboards: campaign conversion by channel/region.
- Alerts: unusual spike in redemptions, failed validation latency.

## 10) Acceptance Criteria and Test Scenarios
### Acceptance Criteria
- Promotion application is deterministic and reproducible for invoice audits.
- Stacking/priority rules are enforced consistently.
- Expired or ineligible codes are rejected with clear error reasons.

### Test Scenarios
1. Apply eligible percent code at checkout -> invoice preview reflects discount.
2. Exceed per-customer redemption limit -> expect `409`.
3. Stack two non-combinable promotions -> only higher priority applied.
4. Edit active promotion -> historical invoice amount remains unchanged.

## 11) Domain Dependencies and Sequencing
- Depends on Plan & Pricing for targets and price bases.
- Feeds Subscription Lifecycle and Billing calculation inputs.
