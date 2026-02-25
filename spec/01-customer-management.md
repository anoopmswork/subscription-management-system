# Domain 01: Customer Management

## 1) Objective and Business Outcomes
Create a reliable customer/account system that supports tenant isolation, multiple contacts, and ownership controls used by subscription, billing, and payment domains.

## 2) Scope
### In Scope
- Customer account creation and lifecycle (`prospect`, `active`, `suspended`, `archived`)
- Billing and service addresses
- Primary/secondary contacts
- Organization hierarchies (parent-child account)
- Customer preferences (locale, timezone, currency)
- Role-based access for customer users (owner, admin, billing, viewer)

### Out of Scope
- Identity provider implementation details (SSO internals)
- CRM opportunity pipeline management

## 3) Actors and Permissions
- **Platform Admin**: full read/write across tenants.
- **Customer Owner**: manage account settings, billing contacts, tax IDs.
- **Billing Manager**: manage invoices, payment methods, subscription assignments.
- **Viewer**: read-only access to account profile and billing artifacts.

## 4) Data Model
- `customer_account`: id, external_ref, legal_name, display_name, status, segment, created_at.
- `customer_contact`: id, customer_id, name, email, phone, role, is_primary, verified_at.
- `customer_address`: id, customer_id, type(`billing`,`service`), line1..lineN, city, region, postal_code, country.
- `customer_tax_profile`: id, customer_id, tax_id_type, tax_id_value(masked), exemption_status, validation_state.
- `customer_user`: id, customer_id, user_id, role, invited_at, accepted_at, status.
- `customer_preference`: customer_id, locale, timezone, invoice_delivery_channel, default_currency.

Relationships:
- One account has many contacts/users/addresses.
- One account can have one parent account; many child accounts.

## 5) API Surface (REST + Events)
### REST
- `POST /v1/customers`
- `GET /v1/customers/{customer_id}`
- `PATCH /v1/customers/{customer_id}`
- `POST /v1/customers/{customer_id}/contacts`
- `PATCH /v1/customers/{customer_id}/contacts/{contact_id}`
- `POST /v1/customers/{customer_id}/users/invite`
- `PATCH /v1/customers/{customer_id}/status`

### Events
- `customer.account.created`
- `customer.account.updated`
- `customer.account.status_changed`
- `customer.contact.added`
- `customer.user.invited`

## 6) Core Workflows and State Transitions
1. **Create Account**: validate uniqueness (`external_ref`, legal name within tenant) -> create default preferences -> emit `customer.account.created`.
2. **Suspend Account**: block new subscription creation, keep historical invoices accessible.
3. **Archive Account**: only allowed if no active subscriptions and no open invoice balance.
4. **Contact Verification**: email verification required for invoice email destination.

State transitions:
- `prospect -> active`
- `active -> suspended -> active`
- `active|suspended -> archived`

## 7) Validation Rules and Edge Cases
- Reject duplicate primary billing contact.
- Require country + postal code for taxable geographies.
- Prevent archive if `open_invoice_balance > 0`.
- Handle parent-child roll-up without cyclic references.
- Support contact email changes with re-verification.

## 8) Security and Compliance
- Encrypt PII fields at rest (email, phone, address lines).
- Field-level masking for tax IDs in APIs and logs.
- Fine-grained RBAC checks on every update endpoint.
- Audit log entries for all permission and status changes.

## 9) Reporting and Observability
- Metrics: customer creation rate, activation lag, suspension rate.
- Logs: account state changes with actor and reason.
- Traces: include `customer_id` correlation in downstream billing/payment calls.

## 10) Acceptance Criteria and Test Scenarios
### Acceptance Criteria
- Can create customer with valid legal profile and primary contact.
- Cannot archive customer with active subscriptions or open balance.
- Role enforcement prevents viewer from mutating customer data.
- State transition events are emitted exactly once.

### Test Scenarios
1. Create account with valid inputs -> expect `201` and event.
2. Attempt duplicate primary contact -> expect `409`.
3. Suspend and reactivate account -> verify subscription creation policy toggles.
4. Archive blocked when unpaid invoice exists -> expect `422`.

## 11) Domain Dependencies and Sequencing
- **Prerequisite for** all remaining domains.
- Must be implemented before plan assignment, billing, payment method linkage, and tax profile resolution.
