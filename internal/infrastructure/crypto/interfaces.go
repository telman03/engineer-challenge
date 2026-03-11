package crypto

// PasswordHasher defines the contract for password hashing.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) bool
}

// TokenIssuer defines the contract for JWT token operations.
type TokenIssuer interface {
	IssueAccessToken(userID, email string) (string, error)
	IssueRefreshToken(userID string) (string, error)
	ValidateAccessToken(token string) (*TokenClaims, error)
}

type TokenClaims struct {
	UserID string
	Email  string
}
