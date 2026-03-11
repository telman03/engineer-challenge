# ADR-007: Redis Sliding Window Rate Limiting

## Status
Accepted

## Date
2026-03-06

## Context

Authentication endpoints are prime targets for brute force, credential stuffing, and abuse. We need rate limiting that is:
- Fast (sub-millisecond decision)
- Distributed-ready (shared state across service instances)
- Accurate (sliding window, not fixed window)
- Fail-open (service degrades gracefully if rate limiter is unavailable)

## Decision

We implemented **sliding window rate limiting** using **Redis sorted sets**, with different limits per operation type.

## Rationale

### Algorithm: Sorted Set Sliding Window

```
Pipeline (1 round-trip to Redis):
1. ZREMRANGEBYSCORE key -inf (now - window)    # Remove expired entries
2. ZCARD key                                    # Count entries in window
3. ZADD key (now, now)                         # Add current request
4. EXPIRE key window                           # Set key TTL
```

Each request is stored as a sorted set member with its timestamp as the score. This provides an accurate sliding window — unlike fixed-window counters that can allow 2x the limit at window boundaries.

### Rate Limits by Operation

| Operation | Key Pattern | Limit | Window | Rationale |
|---|---|---|---|---|
| Registration | `register:global` | 20/min | 1 minute | Prevents mass account creation. Global limit (not per-IP) because legitimate registration is low-volume. |
| Login | `login:{IP}` | 10/min | 1 minute | Per-IP to stop credential stuffing. 10/min allows legitimate users who mistype passwords while slowing attackers. |
| Password Reset | `reset:{email}` | 3/15min | 15 minutes | Per-email to prevent mailbox flooding. Aligned with reset token TTL (15 min). |

### Why Redis?

| Requirement | Redis Capability |
|---|---|
| Sub-millisecond latency | In-memory data store, single-digit microsecond operations |
| Distributed state | Multiple service instances share the same Redis, ensuring global rate limits |
| TTL management | Native key expiration, no cleanup needed |
| Atomic operations | Pipeline ensures count + add + expire happen atomically |
| Operational simplicity | Single binary, simple protocol, wide hosting support |

### Graceful Degradation

```go
if err := rdb.Ping(context.Background()).Err(); err != nil {
    logger.Warn("redis not available, rate limiting disabled", "error", err)
}
```

If Redis is unavailable at startup, the service runs without rate limiting. This is a deliberate trade-off: availability over security in development/staging, with the expectation that Redis is always available in production (monitored with alerts).

In the rate limiter itself, failures are absorbed:
```go
if r.rateLimiter != nil {
    allowed, _ := r.rateLimiter.Allow(...)
    if !allowed { return error }
}
```

The `nil` check means rate limiting is only applied when configured. Redis errors during operation fall through (fail-open).

### Monitoring

Rate limit hits are tracked via Prometheus:
```
auth_rate_limit_hits_total{action="login"}
auth_rate_limit_hits_total{action="register"}
auth_rate_limit_hits_total{action="reset"}
```

This enables alerting on unusual spikes (potential attack detection).

## Alternatives Considered

| Approach | Why Not |
|---|---|
| In-memory rate limiter (e.g., `golang.org/x/time/rate`) | Not shared across instances. Each server has independent counts. Attacker can distribute requests. |
| Fixed-window counters | Allows burst of 2x the limit at window boundaries (e.g., 10 requests at 0:59, 10 at 1:01). Sliding window is more accurate. |
| Token bucket | More complex to implement in Redis. Sliding window is simpler and provides equivalent protection for our use cases. |
| WAF/CDN rate limiting (e.g., Cloudflare) | Good for production but out of scope for local development. Application-level rate limiting is defense-in-depth. |
| PostgreSQL for counters | Too slow for per-request checks. DB round-trip adds 1-5ms vs Redis <0.5ms. |

## Consequences

- **Positive:** Accurate sliding window, shared state for horizontal scaling, fast decisions, Prometheus observability, graceful degradation.
- **Negative:** Additional infrastructure dependency (Redis), fail-open means temporary vulnerability during Redis outages, sorted set memory usage grows with request volume.
- **Mitigation:** Redis monitoring + alerts, sorted set entries self-expire (ZREMRANGEBYSCORE + EXPIRE), consider switching to fail-closed in production with a circuit breaker.
