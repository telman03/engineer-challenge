# ADR-006: bcrypt for Password Hashing

## Status
Accepted

## Date
2026-03-06

## Context

Passwords must be securely stored. We need a hashing algorithm that is resistant to brute force, widely supported, and provides tunable difficulty.

## Decision

We use **bcrypt** with **cost factor 12** for all password hashing.

## Rationale

### Why bcrypt?

1. **Adaptive cost** — the cost factor can be increased as hardware improves without any data migration. Existing hashes remain valid; only new hashes use the higher cost.

2. **Built-in salt** — bcrypt generates a unique random salt for each hash, preventing rainbow table attacks. No need to manage salts separately.

3. **Constant-time comparison** — `bcrypt.CompareHashAndPassword` uses constant-time comparison, preventing timing attacks that could leak information about which characters matched.

4. **Battle-tested** — bcrypt has been in production use since 1999. It's the most widely deployed adaptive password hashing function.

5. **Universal library support** — Go's `golang.org/x/crypto/bcrypt` is well-maintained and part of the Go extended standard library.

### Why Cost Factor 12?

| Cost | Time per Hash | Brute Force Rate |
|---|---|---|
| 10 | ~65ms | ~15/sec |
| 11 | ~130ms | ~7/sec |
| **12** | **~250ms** | **~4/sec** |
| 13 | ~500ms | ~2/sec |
| 14 | ~1s | ~1/sec |

Cost 12 provides:
- **Sufficient brute force resistance**: 4 hashes/second per CPU core means cracking a single 8-character password with brute force would take years.
- **Acceptable UX latency**: 250ms for login/registration is imperceptible to users when combined with network round-trip time.
- **Combined with rate limiting**: Even if an attacker bypasses rate limiting, bcrypt's computational cost bounds the attack rate server-side.

### Implementation

```go
const bcryptCost = 12

func (h *BcryptHasher) Hash(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
    return string(bytes), err
}

func (h *BcryptHasher) Verify(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

The `PasswordHasher` interface allows swapping bcrypt for another algorithm without touching domain or application code.

## Alternatives Considered

| Algorithm | Assessment |
|---|---|
| **Argon2id** | Theoretically superior (memory-hard, resistant to GPU attacks). Winner of Password Hashing Competition (2015). However: less widely supported, more complex parameter tuning (memory, iterations, parallelism), and bcrypt is sufficient for our threat model. Would switch for ultra-high-value targets. |
| **scrypt** | Memory-hard like Argon2. Less standardized parameter recommendations. bcrypt is simpler to configure correctly. |
| **PBKDF2** | NIST-approved but not memory-hard. Vulnerable to GPU-accelerated attacks at lower iteration counts. Not recommended for new applications when bcrypt/Argon2 are available. |
| **SHA-256 + salt** | Fast hash — not an adaptive function. Trivially brute-forced with modern GPUs. Completely unacceptable for password storage. |

## Consequences

- **Positive:** Industry-standard security, simple API, built-in salt, tunable cost, Go standard library support.
- **Negative:** Not memory-hard (vulnerable to ASIC/FPGA attacks in theory), 250ms per hash adds to login latency, CPU-bound operation prevents horizontal scaling of hash operations.
- **Mitigation:** Cost factor can be increased over time. For extreme security requirements, migrate to Argon2id — the `PasswordHasher` interface makes this a single-file change.
