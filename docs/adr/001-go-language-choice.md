# ADR-001: Go as Primary Language

## Status
Accepted

## Date
2026-03-06

## Context

We need to choose a programming language for building an authentication module that must demonstrate engineering maturity: DDD, CQRS, gRPC/GraphQL support, Infrastructure as Code, security best practices, and observability.

The language must support:
- Strong type system for domain modeling (value objects, aggregates)
- Efficient bcrypt/crypto operations
- Native gRPC and GraphQL support
- Production-grade HTTP server
- Structured logging and Prometheus metrics
- Easy deployment

## Decision

We chose **Go (version 1.25)** as the primary language.

## Rationale

1. **Static typing for DDD** — Go's type system enables strongly-typed value objects (`Email`, `UserID`, `Password`) and aggregate roots with clear boundaries. While it lacks generics in older versions, the struct + interface model maps well to DDD tactical patterns.

2. **Performance & concurrency** — Go's goroutine model handles concurrent auth requests efficiently. bcrypt hashing is CPU-bound; Go's M:N scheduler distributes this across OS threads without manual thread pool management.

3. **Standard library strength** — `net/http`, `crypto`, `encoding/json`, `database/sql`, `log/slog` are all production-grade. Minimal third-party dependencies reduce supply chain risk — critical for an auth service.

4. **Single binary deployment** — `go build` produces a statically-linked binary. No runtime deps, no JVM startup, no npm install. Ideal for containerized deployment.

5. **Ecosystem** — Mature libraries for JWT (`golang-jwt`), PostgreSQL (`lib/pq`), Redis (`go-redis`), gRPC (`google.golang.org/grpc`), GraphQL (`graphql-go`), Prometheus (`prometheus/client_golang`).

6. **Simplicity** — Go's explicit error handling and minimal language surface area make auth code easy to audit. No hidden exceptions, no magic annotations, no reflection-heavy frameworks.

## Alternatives Considered

| Language | Why Not |
|---|---|
| TypeScript/Node.js | Single-threaded event loop adds complexity for CPU-bound bcrypt. Less type safety than Go for domain modeling. npm dependency surface area is a security concern for auth. |
| Rust | Excellent for systems programming and type safety, but development velocity is significantly slower (borrow checker, steeper learning curve). Overkill for this scope. |
| Java/Kotlin | JVM startup time, memory overhead, annotation-heavy frameworks (Spring Security) conflict with the "show your architecture, not a framework" requirement. |
| Python | Dynamic typing makes domain invariant enforcement harder. Performance concerns for bcrypt at scale. |

## Consequences

- **Positive:** Fast development, easy deployment, strong crypto/network libraries, readable code for review.
- **Negative:** No sum types (need to encode with constants + validation), boilerplate for simple operations, no built-in dependency injection (manual wiring in `main.go`).
- **Mitigation:** Use value objects with private fields + constructors to enforce invariants. Accept manual wiring as it makes dependencies explicit.
