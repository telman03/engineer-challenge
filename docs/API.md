# GraphQL API Reference

## Endpoint

```
POST /graphql
Content-Type: application/json
```

All operations go through a single endpoint. Authentication is via the `Authorization` header for protected queries.

---

## Types

### User
```graphql
type User {
  id: ID!          # UUID v4
  email: String!   # Normalized email address
  status: String!  # "active" or "blocked"
}
```

### AuthPayload
Returned on successful login. Contains the token pair and user info.
```graphql
type AuthPayload {
  accessToken: String!   # JWT (15-minute TTL)
  refreshToken: String!  # JWT (7-day TTL)
  user: User!
}
```

### TokenPair
Returned on successful token refresh.
```graphql
type TokenPair {
  accessToken: String!   # New JWT access token
  refreshToken: String!  # New JWT refresh token (old one is invalidated)
}
```

### ResetRequestPayload
Returned when a password reset is requested.
```graphql
type ResetRequestPayload {
  success: Boolean!  # Always true (anti-enumeration)
  token: String      # Reset token (dev only — production sends via email)
}
```

---

## Queries

### me
Returns the currently authenticated user's profile.

**Requires**: `Authorization: Bearer <accessToken>` header.

```graphql
query {
  me {
    id
    email
    status
  }
}
```

**Response:**
```json
{
  "data": {
    "me": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "status": "active"
    }
  }
}
```

**Errors:**
| Condition | Error Message |
|---|---|
| Missing Authorization header | `"unauthorized"` |
| Invalid/expired access token | `"unauthorized"` |
| User not found (deleted) | `"user not found: ..."` |

---

## Mutations

### register
Creates a new user account.

```graphql
mutation {
  register(email: "user@example.com", password: "MyP@ssw0rd!") {
    id
    email
    status
  }
}
```

**Arguments:**
| Argument | Type | Validation |
|---|---|---|
| `email` | String! | RFC 5321 format, max 254 chars, normalized to lowercase |
| `password` | String! | Min 8 chars, max 128 chars, requires uppercase + lowercase + digit + special |

**Response:**
```json
{
  "data": {
    "register": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "status": "active"
    }
  }
}
```

**Errors:**
| Condition | Error Message |
|---|---|
| Invalid email format | `"invalid email: invalid email format: ..."` |
| Password too short | `"invalid password: password must be at least 8 characters"` |
| Missing uppercase | `"invalid password: password must contain at least one uppercase letter"` |
| Missing lowercase | `"invalid password: password must contain at least one lowercase letter"` |
| Missing digit | `"invalid password: password must contain at least one digit"` |
| Missing special char | `"invalid password: password must contain at least one special character"` |
| Email already exists | `"email already registered"` |
| Rate limited | `"too many registration attempts, please try again later"` |

**Rate Limit:** 20 registrations per minute (global).

---

### login
Authenticates a user and returns JWT tokens.

```graphql
mutation {
  login(email: "user@example.com", password: "MyP@ssw0rd!") {
    accessToken
    refreshToken
    user {
      id
      email
    }
  }
}
```

**Arguments:**
| Argument | Type | Description |
|---|---|---|
| `email` | String! | User's email address |
| `password` | String! | User's password |

**Response:**
```json
{
  "data": {
    "login": {
      "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "email": "user@example.com"
      }
    }
  }
}
```

**Errors:**
| Condition | Error Message |
|---|---|
| Wrong email or password | `"invalid credentials"` |
| Account blocked | `"account is not active"` |
| Rate limited | `"too many login attempts, please try again later"` |

**Security notes:**
- Error message is identical for "email not found" and "wrong password" (prevents enumeration)
- Rate limited per IP address: 10 attempts per minute
- If user has 5+ active sessions, the oldest session is automatically revoked
- Client IP and User-Agent are stored with the session for audit purposes

---

### refreshToken
Exchanges a valid refresh token for a new token pair. The old refresh token is invalidated (rotation).

