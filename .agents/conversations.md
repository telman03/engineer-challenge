# Dev Notes — Key Sessions

Quick notes on the main development sessions and what came out of each.

## Session 1 — Domain Layer

Started by figuring out the bounded contexts. Auth felt like it needed to be split into two concerns: user identity (registration, passwords) and session management (login, tokens). Ended up with an Identity context and an Auth context, each with their own aggregate.

Spent some time getting the value objects right — Email does normalization and RFC validation, Password enforces complexity rules, ResetToken handles the 15-min expiry logic. The AI helped implement these once I had the rules defined, but I had to be specific about what "valid" meant for each one.

Merged this as PR #1.

## Session 2 — Infrastructure & Application

This was the big session. Wired up:
- bcrypt hasher (cost=12 — good enough balance for auth)
- JWT issuer with separate access/refresh tokens
- PostgreSQL repos for users and sessions
- Redis-based rate limiter (sliding window)
- In-memory event bus (quick and dirty, noted as a production TODO)

The anti-enumeration pattern on password reset was an interesting one — always return success even if the email doesn't exist. Simple but effective.

CQRS handlers came together pretty quickly since the domain layer was solid.

## Session 3 — Deployment & Docs

Set up the full deployment story:
- Multi-stage Docker build (keeps the image small, runs as non-root)
- Docker Compose with Postgres, Redis, Prometheus, and Grafana
- Terraform for AWS (VPC, RDS, ElastiCache, ECS Fargate) — not deployed, but the manifests are there
- Basic K8s deployment with health checks

Also wrote the ADRs. Seven of them covering the major decisions (DDD structure, CQRS approach, GraphQL choice, etc.).

## Session 4 — Tests & Frontend

Wrote unit tests for all domain value objects and aggregate invariants. Also added infrastructure tests for bcrypt round-trips, JWT lifecycle, and event bus routing.

Frontend is a simple React + TypeScript + Vite setup. Login, register, forgot password, reset password flows. Has a password strength indicator that checks against the same rules as the backend value object. GraphQL client auto-injects the Bearer token.

Nothing fancy, but it works end to end.
