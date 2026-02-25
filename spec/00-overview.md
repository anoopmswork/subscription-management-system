# Subscription Management System - Domain Spec Overview

## Vision
Build a full-fledged, API-first subscription management platform that supports B2B and B2C recurring revenue models with global billing, compliant taxation, reliable payment recovery, and extensible integrations.

## Domain Specs (Dependency-First Order)

| Order | Domain | Spec File | Why It Comes Here |
|---|---|---|---|
| 01 | Customer Management | `01-customer-management.md` | Core identity, accounts, and ownership model used by all other domains. |
| 02 | Plan & Pricing Management | `02-plan-pricing-management.md` | Defines sellable catalog and pricing terms for subscriptions and invoices. |
| 03 | Discounts & Promotions | `03-discounts-promotions.md` | Pricing modifiers must be resolved before billing and invoice calculations. |
| 04 | Usage & Metering | `04-usage-metering.md` | Captures billable usage events required for metered invoice line items. |
| 05 | Subscription Lifecycle | `05-subscription-lifecycle.md` | Orchestrates trial/active/pause/cancel state transitions and plan changes. |
| 06 | Payments | `06-payments.md` | Collects money through PSPs with retries, auth/capture, and reconciliation. |
| 07 | Taxation | `07-taxation.md` | Tax determination must be finalized before invoice totals are locked. |
| 08 | Billing & Invoicing | `08-billing-invoicing.md` | Generates legally correct invoices and credit notes from all prior inputs. |
| 09 | Notifications | `09-notifications.md` | Event-driven customer and operator communication after core billing flows exist. |
| 10 | Dunning & Recovery | `10-dunning-recovery.md` | Uses billing/payment states to recover failed renewals and reduce churn. |
| 11 | Admin & Analytics | `11-admin-analytics.md` | Cross-domain operations, financial reporting, and monitoring dashboards. |
| 12 | Integrations | `12-integrations.md` | External sync and webhook ecosystem built on stable domain contracts. |

## Global Product Principles
1. **Idempotent financial operations** for all mutation endpoints.
2. **Auditability** for every monetary and state-changing action.
3. **Event-driven architecture** with durable outbox and replay support.
4. **Multi-currency and timezone-safe** calculations in all billing flows.
5. **Composable domain boundaries** with clear ownership of data and APIs.

## Shared Non-Functional Requirements
- Availability target: 99.9% for API and billing workflows.
- RPO/RTO: RPO <= 5 min, RTO <= 30 min for critical financial data.
- Performance: P95 API latency < 300 ms for read APIs and < 800 ms for write APIs (excluding external PSP/tax calls).
- Security: encryption in transit and at rest, principle of least privilege, periodic key rotation.
- Compliance: GDPR/CCPA, PCI-DSS scope minimization via tokenized payment methods, SOC2-aligned controls.

## Shared Technical Conventions
- IDs: UUIDv7 for domain objects.
- Money: integer minor units (`amount_minor`) + ISO-4217 currency.
- Time: UTC storage, tenant/user timezone for display.
- Versioning: semantic API versioning (`/v1`) + resource-level immutable revision records.
- Event naming: `<domain>.<entity>.<action>` (e.g., `billing.invoice.finalized`).
- Exactly-once semantics: idempotency keys + dedupe tables for critical writes.

## Implementation Tracking

| Domain | Status | Notes |
|---|---|---|
| Customer Management | Ready | Spec drafted |
| Plan & Pricing | Ready | Spec drafted |
| Discounts & Promotions | Ready | Spec drafted |
| Usage & Metering | Ready | Spec drafted |
| Subscription Lifecycle | Ready | Spec drafted |
| Payments | Ready | Spec drafted |
| Taxation | Ready | Spec drafted |
| Billing & Invoicing | Ready | Spec drafted |
| Notifications | Ready | Spec drafted |
| Dunning & Recovery | Ready | Spec drafted |
| Admin & Analytics | Ready | Spec drafted |
| Integrations | Ready | Spec drafted |

## Next Execution Pattern
Implement one domain at a time in the same order as above. For each domain:
1. Confirm data model migrations.
2. Build APIs and domain services.
3. Add tests (unit + integration + contract).
4. Wire domain events.
5. Run validators and release behind feature flags.
