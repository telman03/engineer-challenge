package crypto_test

import (
	"testing"

	"github.com/atls-academy/engineer-challenge/internal/infrastructure/crypto"
)

const testSecret = "super-secret-key-for-testing-only"
const testIssuer = "test-auth-service"

func TestJWTIssuer_IssueAndValidateAccessToken(t *testing.T) {
	issuer := crypto.NewJWTIssuer(testSecret, testIssuer)

	token, err := issuer.IssueAccessToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("IssueAccessToken() error: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	claims, err := issuer.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "test@example.com")
	}
}

func TestJWTIssuer_RefreshTokenNotValidAsAccess(t *testing.T) {
	issuer := crypto.NewJWTIssuer(testSecret, testIssuer)

	refresh, err := issuer.IssueRefreshToken("user-123")
	if err != nil {
		t.Fatalf("IssueRefreshToken() error: %v", err)
	}

	_, err = issuer.ValidateAccessToken(refresh)
	if err == nil {
		t.Error("refresh token should not be valid as access token")
	}
}

func TestJWTIssuer_WrongSecret(t *testing.T) {
	issuerA := crypto.NewJWTIssuer("secret-A", testIssuer)
	issuerB := crypto.NewJWTIssuer("secret-B", testIssuer)

	token, _ := issuerA.IssueAccessToken("user-123", "test@example.com")

	_, err := issuerB.ValidateAccessToken(token)
	if err == nil {
		t.Error("token signed with different secret should fail validation")
	}
}

func TestJWTIssuer_InvalidToken(t *testing.T) {
	issuer := crypto.NewJWTIssuer(testSecret, testIssuer)

	_, err := issuer.ValidateAccessToken("not.a.valid.token")
	if err == nil {
		t.Error("garbage string should fail validation")
	}
}

func TestJWTIssuer_IssueRefreshToken(t *testing.T) {
	issuer := crypto.NewJWTIssuer(testSecret, testIssuer)

	token, err := issuer.IssueRefreshToken("user-456")
	if err != nil {
		t.Fatalf("IssueRefreshToken() error: %v", err)
	}
	if token == "" {
		t.Error("refresh token should not be empty")
	}
}
