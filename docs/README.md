# Authentication Module — Solution Documentation

## Table of Contents

1. [Overview](#overview)
2. [Tech Stack & Rationale](#tech-stack--rationale)
3. [Architecture](#architecture)
   - [High-Level Architecture](#high-level-architecture)
   - [DDD: Domain-Driven Design](#ddd-domain-driven-design)
   - [CQRS: Command Query Responsibility Segregation](#cqrs-command-query-responsibility-segregation)
   - [Layered Architecture](#layered-architecture)
4. [Project Structure](#project-structure)
5. [Domain Model](#domain-model)
   - [Identity Bounded Context](#identity-bounded-context)
   - [Auth Bounded Context](#auth-bounded-context)
   - [Domain Events](#domain-events)
   - [Business Invariants & Rules](#business-invariants--rules)
6. [Application Layer (CQRS)](#application-layer-cqrs)
   - [Commands (Write Side)](#commands-write-side)
   - [Queries (Read Side)](#queries-read-side)
7. [Infrastructure Layer](#infrastructure-layer)
   - [Password Hashing (bcrypt)](#password-hashing-bcrypt)
   - [JWT Tokens](#jwt-tokens)
   - [PostgreSQL Persistence](#postgresql-persistence)
   - [Redis Rate Limiting](#redis-rate-limiting)
   - [Event Bus](#event-bus)
   - [Observability](#observability)
8. [API Layer (GraphQL)](#api-layer-graphql)
   - [Schema](#schema)
   - [Authentication Flow](#authentication-flow)
   - [API Examples](#api-examples)
9. [gRPC Protocol (Proto Definition)](#grpc-protocol-proto-definition)
10. [Database Schema](#database-schema)
11. [Security](#security)
12. [How to Run](#how-to-run)
13. [Environment Variables](#environment-variables)
14. [Testing](#testing)
15. [Trade-offs & Design Decisions](#trade-offs--design-decisions)
16. [What I Would Do Next (Production Roadmap)](#what-i-would-do-next-production-roadmap)

---

## Overview

This project implements a **production-grade authentication module** covering three user scenarios:

1. **Registration** — user signs up with email + password
2. **Authentication (Login)** — user logs in, receives JWT access + refresh tokens
3. **Password Reset** — user requests a reset token, then applies a new password

The solution is designed around **DDD (Domain-Driven Design)**, **CQRS (Command Query Responsibility Segregation)**, and is exposed via **GraphQL** with a **gRPC** protocol definition for future service-to-service communication.

---

## Tech Stack & Rationale

| Technology | Role | Why |
|---|---|---|
| **Go 1.25** | Primary language | Excellent for backend services: strong static typing, fast compilation, first-class concurrency, minimal runtime overhead. Go's standard library covers HTTP, JSON, crypto natively. Compiled binary = easy deployment. |
| **PostgreSQL** | Primary data store | ACID-compliant relational DB perfect for auth data where consistency is critical. Native UUID and TIMESTAMPTZ types. Mature ecosystem. |
| **Redis** | Rate limiting | In-memory store ideal for sliding-window counters. Sub-millisecond latency for rate limit checks. |
| **GraphQL** | Client API | Provides a typed, self-documenting schema. Clients request exactly what they need. Better DX for frontend auth forms than REST. |
| **gRPC/Protobuf** | Service-to-service (planned) | Binary protocol with code generation. Ideal for internal microservice communication. Proto definition ready for implementation. |
| **JWT (HMAC-SHA256)** | Token format | Stateless access tokens reduce DB lookups. HMAC-SHA256 is efficient and doesn't require PKI for single-service setups. |
| **bcrypt (cost=12)** | Password hashing | Industry standard adaptive hashing. Cost 12 provides ~250ms per hash — slow enough to resist brute force, fast enough for UX. |
| **Prometheus** | Metrics | De-facto standard for cloud-native monitoring. Pull-based model works well with Kubernetes. |
| **slog** | Structured logging | Go 1.21+ standard library structured logger. JSON output, zero dependencies, production-ready. |

### Alternatives Considered

| Considered | Rejected Because |
|---|---|
| **Node.js/TypeScript** | Go's type system + compiled binary is stronger for auth-critical services. Node's single-threaded model adds complexity for CPU-bound bcrypt operations. |
| **Rust** | Excellent for systems programming but development velocity is slower. Go strikes a better balance for this scope. |
| **REST** | GraphQL provides better client ergonomics for auth flows (typed schema, single endpoint, no versioning). REST would require more endpoints and manual documentation. |
| **Argon2id** | Superior to bcrypt theoretically, but bcrypt is battle-tested, universally supported, and sufficient for this threat model. |
| **RSA/ECDSA for JWT** | Asymmetric signing required for multi-service JWT validation. Unnecessary for a single-service setup — HMAC-SHA256 is simpler and faster. |
| **MongoDB** | Auth data is inherently relational (users → sessions). NoSQL adds complexity without benefit here. |

---

## Architecture

### High-Level Architecture

```
┌──────────────────────────────────────────────────────┐
│                    Client (Browser)                  │
└──────────────┬───────────────────────────────────────┘
               │ HTTP POST /graphql
               ▼
┌──────────────────────────────────────────────────────┐
│              Interfaces Layer (GraphQL)               │
│  ┌─────────┐  ┌──────────┐  ┌──────────────────┐    │
│  │ Handler │──│ Resolver │──│ Context (IP/Token)│    │
│  └─────────┘  └──────────┘  └──────────────────┘    │
└──────────────┬───────────────────────────────────────┘
               │ Commands / Queries
               ▼
┌──────────────────────────────────────────────────────┐
│              Application Layer (CQRS)                │
│  ┌───────────────────┐  ┌───────────────────────┐    │
│  │  Command Handlers │  │   Query Handlers      │    │
│  │  • RegisterUser   │  │  • GetUserByID        │    │
│  │  • AuthenticateUser│  │  • GetUserSessions    │    │
│  │  • RefreshToken   │  └───────────────────────┘    │
│  │  • RequestReset   │                               │
│  │  • ResetPassword  │                               │
│  └───────────────────┘                               │
└──────────────┬───────────────────────────────────────┘
               │ Domain Interfaces (Ports)
               ▼
┌──────────────────────────────────────────────────────┐
│              Domain Layer (DDD Core)                 │
│  ┌──────────────────────┐ ┌───────────────────────┐  │
│  │ Identity Context     │ │  Auth Context         │  │
│  │ • User Aggregate     │ │  • Session Aggregate  │  │
│  │ • Email, Password,   │ │  • SessionID,         │  │
│  │   UserID, ResetToken │ │    TokenPair          │  │
│  │ • UserRepository     │ │  • SessionRepository  │  │
│  │   (Port)             │ │    (Port)             │  │
│  └──────────────────────┘ └───────────────────────┘  │
└──────────────────────────────────────────────────────┘
               ▲ Implements (Adapters)
               │
┌──────────────────────────────────────────────────────┐
│              Infrastructure Layer                    │
│  ┌──────────┐ ┌──────────┐ ┌───────────┐ ┌───────┐  │
│  │PostgreSQL│ │  Redis   │ │  bcrypt   │ │  JWT  │  │
│  │ Repos    │ │ RateLmt  │ │  Hasher   │ │Issuer │  │
│  └──────────┘ └──────────┘ └───────────┘ └───────┘  │
│  ┌──────────────────┐ ┌──────────────────────────┐   │
│  │ InMemory EventBus│ │ Prometheus + slog Logger │   │
│  └──────────────────┘ └──────────────────────────┘   │
└──────────────────────────────────────────────────────┘
```

### DDD: Domain-Driven Design

The domain is split into **two bounded contexts**:

1. **Identity Context** — owns the concept of a **User**: registration, email validation, password policy, password reset tokens. The User aggregate is the root.

2. **Auth Context** — owns the concept of a **Session**: authentication, token pairs, session lifecycle, revocation. The Session aggregate is the root.

**Why two contexts?** Identity (who you are) and Authentication (proving who you are) are distinct concerns. A User exists even if they've never logged in. A Session exists only during an active authentication. Separating them means we can evolve identity management (e.g., adding OAuth providers) independently from session management (e.g., switching to opaque tokens).

Each context has:
- **Aggregates** — entity clusters with consistency boundaries
- **Value Objects** — immutable types with validation (Email, Password, UserID, ResetToken, SessionID, TokenPair)
- **Domain Events** — facts about what happened (UserRegistered, PasswordResetRequested, etc.)
- **Repository Ports** — interfaces that define what persistence the domain needs (implemented by infrastructure)

### CQRS: Command Query Responsibility Segregation

Commands and queries are explicitly separated:

| Side | Handlers | Purpose |
|---|---|---|
| **Command (Write)** | RegisterUser, AuthenticateUser, RefreshToken, RequestPasswordReset, ResetPassword | Mutate state, enforce invariants, emit events |
| **Query (Read)** | GetUserByID, GetUserSessions | Read current state, return DTOs |

**Why CQRS here?** Auth operations are write-heavy (register, login, refresh) with fundamentally different read patterns (profile lookup). Separating them allows:
- Commands to enforce all business rules and emit domain events
- Queries to return simplified DTOs without loading aggregate complexity
- Future optimization: separate read replicas, different caching strategies

### Layered Architecture

```
Interfaces  →  Application  →  Domain  ←  Infrastructure
(GraphQL)      (CQRS)         (DDD)       (Adapters)
```

**Dependency Rule**: Inner layers never depend on outer layers.
- Domain depends on nothing external
- Application depends on Domain ports (interfaces)
- Infrastructure implements Domain ports
- Interfaces orchestrate Application handlers

---

## Project Structure

```
engineer-challenge/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point: wires everything, starts HTTP server
├── internal/
│   ├── domain/                     # 🧠 Core business logic (no external dependencies)
│   │   ├── identity/               # Bounded Context: Identity
│   │   │   ├── aggregate/
│   │   │   │   └── user.go         # User aggregate root
│   │   │   ├── event/
│   │   │   │   └── events.go       # UserRegistered, PasswordResetRequested/Completed
│   │   │   ├── repository/
│   │   │   │   └── user_repository.go  # Port (interface)
│   │   │   └── valueobject/
│   │   │       ├── email.go        # Validated email (RFC 5321, max 254 chars)
│   │   │       ├── password.go     # Password policy enforcement
│   │   │       ├── user_id.go      # UUID-based typed ID
│   │   │       └── reset_token.go  # Crypto-random, 15min TTL, single-use
│   │   └── auth/                   # Bounded Context: Authentication
│   │       ├── aggregate/
│   │       │   └── session.go      # Session aggregate root
│   │       ├── event/
│   │       │   └── events.go       # UserAuthenticated, SessionRevoked
│   │       ├── repository/
│   │       │   └── session_repository.go  # Port (interface)
│   │       └── valueobject/
│   │           ├── session_id.go   # UUID-based typed ID
│   │           └── token_pair.go   # Access + Refresh token pair
│   ├── application/                # 📋 Use cases (CQRS handlers)
│   │   ├── command/
│   │   │   ├── register_user.go
│   │   │   ├── authenticate_user.go
│   │   │   ├── refresh_token.go
│   │   │   ├── request_password_reset.go
│   │   │   └── reset_password.go
│   │   └── query/
│   │       ├── get_user.go
│   │       └── get_sessions.go
│   ├── infrastructure/             # 🔧 Technical implementations (adapters)
│   │   ├── crypto/
│   │   │   ├── interfaces.go       # PasswordHasher, TokenIssuer interfaces
│   │   │   ├── bcrypt.go           # bcrypt cost=12
│   │   │   └── jwt.go              # HMAC-SHA256 JWT issuer/validator
│   │   ├── eventbus/
│   │   │   └── eventbus.go         # In-memory synchronous event bus
│   │   ├── observability/
│   │   │   └── metrics.go          # Prometheus counters/histograms + slog logger
│   │   └── persistence/
│   │       ├── postgres/
│   │       │   ├── user_repository.go     # PostgreSQL UserRepository adapter
│   │       │   └── session_repository.go  # PostgreSQL SessionRepository adapter
│   │       └── redis/
│   │           └── rate_limiter.go  # Sliding window rate limiter
│   └── interfaces/                 # 🌐 Transport / API layer
│       └── graphql/
│           ├── schema.graphql      # GraphQL SDL schema
│           ├── context.go          # Request context keys (token, IP, user-agent)
│           ├── handler.go          # HTTP handler for /graphql
│           └── resolver.go         # GraphQL resolver → CQRS handlers
├── proto/
│   └── auth.proto                  # gRPC service definition (7 RPCs)
├── migrations/
│   ├── 001_init.up.sql             # CREATE TABLE users, sessions
│   └── 001_init.down.sql           # DROP TABLE
├── docs/
│   ├── README.md                   # ← You are here
│   ├── API.md                      # GraphQL API reference
│   └── adr/                        # Architecture Decision Records
│       ├── 001-go-language-choice.md
│       ├── 002-ddd-bounded-contexts.md
│       ├── 003-cqrs-pattern.md
│       ├── 004-graphql-primary-api.md
│       ├── 005-jwt-hmac-sha256.md
│       ├── 006-bcrypt-password-hashing.md
│       └── 007-redis-rate-limiting.md
├── go.mod
├── go.sum
├── LICENSE
└── README.md                       # Original challenge specification
```

---

## Domain Model

### Identity Bounded Context

The **User** aggregate is the root of the Identity context.

```
User (Aggregate Root)
├── ID: UserID (UUID)
├── Email: Email (validated, normalized, max 254 chars)
├── PasswordHash: string (bcrypt output)
├── Status: UserStatus (active | blocked)
├── ResetToken: *ResetToken (nullable)
├── CreatedAt: time.Time
├── UpdatedAt: time.Time
└── events: []DomainEvent (transient)
```

**Value Objects:**

| Value Object | Validation Rules |
|---|---|
| `Email` | RFC 5321 format via `net/mail`, lowercase normalized, max 254 characters |
| `Password` | Min 8 chars, max 128 chars, requires: 1 uppercase, 1 lowercase, 1 digit, 1 special character. **Never persisted** — only the bcrypt hash is stored. |
| `UserID` | UUID v4, strongly typed (not a bare string) |
| `ResetToken` | 32 crypto-random bytes (hex encoded = 64 chars), 15-minute TTL, single-use flag |

**Aggregate Operations:**

| Method | What It Does | Invariants Enforced |
|---|---|---|
| `Register(email, hash)` | Creates a new User | Email must be valid, hash must be non-empty |
| `RequestPasswordReset()` | Generates a reset token | User must be active |
| `ResetPassword(token, newHash)` | Applies new password | Token must exist, not expired, not used, must match |
| `Reconstruct(...)` | Hydrates from DB (no events emitted) | — |

### Auth Bounded Context

The **Session** aggregate is the root of the Auth context.

```
Session (Aggregate Root)
├── ID: SessionID (UUID)
├── UserID: string
├── RefreshToken: string (JWT)
├── UserAgent: string (browser info)
├── IP: string
├── ExpiresAt: time.Time (7 days from creation)
├── CreatedAt: time.Time
├── RevokedAt: *time.Time (nullable)
└── events: []DomainEvent (transient)
```

**Constants:**

| Constant | Value | Purpose |
|---|---|---|
| `AccessTokenTTL` | 15 minutes | Short-lived access token — limits blast radius of token theft |
| `RefreshTokenTTL` | 7 days | Long-lived refresh token — stored in session, rotated on each use |
| `MaxSessions` | 5 | Maximum concurrent active sessions per user |

**Aggregate Operations:**

| Method | What It Does | Invariants Enforced |
|---|---|---|
| `CreateSession(userID, refreshToken, userAgent, ip)` | Creates a new session | UserID and refreshToken required |
| `Revoke()` | Marks session as revoked | Cannot revoke an already-revoked session |
| `RotateRefreshToken(newToken)` | Replaces refresh token, extends TTL | — |
| `IsActive()` | Returns true if not revoked and not expired | — |

### Domain Events

Events are collected within aggregates and published after successful persistence:

| Event | Emitted When | Data |
|---|---|---|
| `identity.user.registered` | User.Register() | UserID, Email |
| `identity.password_reset.requested` | User.RequestPasswordReset() | UserID, Email |
| `identity.password_reset.completed` | User.ResetPassword() | UserID |
| `auth.user.authenticated` | Session.CreateSession() | SessionID, UserID |
| `auth.session.revoked` | Session.Revoke() | SessionID |

Events follow a `BaseEvent` structure with `EventName()`, `OccurredAt()`, `AggregateID()` — implementing the `DomainEvent` interface.

### Business Invariants & Rules

1. **Email Uniqueness** — enforced at application layer (ExistsByEmail check) and DB level (UNIQUE constraint)
2. **Password Policy** — min 8 chars, requires uppercase + lowercase + digit + special char, max 128 chars
3. **Password Storage** — only bcrypt hash stored (cost=12), plaintext `Password` value object is transient
4. **Reset Token Security** — 32 bytes of `crypto/rand`, hex-encoded, 15-minute TTL, single-use
5. **Max Concurrent Sessions** — 5 per user; when exceeded, the oldest active session is revoked
6. **Token Rotation** — on each refresh, the old refresh token is replaced with a new one (prevents replay)
7. **Anti-Enumeration** — password reset always returns success regardless of whether the email exists
8. **Immutable Value Objects** — Email, Password, UserID, SessionID cannot be modified after creation

---

## Application Layer (CQRS)

### Commands (Write Side)

Each command handler receives a command struct, enforces business rules, mutates aggregates, persists changes, and publishes domain events.

#### 1. RegisterUser

```
Input:  RegisterUserCommand { Email, Password }
Output: RegisterUserResult  { UserID, Email }
```

**Flow:**
1. Validate email format → `NewEmail()`
2. Validate password policy → `NewPassword()`
3. Check email uniqueness → `repo.ExistsByEmail()`
4. Hash password → `hasher.Hash()` (bcrypt cost=12)
5. Create User aggregate → `aggregate.Register()`
6. Persist → `repo.Save()`
7. Publish `UserRegistered` event
8. Return result with user ID

#### 2. AuthenticateUser (Login)

```
Input:  AuthenticateUserCommand { Email, Password, UserAgent, IP }
Output: AuthenticateUserResult  { AccessToken, RefreshToken, UserID, Email }
```

**Flow:**
1. Validate email → `NewEmail()`
2. Find user by email → `repo.FindByEmail()`
3. Check user is active
4. Verify password → `hasher.Verify(plaintext, hash)` (constant-time bcrypt comparison)
5. Check active sessions count → if ≥ 5, revoke oldest
6. Issue access token (15min) → `tokenIssuer.IssueAccessToken()`
7. Issue refresh token (7 days) → `tokenIssuer.IssueRefreshToken()`
8. Create Session aggregate → `aggregate.CreateSession()`
9. Persist session → `sessionRepo.Save()`
10. Publish `UserAuthenticated` event
11. Return tokens + user info

**Security:** On invalid email/password, returns generic "invalid credentials" — no distinction between "user not found" and "wrong password" (prevents user enumeration).

#### 3. RefreshToken

```
Input:  RefreshTokenCommand { RefreshToken }
Output: RefreshTokenResult  { AccessToken, RefreshToken }
```

**Flow:**
1. Find session by refresh token → `sessionRepo.FindByRefreshToken()`
2. Check session is active (not revoked, not expired)
3. Issue new access token
4. Issue new refresh token
5. Rotate: replace old refresh token in session → `session.RotateRefreshToken()`
6. Update session in DB
7. Return new token pair

**Security:** Old refresh token becomes invalid after rotation — prevents replay attacks.

#### 4. RequestPasswordReset

```
Input:  RequestPasswordResetCommand { Email }
Output: RequestPasswordResetResult  { Token }
```

**Flow:**
1. Validate email format
2. Find user by email
3. If user not found → **still return success** (anti-enumeration)
4. Generate reset token → `user.RequestPasswordReset()` (32 crypto-random bytes, 15min TTL)
5. Update user in DB
6. Publish `PasswordResetRequested` event
7. Return token (in production, this would be sent via email, not in the API response)

#### 5. ResetPassword

```
Input:  ResetPasswordCommand { Email, Token, NewPassword }
Output: error (nil on success)
```

**Flow:**
1. Validate email and new password policy
2. Find user by email
3. Hash new password
4. Apply reset → `user.ResetPassword(token, newHash)` — validates token match, TTL, single-use
5. Update user in DB (stores new hash, marks token as used)
6. Publish `PasswordResetCompleted` event

### Queries (Read Side)

#### GetUserByID

```
Input:  GetUserByIDQuery  { UserID }
Output: UserDTO           { ID, Email, Status, CreatedAt }
```

Returns a simplified read-model projection. Does not load reset token or events.

#### GetUserSessions

```
Input:  GetUserSessionsQuery { UserID }
Output: []SessionDTO         { ID, UserID, UserAgent, IP, ExpiresAt, CreatedAt, RevokedAt, IsActive }
```

Returns all active sessions for a user.

---

## Infrastructure Layer

### Password Hashing (bcrypt)

```go
type BcryptHasher struct{}

Hash(password string) → (string, error)   // bcrypt.GenerateFromPassword(cost=12)
Verify(password, hash string) → bool      // bcrypt.CompareHashAndPassword (constant-time)
```

- **Cost factor 12** = approximately 250ms per hash operation on modern hardware
- Constant-time comparison prevents timing attacks
- Adaptive: cost can be increased as hardware improves without data migration

### JWT Tokens

```go
type JWTIssuer struct{ secret, issuer }

IssueAccessToken(userID, email) → (string, error)
IssueRefreshToken(userID) → (string, error)
ValidateAccessToken(token) → (*TokenClaims, error)
```

**Access Token Claims:**
```json
{
  "sub": "user-uuid",
  "email": "user@example.com",
  "iss": "auth-service",
  "iat": 1234567890,
  "exp": 1234568790,
  "type": "access"
}
```

**Refresh Token Claims:**
```json
{
  "sub": "user-uuid",
  "iss": "auth-service",
  "iat": 1234567890,
  "exp": 1235172690,
  "type": "refresh"
}
```

- Signed with **HMAC-SHA256** using a shared secret
- Validation rejects non-HMAC algorithms (prevents `alg: none` attack)
- Token type check prevents using a refresh token as an access token

### PostgreSQL Persistence

Two repository adapters implement the domain ports:

**UserRepository:**
- `Save()` — INSERT with parameterized query (prevents SQL injection)
- `FindByID()` / `FindByEmail()` — SELECT with aggregate reconstruction
- `ExistsByEmail()` — EXISTS check for uniqueness
- `Update()` — UPDATE including reset token fields

**SessionRepository:**
- `Save()` — INSERT new session
- `FindByRefreshToken()` — lookup for token rotation
- `FindActiveByUserID()` — active sessions (not revoked, not expired), ordered by creation (oldest first)
- `Update()` — update refresh token and expiry
- `RevokeAllForUser()` — mass revocation (for password change scenarios)

All queries use **parameterized statements** (`$1, $2...`) — no string concatenation.

### Redis Rate Limiting

```go
type RateLimiter struct{ client *redis.Client }

Allow(ctx, key, limit, window) → (bool, error)
```

**Algorithm: Sliding Window Counter**
1. Remove entries outside the window (`ZREMRANGEBYSCORE`)
2. Count entries in the window (`ZCARD`)
3. Add current request with timestamp as score (`ZADD`)
4. Set TTL on the key (`EXPIRE`)
5. All operations in a single Redis pipeline (1 round trip)

**Rate Limits:**
| Action | Key Pattern | Limit | Window |
|---|---|---|---|
| Registration | `register:global` | 20 requests | 1 minute |
| Login | `login:{IP}` | 10 requests per IP | 1 minute |
| Password Reset | `reset:{email}` | 3 requests per email | 15 minutes |

**Graceful Degradation:** If Redis is unavailable, the service starts normally with rate limiting disabled (logged as a warning).

### Event Bus

```go
type InMemoryEventBus struct{ handlers, logger }

Publish(ctx, event)                    // Synchronous dispatch
Subscribe(eventName, handler)          // Register handler
```

- **Synchronous, in-process** — events are dispatched immediately after persistence
- Each event is logged at INFO level
- In production, this would be replaced with **NATS**, **Kafka**, or **RabbitMQ** for:
  - Asynchronous processing
  - At-least-once delivery guarantees
  - Cross-service event propagation
  - Event replay capability

### Observability

**Prometheus Metrics:**

| Metric | Type | Description |
|---|---|---|
| `auth_registration_total` | Counter | Total registration attempts |
| `auth_registration_errors_total` | Counter | Failed registrations |
| `auth_login_total` | Counter | Total login attempts |
| `auth_login_errors_total` | Counter | Failed logins |
| `auth_password_reset_request_total` | Counter | Password reset requests |
| `auth_token_refresh_total` | Counter | Token refresh operations |
| `auth_request_duration_seconds` | Histogram | Request latency per method |
| `auth_rate_limit_hits_total` | Counter (labels: action) | Rate limit violations |

**Structured Logging (slog):**
- JSON format for machine parsing
- Configurable level (debug/info/warn/error)
- Source location included (`AddSource: true`)
- Key operation logs: registration, authentication, token refresh, password reset, rate limit hits

---

## API Layer (GraphQL)

### Schema

```graphql
type User {
  id: ID!
  email: String!
  status: String!
}

type AuthPayload {
  accessToken: String!
  refreshToken: String!
  user: User!
}

type TokenPair {
  accessToken: String!
  refreshToken: String!
}

type ResetRequestPayload {
  success: Boolean!
  token: String          # Only in dev — production sends via email
}

type Query {
  me: User!              # Requires Authorization: Bearer <accessToken>
}

type Mutation {
  register(email: String!, password: String!): User!
  login(email: String!, password: String!): AuthPayload!
  refreshToken(refreshToken: String!): TokenPair!
  requestPasswordReset(email: String!): ResetRequestPayload!
  resetPassword(email: String!, token: String!, newPassword: String!): Boolean!
  logout(refreshToken: String!): Boolean!
}
```

### Authentication Flow

```
                    Registration
                    ════════════
  Client                                    Server
    │                                         │
    │  mutation register(email, password)      │
    │────────────────────────────────────────►│
    │                                         │ 1. Validate email format
    │                                         │ 2. Enforce password policy
    │                                         │ 3. Check email uniqueness
    │                                         │ 4. Hash password (bcrypt)
    │                                         │ 5. Create User aggregate
    │                                         │ 6. Store in PostgreSQL
    │                                         │ 7. Publish UserRegistered
    │  { id, email, status }                  │
    │◄────────────────────────────────────────│


                    Login
                    ═════
  Client                                    Server
    │                                         │
    │  mutation login(email, password)         │
    │────────────────────────────────────────►│
    │                                         │ 1. Rate limit check (Redis)
    │                                         │ 2. Find user by email
    │                                         │ 3. Verify password (bcrypt)
    │                                         │ 4. Check active sessions ≤ 5
    │                                         │ 5. Issue JWT access token (15min)
    │                                         │ 6. Issue JWT refresh token (7d)
    │                                         │ 7. Create Session aggregate
    │                                         │ 8. Store session
    │  { accessToken, refreshToken, user }    │
    │◄────────────────────────────────────────│


                    Token Refresh
                    ═════════════
  Client                                    Server
    │                                         │
    │  mutation refreshToken(refreshToken)     │
    │────────────────────────────────────────►│
    │                                         │ 1. Find session by refresh token
    │                                         │ 2. Check session active
    │                                         │ 3. Issue new token pair
    │                                         │ 4. Rotate refresh token in session
    │  { accessToken, refreshToken }          │
    │◄────────────────────────────────────────│


                    Password Reset
                    ══════════════
  Client                                    Server
    │                                         │
    │  mutation requestPasswordReset(email)    │
    │────────────────────────────────────────►│
    │                                         │ 1. Rate limit check
    │                                         │ 2. Find user (silent fail if not found)
    │                                         │ 3. Generate reset token (32 bytes)
    │                                         │ 4. Store token with 15min TTL
    │  { success: true, token: "..." }        │
    │◄────────────────────────────────────────│
    │                                         │
    │  mutation resetPassword(email, token,    │
    │                         newPassword)     │
    │────────────────────────────────────────►│
    │                                         │ 1. Validate new password policy
    │                                         │ 2. Find user
    │                                         │ 3. Verify reset token
    │                                         │ 4. Hash new password
    │                                         │ 5. Update user, mark token used
    │  true                                   │
    │◄────────────────────────────────────────│
```

### API Examples

**Register:**
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { register(email: \"user@example.com\", password: \"MyP@ssw0rd!\") { id email status } }"
  }'
```

**Login:**
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { login(email: \"user@example.com\", password: \"MyP@ssw0rd!\") { accessToken refreshToken user { id email } } }"
  }'
```

**Get Current User (authenticated):**
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <accessToken>" \
  -d '{
    "query": "{ me { id email status } }"
  }'
```

**Refresh Token:**
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { refreshToken(refreshToken: \"<refreshToken>\") { accessToken refreshToken } }"
  }'
```

**Request Password Reset:**
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { requestPasswordReset(email: \"user@example.com\") { success token } }"
  }'
```

**Reset Password:**
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { resetPassword(email: \"user@example.com\", token: \"<resetToken>\", newPassword: \"NewP@ssw0rd!\") }"
  }'
```

---

## gRPC Protocol (Proto Definition)

The `proto/auth.proto` file defines a gRPC service with 7 RPCs, ready for code generation:

```protobuf
service AuthService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Authenticate(AuthenticateRequest) returns (AuthenticateResponse);
  rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse);
  rpc RequestPasswordReset(RequestPasswordResetRequest) returns (RequestPasswordResetResponse);
  rpc ResetPassword(ResetPasswordRequest) returns (ResetPasswordResponse);
  rpc GetMe(GetMeRequest) returns (GetMeResponse);
  rpc Logout(LogoutRequest) returns (LogoutResponse);
}
```

This shares the same application handlers as GraphQL. In a microservice setup, internal services would use gRPC for low-latency communication while client-facing traffic goes through GraphQL.

---

## Database Schema

### Users Table

```sql
CREATE TABLE users (
    id                     UUID PRIMARY KEY,
    email                  VARCHAR(254) NOT NULL UNIQUE,
    password_hash          TEXT NOT NULL,
    status                 VARCHAR(20) NOT NULL DEFAULT 'active',
    reset_token            TEXT,
    reset_token_expires_at TIMESTAMPTZ,
    reset_token_used       BOOLEAN,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

### Sessions Table

```sql
CREATE TABLE sessions (
    id            UUID PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token TEXT NOT NULL,
    user_agent    TEXT,
    ip            VARCHAR(45),
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at    TIMESTAMPTZ
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token);
CREATE INDEX idx_sessions_active ON sessions(user_id, revoked_at, expires_at);
```

**Design notes:**
- `ON DELETE CASCADE` on sessions — deleting a user cascades to their sessions
- `VARCHAR(45)` for IP — supports both IPv4 and IPv6
- `TIMESTAMPTZ` everywhere — timezone-aware timestamps prevent time zone bugs
- Composite index `idx_sessions_active` — optimizes the "find active sessions by user" query
- Reset token fields are nullable — only populated during an active reset flow

---

## Security

### Threat Model & Mitigations

| Threat | Mitigation |
|---|---|
| **Brute force password** | bcrypt cost=12 (~250ms/hash), Redis rate limiting (10/min per IP on login) |
| **User enumeration** | Generic "invalid credentials" error on login, silent success on password reset for unknown emails |
| **SQL injection** | All queries use parameterized statements ($1, $2...) |
| **JWT algorithm confusion** | Explicit HMAC check in validation — rejects non-HMAC algorithms |
| **Token replay** | Refresh token rotation — old token invalidated on each refresh |
| **Session hijacking** | Short access token TTL (15min), sessions track IP + User-Agent |
| **Password in logs** | Password value object is transient, only hash persisted. Structured logging does not log password fields. |
| **Reset token guessing** | 32 bytes of `crypto/rand` (256 bits of entropy), 15-minute TTL, single-use |
| **Excessive sessions** | Max 5 concurrent sessions; oldest revoked when limit exceeded |
| **DDoS on registration** | Global rate limit: 20 registrations/minute |
| **Reset abuse** | 3 reset requests per email per 15 minutes |
| **XSS via GraphQL** | JSON responses only, Content-Type: application/json |
| **CORS** | Configurable origins (currently permissive for development) |

### Password Policy

| Rule | Requirement |
|---|---|
| Minimum length | 8 characters |
| Maximum length | 128 characters |
| Uppercase letter | At least 1 (`A-Z`) |
| Lowercase letter | At least 1 (`a-z`) |
| Digit | At least 1 (`0-9`) |
| Special character | At least 1 (punctuation or symbol) |

---

## How to Run

### Prerequisites

- Go 1.21+
- PostgreSQL 14+
- Redis 7+ (optional — rate limiting degrades gracefully)

### Quick Start

1. **Start PostgreSQL and Redis:**
   ```bash
   # Using Docker (if you have it):
   docker run -d --name auth-postgres -p 5432:5432 \
     -e POSTGRES_USER=auth -e POSTGRES_PASSWORD=auth -e POSTGRES_DB=auth \
     postgres:16-alpine

   docker run -d --name auth-redis -p 6379:6379 redis:7-alpine
   ```

2. **Run migrations:**
   ```bash
   psql -h localhost -U auth -d auth -f migrations/001_init.up.sql
   ```

3. **Start the server:**
   ```bash
   export DATABASE_URL="postgres://auth:auth@localhost:5432/auth?sslmode=disable"
   export JWT_SECRET="your-32-char-secret-key-here!!!" 
   go run cmd/server/main.go
   ```

4. **Verify:**
   ```bash
   curl http://localhost:8080/health
   # → {"status":"ok"}
   ```

### Endpoints

| Path | Purpose |
|---|---|
| `POST /graphql` | GraphQL API (all auth operations) |
| `GET /metrics` | Prometheus metrics |
| `GET /health` | Health check |

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | `postgres://auth:auth@localhost:5432/auth?sslmode=disable` | PostgreSQL connection string |
| `REDIS_URL` | `localhost:6379` | Redis address |
| `REDIS_PASSWORD` | ` ` (empty) | Redis password |
| `JWT_SECRET` | `change-me-in-production-32-chars!` | HMAC-SHA256 signing secret (**must change in production**) |
| `JWT_ISSUER` | `auth-service` | JWT issuer claim |
| `LOG_LEVEL` | `info` | Log level: debug, info, warn, error |

---

## Testing

### Running Tests
```bash
go test ./... -v
```

### Test Coverage
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### What to Test

| Layer | What to Test | Priority |
|---|---|---|
| **Domain** | Value object validation (email, password, reset token), aggregate invariants (register, reset, max sessions) | Critical |
| **Application** | Command handler flows (happy path + edge cases), anti-enumeration on reset | Critical |
| **Infrastructure** | Repository queries (integration with test DB), JWT issuance/validation, bcrypt round-trip | High |
| **Integration** | Full register → login → refresh → reset flow via GraphQL | High |

---

## Trade-offs & Design Decisions

### 1. In-Memory Event Bus vs. Message Broker
**Decision:** Synchronous in-memory event bus.
**Trade-off:** Simpler to implement and debug, but events are lost on crash, cannot scale to multiple instances, and no replay. For a single-service auth module, this is acceptable. The interface (`EventBus`) allows swapping in NATS/Kafka later without changing application code.

### 2. GraphQL over REST
**Decision:** GraphQL as primary client API.
**Trade-off:** Slightly more complex server-side setup (schema definition, resolver wiring) but provides: typed schema, single endpoint, no API versioning, client-driven queries. For auth flows where different clients need different data shapes, this pays off.

### 3. JWT Access + Refresh Token Pattern
**Decision:** Short-lived access tokens (15min) + long-lived refresh tokens (7 days) with rotation.
**Trade-off:** More complex than simple session cookies, but stateless access verification (no DB lookup per request), explicit token lifecycle, and cross-domain support. Refresh rotation mitigates token theft.

### 4. Two Bounded Contexts (Identity + Auth)
**Decision:** Separate Identity (who you are) from Authentication (proving identity).
**Trade-off:** More packages and files, but cleaner evolution. Adding OAuth providers only touches Identity. Switching from JWT to opaque tokens only touches Auth. Neither context needs to know about the other's internals.

### 5. Reset Token in API Response (Development)
**Decision:** Return reset token directly in the GraphQL response.
**Trade-off:** **Not production-safe.** In production, the token would be sent via email. Done this way for development/testing convenience — the `RequestPasswordResetResult` comment documents this explicitly.

### 6. HMAC-SHA256 vs. Asymmetric JWT
**Decision:** Symmetric signing (shared secret).
**Trade-off:** Simpler key management (one secret), but any service that can validate tokens can also forge them. Acceptable for a single-service deployment. For microservices, switch to RS256/ES256 so validators don't need the signing key.

### 7. bcrypt Cost=12
**Decision:** Cost factor 12 for password hashing.
**Trade-off:** ~250ms per hash. High enough to resist brute force (4 attempts/second per CPU core), low enough for acceptable login latency. Can be increased to 13 or 14 as hardware improves. Argon2id would be theoretically stronger but bcrypt is more widely supported.

---

## What I Would Do Next (Production Roadmap)

### Immediate (Before First Deploy)
1. **Docker Compose** — full environment with PostgreSQL, Redis, app container, migrations
2. **Terraform/K8s manifests** — IaC for cloud deployment
3. **Unit + Integration tests** — domain invariants, auth flow, SQL queries
4. **Frontend** — React auth forms matching the Figma design
5. **Email delivery** — remove reset token from API response, send via SMTP/SES
6. **Secrets management** — use Vault/AWS Secrets Manager instead of env vars

### Short-Term (Production Hardening)
7. **HTTPS/TLS termination** — via reverse proxy (nginx/Caddy) or cloud LB
8. **CORS configuration** — restrict to actual frontend domain
9. **Account lockout** — temporary lock after N failed login attempts
10. **Audit log** — persist all auth events for security review
11. **gRPC server** — implement the proto service for internal communication
12. **Database migrations tool** — golang-migrate or Atlas for version-controlled schema changes

### Medium-Term (Scale & Features)
13. **OAuth2 providers** — Google, GitHub login (new Identity adapters)
14. **MFA/2FA** — TOTP or WebAuthn support
15. **Distributed event bus** — NATS/Kafka for cross-service events
16. **Read replicas** — separate DB for query handlers
17. **Session management UI** — users can see and revoke their sessions
18. **IP geolocation** — suspicious login detection
19. **OpenTelemetry tracing** — distributed traces across services
