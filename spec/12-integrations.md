# Domain 12: Integrations

## 1) Objective and Business Outcomes
Expose stable integration surfaces for ecosystem systems (CRM, ERP, accounting, data warehouse, support tools) with secure and reliable data exchange.

## 2) Scope
### In Scope
- Outbound webhooks for domain events
- Inbound API connectors and scheduled sync jobs
- ERP/accounting export contracts
- Data warehouse event/CDC pipelines
- Integration observability and replay

### Out of Scope
- Partner marketplace UI and app store billing
- Internal domain business logic changes

## 3) Actors and Permissions
- **Integration Admin**: configure endpoints, secrets, mappings.
- **External Systems**: consume webhooks and call APIs.
- **Platform Ops**: monitor failures, replay events.

## 4) Data Model
- `integration_endpoint`: id, name, type(`webhook`,`pull_api`,`sftp`,`queue`), url, auth_type, status.
- `integration_subscription`: id, endpoint_id, event_type, filter_json, active.
- `integration_delivery`: id, endpoint_id, event_id, attempt_no, status, response_code, latency_ms.
- `integration_secret`: id, endpoint_id, key_id, rotated_at, expires_at.
- `sync_job`: id, connector_type, direction(`inbound`,`outbound`), schedule, status, cursor, last_run_at.
- `mapping_profile`: id, connector_type, entity_type, mapping_json, version.

## 5) API Surface (REST + Events)
### REST
- `POST /v1/integrations/endpoints`
- `POST /v1/integrations/endpoints/{endpoint_id}/rotate-secret`
- `POST /v1/integrations/subscriptions`
- `POST /v1/integrations/deliveries/{delivery_id}/replay`
- `POST /v1/integrations/sync-jobs/{job_id}/run`

### Events
- `integration.endpoint.created`
- `integration.delivery.failed`
- `integration.delivery.retried`
- `integration.sync.completed`

## 6) Core Workflows and State Transitions
1. **Webhook Delivery**: publish domain event -> sign payload -> send -> retry with exponential backoff on failures.
2. **Replay**: operator replays failed deliveries by event range or endpoint.
3. **Connector Sync**: scheduled pull/push jobs with checkpoint cursors and idempotent upserts.
4. **Schema Evolution**: versioned payload contracts with deprecation windows.

State model:
- Delivery: `queued -> sending -> delivered|failed|dead_lettered`

## 7) Validation Rules and Edge Cases
- Verify endpoint URL format and TLS requirements.
- Reject unsupported payload schema versions.
- Handle partner-side rate limiting with retry/backoff and jitter.
- Preserve event order per aggregate key when required.
- Ensure replays do not violate idempotency of downstream systems.

## 8) Security and Compliance
- HMAC signing for outbound webhooks.
- Secret rotation with overlap windows.
- IP allowlisting and optional mTLS for sensitive endpoints.
- Least-privilege API keys and scoped tokens.

## 9) Reporting and Observability
- Metrics: delivery success rate, retry count, dead-letter volume, sync lag.
- Dashboards: endpoint health and top failing connectors.
- Alerts: prolonged endpoint failure, secret expiration nearing, schema mismatch spikes.

## 10) Acceptance Criteria and Test Scenarios
### Acceptance Criteria
- Webhooks are signed, retryable, and replayable.
- Connector sync jobs are resumable and idempotent.
- Integration contract versions are backward compatible for supported window.

### Test Scenarios
1. Register endpoint and receive signed invoice-finalized webhook.
2. Simulate endpoint 500 errors -> retries then dead-letter after threshold.
3. Replay dead-letter event -> successful redelivery.
4. Run inbound sync with cursor resume after interruption.

## 11) Domain Dependencies and Sequencing
- Depends on stable events and schemas from all core domains.
- Final domain to reduce contract churn during earlier architecture evolution.
