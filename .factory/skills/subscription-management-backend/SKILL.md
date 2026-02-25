---
name: subscription-management-backend
description: Implement and modify backend work in this repository using its existing Go service architecture and validation patterns.
---

# Subscription Management Backend

## When to use

Use this skill for changes in this repo, especially under `cmd/api` and `internal/*`.

## Instructions

1. Keep the existing layer boundaries:
   - `internal/http/api`: request parsing, status codes, and response mapping.
   - `internal/customer`: business rules, validation, and orchestration.
   - `internal/customer/postgres_repository.go`: persistence and SQL interactions.
   - `internal/domain/customer`: core model types and enums.
2. Keep authorization and validation behavior consistent with current service methods and `respondError` mapping.
3. Add or update tests when behavior changes, primarily in:
   - `internal/customer/service_test.go`
   - `internal/http/api/handler_test.go`
4. Prefer existing dependencies and patterns already used in this repository.

## Verification

Run:

```bash
go test ./...
```

## Success criteria

- Changes are made in the correct layer(s).
- Behavior changes are covered by tests.
- `go test ./...` passes.
