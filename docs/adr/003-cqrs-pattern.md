# ADR-003: CQRS Pattern

## Status
Accepted

## Date
2026-03-06

## Context

The challenge requires demonstrating CQRS (Command Query Responsibility Segregation). We need to decide the level of CQRS implementation appropriate for a single-service authentication module.

## Decision

We implemented **logical CQRS** — separate command handlers (write side) and query handlers (read side) at the application layer, sharing the same database.

### Command Handlers (Write Side)

| Handler | Command | Effect |
|---|---|---|
| `RegisterUserHandler` | `RegisterUserCommand` | Creates User, hashes password, emits `UserRegistered` |
| `AuthenticateUserHandler` | `AuthenticateUserCommand` | Verifies credentials, creates Session, issues tokens |
| `RefreshTokenHandler` | `RefreshTokenCommand` | Rotates tokens, updates Session |
| `RequestPasswordResetHandler` | `RequestPasswordResetCommand` | Generates reset token, updates User |
| `ResetPasswordHandler` | `ResetPasswordCommand` | Validates token, updates password |

### Query Handlers (Read Side)

| Handler | Query | Returns |
|---|---|---|
| `GetUserByIDHandler` | `GetUserByIDQuery` | `UserDTO` (read-model projection) |
| `GetUserSessionsHandler` | `GetUserSessionsQuery` | `[]SessionDTO` |

## Rationale

### Why CQRS for Auth?

1. **Auth is write-heavy** — register, login, refresh, reset are all mutations. Read operations (get profile) have fundamentally different patterns and performance characteristics.

2. **Separation of concerns** — command handlers enforce business rules and emit events. Query handlers return simplified DTOs without loading aggregate complexity (no need to reconstruct domain objects for a profile lookup).

3. **Independent scalability (future)** — if read traffic grows (e.g., `me` query called on every page navigation), the read side can be independently scaled with read replicas and caching, while the write side maintains strong consistency.

4. **Audit trail** — commands explicitly represent intentions. Each command handler logs what was attempted and what happened. Combined with domain events, this provides a clear audit path.

### Level of CQRS: Logical, Not Physical

We chose **logical separation** (same DB, same codebase) rather than **physical separation** (separate databases, event sourcing, eventual consistency).

This is appropriate because:
- Single service, single database — adding event sourcing and projections would be over-engineering
- Auth requires strong consistency (a user should be able to log in immediately after registering)
- The command/query handler pattern provides the structural benefit of CQRS without the operational complexity of separate stores

### DTOs vs. Domain Objects

Query handlers return DTOs, not domain aggregates:
- `UserDTO` has no domain methods, no events, no reset token (only `id`, `email`, `status`, `created_at`)
- `SessionDTO` adds a computed `IsActive` field

This means the read side never accidentally triggers mutations on domain objects.

## Alternatives Considered

| Approach | Why Not |
|---|---|
| Full Event Sourcing + Projections | Massive overkill for auth. Events are useful for audit, but reconstructing user state from event history adds complexity without benefit for 3 user scenarios. |
| No CQRS (single service methods) | Mixes read and write concerns. Service methods that both mutate and return data are harder to reason about, optimize, and test independently. |
| Separate read database | Adds operational complexity (replication lag, consistency guarantees) that isn't justified for auth workloads at this scale. |

## Consequences

- **Positive:** Clear intent in code (commands mutate, queries read), each handler is independently testable, natural evolution path to event sourcing if needed.
- **Negative:** More boilerplate (command struct + result struct + handler per operation), some duplication between DTO and aggregate fields.
- **Mitigation:** Keep DTOs minimal (only what the API needs), handler constructors use dependency injection for testability.
