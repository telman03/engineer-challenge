# Full Workspace Guide (Easy, Detailed)

This guide explains the entire `engineer-challenge` workspace in plain language.

If you are new to this project, read this file first.

## 1. What This Project Does

This is an authentication system with:

1. User registration
2. User login
3. JWT access/refresh tokens
4. Password reset flow
5. A simple React frontend
6. Local and cloud deployment configs

The backend is written in Go and follows:

- DDD (Domain-Driven Design)
- CQRS (Command Query Responsibility Segregation)

## 2. Big Picture Architecture

There are four main backend layers:

1. `interfaces` layer
What comes in from the outside world (GraphQL HTTP API).

2. `application` layer
Use-case handlers (commands and queries).

3. `domain` layer
Core business rules and domain models.

4. `infrastructure` layer
Technical implementations (Postgres, Redis, JWT, bcrypt, metrics, event bus).

Data flow for a typical request:

1. Frontend calls `/graphql`
2. GraphQL resolver receives request
3. Resolver calls a command/query handler
4. Handler validates business rules via domain objects
5. Handler uses repositories/services implemented in infrastructure
6. Result is returned to GraphQL response

## 3. Workspace Map (Top-Level)

### `cmd/`
Contains program entrypoints.

- `cmd/server/main.go`
  Starts the HTTP server, wires dependencies, and mounts `/graphql`, `/health`, `/metrics`.

### `internal/`
Main backend source code.

- `internal/domain/`
  Pure business logic. No DB/HTTP framework details.
- `internal/application/`
  Command/query handlers that orchestrate use cases.
- `internal/infrastructure/`
  DB repos, Redis limiter, JWT, bcrypt, event bus, observability.
- `internal/interfaces/`
  GraphQL schema, resolver, and HTTP handler.

### `migrations/`
SQL migrations for database schema.

- `001_init.up.sql` creates `users` and `sessions` tables.

### `proto/`
`auth.proto` defines gRPC contracts for future service-to-service integration.

### `web/`
React + TypeScript frontend (Vite).

### `deployments/`
Infrastructure/deployment assets:

- `terraform/` for AWS infrastructure
- `k8s/` for Kubernetes manifests
- `prometheus/` and `grafana/` for monitoring

### `docs/`
Project documentation, API reference, ADRs, and this full guide.

### `.agents/`
Documentation about AI-assisted development process.

### Root config files

- `docker-compose.yml` local stack orchestration
- `Dockerfile` backend image build
- `.env.example` environment variables
- `go.mod` and `go.sum` Go dependencies

## 4. Backend Domain Model (DDD)

The domain is split into two bounded contexts.

### Identity Context (`internal/domain/identity`)
Responsible for user identity rules.

Core pieces:

- Aggregate: `User`
- Value objects: `Email`, `Password`, `UserID`, `ResetToken`
- Domain events: user registered, password reset requested/completed
- Repository port: `UserRepository`

Important rules enforced here:

1. Email is normalized and validated.
2. Password has complexity requirements.
3. Reset token has TTL and single-use behavior.

### Auth Context (`internal/domain/auth`)
Responsible for session/authentication rules.

Core pieces:

- Aggregate: `Session`
- Value objects: `SessionID`, `TokenPair`
- Domain events: user authenticated, session revoked
- Repository port: `SessionRepository`

Important rules enforced here:

1. Session expiration and revocation state are tracked.
2. Token lifecycle is tied to session lifecycle.

## 5. Application Layer (CQRS)

Location: `internal/application/`

### Commands (`internal/application/command/`)
Commands change system state.

- `register_user.go`
  Creates user with hashed password.

- `authenticate_user.go`
  Validates credentials, creates session, returns tokens.

- `refresh_token.go`
  Rotates refresh token and returns a new token pair.

- `request_password_reset.go`
  Creates reset token.

- `reset_password.go`
  Verifies reset token and updates password hash.

### Queries (`internal/application/query/`)
Queries read state.

- `get_user.go`
  Fetches current user profile.

- `get_sessions.go`
  Fetches user sessions.

Why split commands and queries:

- Clear separation of write logic vs read logic
- Easier maintenance and future scaling

## 6. Infrastructure Layer (Adapters)

Location: `internal/infrastructure/`

### `crypto/`
- `bcrypt.go`: password hash/verify
- `jwt.go`: issue and validate JWT access/refresh tokens

### `persistence/postgres/`
- `user_repository.go`: DB operations for users
- `session_repository.go`: DB operations for sessions

### `persistence/redis/`
- `rate_limiter.go`: sliding-window rate limiting

### `eventbus/`
- `eventbus.go`: in-memory synchronous domain event bus

### `observability/`
- `metrics.go`: Prometheus metrics and structured logging support

## 7. API Layer (GraphQL)

Location: `internal/interfaces/graphql/`

Main files:

- `schema.graphql` GraphQL schema
- `resolver.go` maps fields/mutations to command/query handlers
- `handler.go` HTTP GraphQL handler
- `context.go` pulls request context values (token, IP, user-agent)

### Main GraphQL Operations

Queries:

- `me`

Mutations:

