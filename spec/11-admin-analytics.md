# Domain 11: Admin & Analytics

## 1) Objective and Business Outcomes
Provide operational control planes and analytics to monitor revenue, customer lifecycle, billing health, and system reliability.

## 2) Scope
### In Scope
- Admin configuration interfaces and guarded actions
- Financial and subscription KPI dashboards
- Audit logs and operational activity history
- Role-based data access and report exports
- Domain health and SLO monitoring views

### Out of Scope
- External BI tool modeling (handled via Integrations)
- End-user customer portal UX specifics

## 3) Actors and Permissions
- **Super Admin**: cross-domain configuration and emergency controls.
- **Finance Analyst**: financial reporting and exports.
- **Support Ops**: customer-level diagnostics and adjustments.
- **Security Auditor**: read-only compliance and audit access.

## 4) Data Model
- `admin_audit_log`: id, actor_id, action, target_type, target_id, before_json, after_json, created_at.
- `kpi_snapshot`: id, date_bucket, mrr_minor, arr_minor, churn_rate, ltv, cac, nrr.
- `billing_health_snapshot`: id, date_bucket, invoice_success_rate, payment_success_rate, dunning_recovery_rate.
- `report_job`: id, type, filters_json, status, file_uri, requested_by, completed_at.
- `feature_flag`: id, key, scope, enabled, rules_json.

## 5) API Surface (REST + Events)
### REST
- `GET /v1/admin/audit-logs`
- `GET /v1/admin/kpis?from=&to=&granularity=`
- `GET /v1/admin/health/billing`
- `POST /v1/admin/reports/export`
- `PATCH /v1/admin/feature-flags/{flag_key}`

### Events
- `admin.report.generated`
- `admin.feature_flag.changed`
- `admin.audit_log.created`

## 6) Core Workflows and State Transitions
1. **KPI Pipeline**: daily/hourly aggregate jobs materialize financial metrics.
2. **Export Flow**: user requests report -> async job -> signed download link.
3. **Operational Override**: guarded admin action with mandatory reason and approval policy.

State model:
- Report job: `queued -> processing -> completed|failed|expired`

## 7) Validation Rules and Edge Cases
- Enforce row-level access by tenant/legal entity.
- Large exports use async processing and chunking.
- Metrics backfill supports corrected historical events.
- Prevent feature-flag edits without proper role.

## 8) Security and Compliance
- Tamper-evident audit logs.
- Least-privilege report access and expiring signed URLs.
- Sensitive data masking in dashboards and exports by role.

## 9) Reporting and Observability
- Core KPIs: MRR, ARR, NRR, logo churn, revenue churn, AR aging, failed payment rate.
- Operational metrics: queue lag, job failures, API error rates by domain.
- Alerts: KPI pipeline failures, stale dashboards, abnormal churn spikes.

## 10) Acceptance Criteria and Test Scenarios
### Acceptance Criteria
- Admin actions are fully auditable and queryable.
- KPI values are reproducible from source financial events.
- Export jobs are secure, asynchronous, and access-controlled.

### Test Scenarios
1. Request KPI dashboard range and verify expected fields.
2. Generate export and verify signed URL expiry.
3. Unauthorized role attempts feature-flag update -> forbidden.
4. Audit log captures before/after payload for critical update.

## 11) Domain Dependencies and Sequencing
- Depends on all upstream domains for data completeness.
- Provides operational visibility for release governance and scale readiness.