```graphql
mutation {
  refreshToken(refreshToken: "eyJhbGciOiJIUzI1NiIs...") {
    accessToken
    refreshToken
  }
}
```

**Arguments:**
| Argument | Type | Description |
|---|---|---|
| `refreshToken` | String! | The current refresh token |

**Response:**
```json
{
  "data": {
    "refreshToken": {
      "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    }
  }
}
```

**Errors:**
| Condition | Error Message |
|---|---|
| Invalid/unknown refresh token | `"invalid refresh token"` |
| Session revoked or expired | `"session expired or revoked"` |

**Important:** After this mutation, the old refresh token is no longer valid. The client **must** store the new refresh token.

---

### requestPasswordReset
Initiates a password reset flow. Generates a reset token (in production, this would be sent via email).

```graphql
mutation {
  requestPasswordReset(email: "user@example.com") {
    success
    token
  }
}
```

**Arguments:**
| Argument | Type | Description |
|---|---|---|
| `email` | String! | Email address of the account to reset |

**Response:**
```json
{
  "data": {
    "requestPasswordReset": {
      "success": true,
      "token": "a1b2c3d4e5f6...64-hex-chars..."
    }
  }
}
```

**Security notes:**
- **Always returns `success: true`** even if the email doesn't exist (anti-enumeration)
- The `token` field is returned for development convenience; in production, it would be sent via email and not included in the API response
- Token is 32 crypto-random bytes (64 hex characters)
- Token expires after 15 minutes
- Token is single-use

**Rate Limit:** 3 requests per email per 15 minutes.

---

### resetPassword
Applies a new password using a valid reset token.

```graphql
mutation {
  resetPassword(
    email: "user@example.com"
    token: "a1b2c3d4e5f6...64-hex-chars..."
    newPassword: "NewP@ssw0rd!"
  )
}
```

**Arguments:**
| Argument | Type | Validation |
|---|---|---|
| `email` | String! | Must match the email for the reset token |
| `token` | String! | The reset token received via requestPasswordReset |
| `newPassword` | String! | Must meet password policy (same as registration) |

**Response:**
```json
{
  "data": {
    "resetPassword": true
  }
}
```

**Errors:**
| Condition | Error Message |
|---|---|
| Invalid email format | `"invalid email: ..."` |
| Password policy violation | `"invalid password: ..."` |
| No reset token issued | `"password reset failed: no reset token issued"` |
| Token expired | `"password reset failed: reset token is expired or already used"` |
| Token already used | `"password reset failed: reset token is expired or already used"` |
| Token mismatch | `"password reset failed: invalid reset token"` |
| User not found | `"invalid reset request"` |

---

### logout
Invalidates the current session (placeholder — full implementation pending).

```graphql
mutation {
  logout(refreshToken: "eyJhbGciOiJIUzI1NiIs...") 
}
```

**Response:**
```json
{
  "data": {
    "logout": true
  }
}
```

---

## HTTP Headers

### Request Headers

| Header | Required | Description |
|---|---|---|
| `Content-Type` | Yes | Must be `application/json` |
| `Authorization` | For `me` query | `Bearer <accessToken>` |

### Injected Context

The server automatically extracts and injects into the request context:
- **IP Address** — from `X-Forwarded-For` header (or `RemoteAddr` if not behind a proxy)
- **User-Agent** — from the standard `User-Agent` header
- **Auth Token** — from the `Authorization: Bearer` header

These are used for:
- Rate limiting (IP)
- Session tracking (IP + User-Agent)
- Authentication (token)

---

## Error Format

GraphQL errors follow the standard format:
```json
{
  "data": null,
  "errors": [
    {
      "message": "invalid credentials",
      "locations": [{"line": 2, "column": 3}],
      "path": ["login"]
    }
  ]
}
```

The HTTP status is always `200 OK` (per GraphQL spec). Clients should check the `errors` array.

---

## Other Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/health` | GET | Returns `{"status":"ok"}` — for load balancer health checks |
| `/metrics` | GET | Prometheus metrics (counters, histograms) — for monitoring |