- `register(email, password)`
- `login(email, password)`
- `refreshToken(refreshToken)`
- `requestPasswordReset(email)`
- `resetPassword(email, token, newPassword)`
- `logout(refreshToken)`

Note:

- `logout` currently returns `true` in resolver and is a placeholder behavior.

## 8. Database Model

Defined in `migrations/001_init.up.sql`.

### `users`
Fields include:

- `id` UUID (PK)
- `email` unique
- `password_hash`
- `status`
- reset token fields
- timestamps

### `sessions`
Fields include:

- `id` UUID (PK)
- `user_id` FK to users
- `refresh_token`
- `user_agent`
- `ip`
- `expires_at`
- `revoked_at`

Indexes exist for common lookups (email, refresh token, active sessions).

## 9. Frontend (React)

Location: `web/`

Purpose:

- Provide simple UI for all auth flows

Key files:

- `web/src/App.tsx`
  Selects active auth screen by current state.

- `web/src/hooks/useAuth.ts`
  Central auth state and handlers (login/register/forgot/reset/logout).

- `web/src/api/graphql.ts`
  Fetch wrapper and GraphQL operations.

- `web/src/components/*`
  Forms and dashboard components.

Important frontend behavior:

1. Registration calls `register`, then auto-login to obtain tokens.
2. Tokens are stored in `localStorage`.
3. Forgot-password flow can display reset token from API (dev-friendly behavior).
4. Reset-password requires email + token + new password.

## 10. Local Development Setup

### Backend + dependencies via Docker Compose

From repository root:

```bash
docker compose up --build
```

Main endpoints:

- GraphQL: `http://localhost:8080/graphql`
- Health: `http://localhost:8080/health`
- Metrics: `http://localhost:8080/metrics`
- Prometheus UI: `http://localhost:9090`
- Grafana: `http://localhost:3001`

### Frontend

```bash
cd web
npm install
npm run dev
```

Frontend URL:

- `http://localhost:3000`

## 11. Environment Variables

From `.env.example`:

- `PORT`
- `DATABASE_URL`
- `REDIS_URL`
- `REDIS_PASSWORD`
- `JWT_SECRET`
- `JWT_ISSUER`
- `LOG_LEVEL`

For production:

1. Use a long random `JWT_SECRET`
2. Store secrets in a secret manager
3. Avoid default credentials

## 12. Testing

Current Go tests are in domain and infrastructure packages.

Run:

```bash
go test ./internal/... -v
```

Areas already covered:

- Value object validation
- Aggregate behavior/invariants
- JWT and bcrypt helpers
- In-memory event bus

## 13. Deployment Assets

### Docker

- `Dockerfile` uses multi-stage build and runs non-root user.

### Docker Compose

`docker-compose.yml` includes:

1. `auth-server`
2. `postgres`
3. `redis`
4. `migrate`
5. `prometheus`
6. `grafana`

### Terraform (`deployments/terraform/`)

Defines AWS-targeted infrastructure components such as VPC/network, DB/cache, and runtime resources.

### Kubernetes (`deployments/k8s/`)

Provides a deployment with:

- 2 replicas
- liveness/readiness probes
- resource requests/limits
- service and config map

## 14. Security Model (Practical View)

Implemented protections:

1. Password hashing with bcrypt
2. Access + refresh token model
3. Rate limiting with Redis
4. Session tracking and revocation support
5. Password reset token workflow
6. Structured logs and metrics for monitoring

Gaps to consider for production hardening:

1. Fully implemented logout invalidation behavior
2. Stronger secret management integration
3. MFA support
4. Audit trail persistence and alerting

## 15. Request Lifecycle Example (Login)

1. Frontend calls GraphQL `login` mutation.
2. Resolver receives email/password plus IP/user-agent context.
3. Resolver applies rate limit check in Redis.
4. `AuthenticateUserHandler` verifies user credentials via bcrypt.
5. Session is created/stored in Postgres.
6. JWT access and refresh tokens are issued.
7. GraphQL response returns token pair + user info.
8. Frontend stores tokens in `localStorage` and switches to dashboard.

## 16. Workspace Health Notes

Current codebase includes:

- Backend service implementation
- GraphQL API
- Frontend UI
- Monitoring setup
- IaC artifacts
- Architectural documentation + ADRs

This means the workspace is organized for both:

1. Feature development (code + tests)
2. Deployment/operations (Docker/K8s/Terraform/monitoring)

## 17. File-by-File Starting Points

If you want to start quickly, open these in order:

1. `cmd/server/main.go`
2. `internal/interfaces/graphql/resolver.go`
3. `internal/application/command/authenticate_user.go`
4. `internal/domain/identity/aggregate/user.go`
5. `internal/domain/auth/aggregate/session.go`
6. `web/src/hooks/useAuth.ts`
7. `web/src/api/graphql.ts`
8. `docker-compose.yml`

## 18. Summary

This repository is a full-stack authentication module with clear layering and strong engineering structure.

- Domain rules are explicit and separated from infrastructure.
- Command and query responsibilities are cleanly separated.
- API is exposed via GraphQL with a typed schema.
- Frontend provides a complete auth UX.
- Deployment and observability assets are included.

For day-to-day work, this guide plus `docs/API.md` is enough to be productive quickly.
