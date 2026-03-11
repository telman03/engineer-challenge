package valueobject_test

import (
	"testing"
	"time"

	"github.com/atls-academy/engineer-challenge/internal/domain/identity/valueobject"
)

func TestNewResetToken(t *testing.T) {
	token, err := valueobject.NewResetToken()
	if err != nil {
		t.Fatalf("NewResetToken() returned error: %v", err)
	}

	if len(token.Token) != 64 {
		t.Errorf("token should be 64 hex chars (32 bytes), got %d chars", len(token.Token))
	}

	if token.Used {
		t.Error("new token should not be used")
	}

	if token.IsExpired() {
		t.Error("new token should not be expired")
	}

	if !token.IsValid() {
		t.Error("new token should be valid")
	}
}

func TestResetToken_Uniqueness(t *testing.T) {
	t1, _ := valueobject.NewResetToken()
	t2, _ := valueobject.NewResetToken()

	if t1.Token == t2.Token {
		t.Error("two tokens should be different")
	}
}

func TestResetToken_MarkUsed(t *testing.T) {
	token, _ := valueobject.NewResetToken()

	used := token.MarkUsed()

	if !used.Used {
		t.Error("MarkUsed() should set Used to true")
	}
	if used.IsValid() {
		t.Error("used token should not be valid")
	}
	// Original should be unchanged (value semantics)
	if token.Used {
		t.Error("original token should not be modified")
	}
}

func TestResetToken_Expired(t *testing.T) {
	token := valueobject.ResetToken{
		Token:     "test-token",
		ExpiresAt: time.Now().Add(-1 * time.Minute),
		Used:      false,
	}

	if !token.IsExpired() {
		t.Error("token with past ExpiresAt should be expired")
	}
	if token.IsValid() {
		t.Error("expired token should not be valid")
	}
}
