---
name: documentation-skill
description: Validate and add production-grade documentation for public/exported classes, interfaces, structs, functions, and methods while preserving behavior.
---

# Documentation Verification Skill

## When to use

Use this skill when code documentation must be created, improved, or verified across classes, interfaces, structs, functions, and methods.

## Inputs

- Target files or directories
- Language(s) in scope
- Relevant domain/context from existing code and tests

## Required workflow

1. Identify the full public/exported API surface in the requested scope.
2. Read implementation and tests before writing any documentation.
3. Add or update documentation comments/docstrings only.
4. Validate that documentation matches business logic and actual behavior.

## Non-negotiable rules

1. Do not change logic, signatures, behavior, or API contracts.
2. Do not refactor or rename symbols.
3. Only add or update documentation comments/docstrings.
4. Keep documentation concise, accurate, architecture-aware, and business-rule aware.
5. Do not leave placeholder docs (for example: TODO, TBD, FIXME).

## Documentation requirements

For every public/exported symbol in scope:

1. Structs/classes:
   - Purpose and responsibility
   - Architectural role
   - Important invariants
2. Interfaces:
   - Contract and expected behavior
   - Implementation guarantees
3. Functions/methods:
   - What and why
   - Parameters and return values
   - Possible errors
   - Side effects
   - Concurrency considerations (if any)
4. Business logic:
   - Document key business rules and constraints near the enforcing code.
   - Include validation rules (required fields, ranges, formats), state transitions, authorization checks, and idempotency behavior when relevant.

## Business logic validation requirements

For each documented symbol, confirm comments align with:

1. Validation and domain constraints enforced in code.
2. State transitions and rejected paths.
3. Error conditions and error type semantics.
4. Data persistence/transaction expectations.
5. Existing tests and current callers.

## Language style rules

- Go: GoDoc comments that start with the symbol name.
- TypeScript/JavaScript: JSDoc/TSDoc.
- Python: docstrings.
- Java: Javadoc.

## Verification checklist

1. Coverage: all public/exported symbols in changed scope are documented.
2. Accuracy: docs match actual behavior, params, returns, and errors.
3. Business logic: docs correctly describe enforced business rules and constraints.
4. Safety: diff contains documentation-only changes.
5. Repository validation command:

```bash
go test ./...
```

## Success criteria

- Documentation added/updated for target symbols.
- Business rules documented where logic is enforced.
- No logic changes introduced.
- Validation command passes.

## Output format

- Files updated
- Symbols documented
- Business-rule notes added
- Verification result