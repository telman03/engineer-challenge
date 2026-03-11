# ADR-004: GraphQL as Primary Client API

## Status
Accepted

## Date
2026-03-06

## Context

The challenge requires gRPC and/or GraphQL, with REST being acceptable only with strong architectural argumentation. We need to choose the primary transport for client-facing auth operations.

## Decision

**GraphQL** is the primary client-facing API. A **gRPC protocol definition** is provided for future internal service-to-service communication.

## Rationale

### Why GraphQL for Client-Facing Auth?

1. **Single endpoint** — `POST /graphql` handles all auth operations. No need to coordinate multiple REST endpoints (`/register`, `/login`, `/reset`, `/refresh`) with different HTTP methods and status codes.

2. **Typed schema, self-documenting** — the GraphQL schema (`schema.graphql`) serves as both documentation and contract. Clients know the exact shape of requests and responses without external tooling.

3. **Client-driven responses** — auth flows have different data needs:
   - Registration only needs `{ id, email, status }`
   - Login needs `{ accessToken, refreshToken, user { id, email } }`
   - A frontend can request exactly what it needs without over-fetching.

4. **Error handling** — GraphQL's standardized `errors` array is cleaner than mapping auth errors to HTTP status codes (is "invalid credentials" a 401 or 400? Are rate limit errors 429 or 200 with error body?).

5. **Introspection** — dev tools (GraphiQL, Apollo Studio) can introspect the running server for schema discovery. No OpenAPI spec maintenance needed.

### Why gRPC in Addition (Proto Definition)?

The `proto/auth.proto` file defines all 7 auth RPCs. This enables:
- **Internal microservice calls** — other services (e.g., API gateway, user profile service) can validate tokens or fetch user data via efficient binary protocol.
- **Code generation** — Go, TypeScript, Python clients can be generated from the proto.
- **Future evolution** — when the auth module becomes a standalone service, gRPC is the natural internal transport.

The proto is defined but not yet implemented as a running server. The GraphQL resolver and a future gRPC server would share the same application layer handlers.

### Why Not REST?

| REST Concern | Our Solution |
|---|---|
| Multiple endpoints to maintain | Single `/graphql` endpoint |
| HTTP status code ambiguity for auth errors | GraphQL error array with consistent structure |
| API versioning (`/v1/`, `/v2/`) | GraphQL schema evolution (add fields, deprecate old ones) |
| Manual OpenAPI documentation | Introspectable schema |
| Request/response shape negotiation | Client specifies exact fields needed |

REST would be perfectly functional for auth, but GraphQL provides a better developer experience for the frontend integration required by this challenge.

## Implementation

```
POST /graphql     — GraphQL API (all auth operations)
GET  /metrics     — Prometheus metrics (not GraphQL)
GET  /health      — Health check endpoint
```

The GraphQL layer is thin:
- `handler.go` — HTTP handler: parses JSON body, extracts auth token from `Authorization: Bearer` header, injects IP/User-Agent into context
- `resolver.go` — maps GraphQL mutations/queries to CQRS command/query handlers
- `context.go` — typed context keys for request metadata

## Alternatives Considered

| Approach | Why Not |
|---|---|
| REST only | Explicitly discouraged by challenge. Would require more endpoints, status code management, and API versioning. |
| gRPC only | gRPC is not browser-native. Requires gRPC-Web proxy or translation layer for frontend. GraphQL is natively consumed by React/Apollo. |
| REST + gRPC | More API surface to maintain. GraphQL covers what REST would provide, plus gRPC for internal. |
| tRPC | TypeScript-specific. Go backend doesn't benefit from tRPC's type inference. |

## Consequences

- **Positive:** Single endpoint for all operations, typed schema, great DX for frontend, schema-as-documentation.
- **Negative:** More complex server-side setup than REST handlers, no built-in content negotiation or response caching (though auth responses shouldn't be cached), `graphql-go` library is less actively maintained than some alternatives.
- **Mitigation:** The resolver is a thin translation layer — all logic is in application handlers. Switching GraphQL libraries (e.g., to `gqlgen`) would only require rewriting the resolver, not the business logic.
