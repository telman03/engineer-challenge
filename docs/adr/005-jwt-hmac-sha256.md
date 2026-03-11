# ADR-005: JWT with HMAC-SHA256

## Status
Accepted

## Date
2026-03-06

## Context

We need a token format for access and refresh tokens in the authentication flow. The token must be verifiable without a database lookup (for access tokens), support expiration, and carry user identity claims.

## Decision

We chose **JWT (JSON Web Tokens)** signed with **HMAC-SHA256** (HS256), implementing a dual-token pattern:
- **Access token**: 15-minute TTL, carries user ID + email, used for API authorization
- **Refresh token**: 7-day TTL, carries user ID only, stored in a database session record, rotated on each use

## Rationale

### JWT over Opaque Tokens

| JWT | Opaque Token |
|---|---|
| Self-contained: verifiable without DB lookup | Requires DB/cache lookup on every request |
| Carries claims (user ID, email, role) | Server must look up claims separately |
| Larger size (~200-400 bytes) | Small (32 bytes typically) |
| Cannot be revoked instantly (until expiry) | Instantly revocable |

For **access tokens**: JWT is ideal. The `me` query verifies the token cryptographically without touching the database. With 15-minute TTL, the revocation gap is acceptable.

For **refresh tokens**: We use JWT format but verify against the database session (checking that the session is active and the refresh token matches). This gives us the best of both worlds — structured claims + server-side validation.

### HMAC-SHA256 over Asymmetric Signing

| HMAC-SHA256 (HS256) | RSA/ECDSA (RS256/ES256) |
|---|---|
| Same key for signing and verifying | Separate keys (private signs, public verifies) |
| Simpler key management (one secret) | Requires PKI, key rotation, distribution |
| Fast (symmetric crypto) | Slower (asymmetric crypto) |
| Any holder of the key can forge tokens | Only private key holder can forge |

For a **single-service deployment**, HS256 is appropriate — the signing service **is** the verifying service. There's no need to distribute public keys.

**Migration path**: If the auth module becomes a standalone service (other services need to validate tokens), we'd switch to RS256/ES256 so downstream services can verify without having the signing key.

### Token Lifecycle

```
Login → Issue Access (15min) + Refresh (7d)
        ↓
        Use Access for API calls
        ↓
        Access expires (15min)
        ↓
Refresh → Issue new Access + new Refresh (rotation)
          Old refresh token invalidated
        ↓
        ... repeat until Refresh expires (7d) ...
        ↓
        Must re-login
```

### Security Controls

1. **Algorithm validation**: Token verification explicitly checks for HMAC signing method. This prevents the classic `alg: none` attack where an attacker modifies the JWT header.

2. **Token type discrimination**: Access tokens include `"type": "access"`, refresh tokens include `"type": "refresh"`. Validation rejects a refresh token used as an access token.

3. **Refresh token rotation**: Every refresh operation replaces the old refresh token with a new one in the database. An attacker who steals a refresh token gets one use — the legitimate user's next refresh will fail, triggering re-authentication.

4. **Session binding**: Refresh tokens are stored in the sessions table. A Session can be revoked by the user or by the max-sessions enforcement, invalidating the associated refresh token.

## Token Claims

**Access Token:**
```json
{
  "sub": "uuid-of-user",
  "email": "user@example.com",
  "iss": "auth-service",
  "iat": 1234567890,
  "exp": 1234568790,
  "type": "access"
}
```

**Refresh Token:**
```json
{
  "sub": "uuid-of-user",
  "iss": "auth-service",
  "iat": 1234567890,
  "exp": 1235172690,
  "type": "refresh"
}
```

Note: Refresh tokens intentionally omit email — they're only used for token rotation, not for API authorization.

## Alternatives Considered

| Approach | Why Not |
|---|---|
| Opaque tokens only | Requires database lookup on every API request. Defeats the purpose of stateless authorization. |
| Long-lived JWT (no refresh) | Unacceptable security: a stolen token is valid for its entire lifetime. Short-lived + refresh is standard. |
| PASETO (Platform-Agnostic Security Tokens) | Stronger security model (no algorithm confusion possible), but less ecosystem support. JWT with explicit algorithm validation achieves the same safety. |
| Cookie-based sessions | Tied to browser cookies, doesn't work for mobile apps or cross-domain API consumers. |

## Consequences

- **Positive:** Stateless access verification, standard format with wide library support, clear token lifecycle, rotation prevents replay.
- **Negative:** Cannot instantly revoke access tokens (15-min window), JWT size is larger than opaque tokens, requires secure secret management.
- **Mitigation:** 15-minute TTL limits blast radius. For instant revocation needs, add a token blocklist in Redis (checked on validation).
