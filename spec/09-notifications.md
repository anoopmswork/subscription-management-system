# Domain 09: Notifications

## 1) Objective and Business Outcomes
Deliver timely, compliant, and preference-aware communications for subscription, billing, payment, and dunning events.

## 2) Scope
### In Scope
- Notification templates and localization
- Channel orchestration (email, SMS, in-app, webhook)
- Customer notification preferences and opt-outs
- Event-triggered and scheduled notifications
- Delivery tracking and retry

### Out of Scope
- Campaign marketing automation beyond billing lifecycle notices
- Third-party CRM email journeys

## 3) Actors and Permissions
- **Comms Admin**: manage templates/channels.
- **Billing/Dunning Services**: emit trigger events.
- **Customer User**: manage preferences and destinations.

## 4) Data Model
- `notification_template`: id, code, channel, locale, subject, body, active_version.
- `notification_preference`: id, customer_id, event_type, channel, enabled.
- `notification_message`: id, customer_id, event_type, template_id, channel, payload_ref, status.
- `notification_delivery`: id, message_id, provider, provider_ref, status, sent_at, delivered_at, failure_reason.
- `notification_schedule`: id, event_ref, send_at, status.

## 5) API Surface (REST + Events)
### REST
- `POST /v1/notifications/templates`
- `POST /v1/notifications/send`
- `GET /v1/customers/{customer_id}/notification-preferences`
- `PATCH /v1/customers/{customer_id}/notification-preferences`
- `POST /v1/notifications/webhooks/{provider}`

### Events
- `notification.message.queued`
- `notification.message.sent`
- `notification.message.delivered`
- `notification.message.failed`

## 6) Core Workflows and State Transitions
1. **Trigger Notification**: consume domain event -> resolve template + locale -> check preference -> queue delivery.
2. **Channel Delivery**: send via provider -> handle callback -> update delivery status.
3. **Scheduled Reminders**: emit reminder notices before renewal/due dates.

State model:
- Message: `queued -> sending -> sent -> delivered|failed|suppressed`

## 7) Validation Rules and Edge Cases
- Suppress notifications when customer opted out for non-mandatory events.
- Mandatory legal notices ignore marketing opt-out.
- Handle invalid destination addresses with fallback channel rules.
- Prevent duplicate sends on repeated source events using event dedupe.

## 8) Security and Compliance
- PII-safe template rendering and log redaction.
- Store provider webhook signatures for verification.
- Regional data residency controls for notification payloads.

## 9) Reporting and Observability
- Metrics: send success rate, delivery latency, failure reasons by provider.
- Dashboards: event-type volume and channel effectiveness.
- Alerts: provider outage, bounce spike, queue backlog growth.

## 10) Acceptance Criteria and Test Scenarios
### Acceptance Criteria
- Event-triggered notifications honor preferences and legal requirements.
- Delivery retries occur per configured policy without duplicates.
- Localized templates render correctly for selected locale.

### Test Scenarios
1. Invoice finalized event -> email sent to billing contact.
2. Opted-out optional event -> notification suppressed.
3. Provider transient error -> retry then succeed.
4. Duplicate source event -> single message emitted.

## 11) Domain Dependencies and Sequencing
- Depends on core domain events (Subscription, Billing, Payments, Dunning).
- Supports customer experience and recovery outcomes.
