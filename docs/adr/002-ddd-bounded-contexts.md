# ADR-002: DDD with Two Bounded Contexts

## Status
Accepted

## Date
2026-03-06

## Context

The authentication module handles three user scenarios: registration, login, and password reset. We need a domain model that clearly separates concerns, enforces business invariants, and enables independent evolution of different aspects of user management and authentication.

## Decision

We structured the domain into **two bounded contexts**:

1. **Identity Context** (`internal/domain/identity/`) — owns the User aggregate, email/password validation, and password reset flow.
2. **Auth Context** (`internal/domain/auth/`) — owns the Session aggregate, token pairs, and session lifecycle.

Each context has its own:
- Aggregate root (User, Session)
- Value objects
- Domain events
- Repository port (interface)

## Rationale

### Why Two Contexts, Not One?

**Identity** answers "Who is this person?" — it manages the user's existence, credentials, and account state. A User exists from the moment they register, regardless of whether they've ever logged in.

**Authentication** answers "Is this person who they claim to be?" — it manages sessions, tokens, and proof of identity. A Session only exists during an active authentication. Sessions belong to users but are a separate lifecycle.

Separating them means:
- **Adding OAuth/SSO** only touches Identity (new credential types, user linking) without changing anything about sessions.
- **Switching from JWT to opaque tokens** only touches Auth without affecting user management.
- **Account deletion** cascades naturally — deleting a User (Identity) removes their Sessions (Auth) via DB foreign key.

### DDD Tactical Patterns Used

| Pattern | Implementation | Purpose |
|---|---|---|
| **Aggregate Root** | `User`, `Session` | Consistency boundary — all changes go through the root |
| **Value Object** | `Email`, `Password`, `UserID`, `ResetToken`, `SessionID`, `TokenPair` | Immutable, validated types with no identity |
| **Domain Event** | `UserRegistered`, `PasswordResetRequested`, `UserAuthenticated`, `SessionRevoked` | Record facts about what happened |
| **Repository (Port)** | `UserRepository`, `SessionRepository` interfaces | Domain defines what it needs; infrastructure provides it |
| **Factory Method** | `Register()`, `CreateSession()` | Encapsulate aggregate creation with invariant enforcement |
| **Reconstruct** | `Reconstruct()`, `ReconstructSession()` | Hydrate from persistence without triggering events |

### Cross-Context Communication

Currently, the application layer (command handlers) coordinates between contexts — e.g., `AuthenticateUserHandler` reads from `UserRepository` and writes to `SessionRepository`. This is acceptable for a single-service deployment.

In a microservice evolution, contexts would become separate services communicating via:
- gRPC for synchronous queries
- Domain events via message broker for async notifications

## Alternatives Considered

| Approach | Why Not |
|---|---|
| Single `auth` context | Mixes user management and session management. Adding OAuth would require touching session code. Violates Single Responsibility. |
| Three contexts (Identity + Auth + PasswordReset) | Password reset is an operation on the User aggregate, not a separate entity with its own lifecycle. Over-fragmenting degrades cohesion. |
| Anemic domain model (entities with only getters/setters, logic in services) | Violates DDD principles. Business rules should live in the domain, not be scattered across service methods. |

## Consequences

- **Positive:** Clear separation of concerns, each context is independently understandable, invariants are enforced at the aggregate level, events enable future decoupling.
- **Negative:** More packages, more files, cross-context coordination in the application layer, domain events are duplicated (each context has its own `DomainEvent` interface).
- **Mitigation:** Keep-it-simple where possible — shared event interface could be extracted to a `shared/kernel` package if contexts multiply.
