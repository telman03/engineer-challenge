package aggregate_test

import (
	"testing"
	"time"

	"github.com/atls-academy/engineer-challenge/internal/domain/identity/aggregate"
	"github.com/atls-academy/engineer-challenge/internal/domain/identity/valueobject"
)

func TestRegister_Success(t *testing.T) {
	email, _ := valueobject.NewEmail("test@example.com")
	user, err := aggregate.Register(email, "$2a$12$fakehash")
	if err != nil {
		t.Fatalf("Register() error: %v", err)
	}
	if user.ID.String() == "" {
		t.Error("user ID should not be empty")
	}
	if user.Email.String() != "test@example.com" {
		t.Error("email mismatch")
	}
	if user.Status != aggregate.UserStatusActive {
		t.Error("new user should be active")
	}
	if user.CreatedAt.IsZero() {
		t.Error("created_at should be set")
	}
	events := user.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventName() != "identity.user.registered" {
		t.Errorf("event name = %q, want %q", events[0].EventName(), "identity.user.registered")
	}
}

func TestRegister_EmptyHash(t *testing.T) {
	email, _ := valueobject.NewEmail("test@example.com")
	_, err := aggregate.Register(email, "")
	if err == nil {
		t.Error("Register with empty hash should fail")
	}
}

func TestRequestPasswordReset_Success(t *testing.T) {
	email, _ := valueobject.NewEmail("test@example.com")
	user, _ := aggregate.Register(email, "$2a$12$fakehash")
	_ = user.Events()

	token, err := user.RequestPasswordReset()
	if err != nil {
		t.Fatalf("RequestPasswordReset() error: %v", err)
	}
	if token.Token == "" {
		t.Error("token should not be empty")
	}
	if token.IsExpired() {
		t.Error("token should not be expired")
	}

	events := user.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestRequestPasswordReset_BlockedUser(t *testing.T) {
	email, _ := valueobject.NewEmail("test@example.com")
	id := valueobject.NewUserID()
	user := aggregate.Reconstruct(id, email, "$2a$12$fakehash", aggregate.UserStatusBlocked, nil, time.Now(), time.Now())

	_, err := user.RequestPasswordReset()
	if err == nil {
		t.Error("blocked user should not be able to request password reset")
	}
}

func TestResetPassword_Success(t *testing.T) {
	email, _ := valueobject.NewEmail("test@example.com")
	user, _ := aggregate.Register(email, "$2a$12$fakehash")
	_ = user.Events()

	token, _ := user.RequestPasswordReset()
	_ = user.Events()

	err := user.ResetPassword(token.Token, "$2a$12$newhash")
	if err != nil {
		t.Fatalf("ResetPassword() error: %v", err)
	}
	if user.PasswordHash != "$2a$12$newhash" {
		t.Error("password hash should be updated")
	}
	if user.ResetToken == nil || !user.ResetToken.Used {
		t.Error("reset token should be marked as used")
	}
}

func TestResetPassword_NoToken(t *testing.T) {
	email, _ := valueobject.NewEmail("test@example.com")
	user, _ := aggregate.Register(email, "$2a$12$fakehash")

	err := user.ResetPassword("sometoken", "$2a$12$newhash")
	if err == nil {
		t.Error("should fail when no reset token issued")
	}
}

func TestResetPassword_WrongToken(t *testing.T) {
	email, _ := valueobject.NewEmail("test@example.com")
	user, _ := aggregate.Register(email, "$2a$12$fakehash")
	user.RequestPasswordReset()

	err := user.ResetPassword("wrong-token", "$2a$12$newhash")
	if err == nil {
		t.Error("should fail with wrong token")
	}
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	email, _ := valueobject.NewEmail("test@example.com")
	id := valueobject.NewUserID()
	expired := valueobject.ResetToken{
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		Used:      false,
	}
	user := aggregate.Reconstruct(id, email, "$2a$12$fakehash", aggregate.UserStatusActive, &expired, time.Now(), time.Now())

	err := user.ResetPassword("expired-token", "$2a$12$newhash")
	if err == nil {
		t.Error("should fail with expired token")
	}
}

func TestResetPassword_UsedToken(t *testing.T) {
	email, _ := valueobject.NewEmail("test@example.com")
	id := valueobject.NewUserID()
	used := valueobject.ResetToken{
		Token:     "used-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Used:      true,
	}
	user := aggregate.Reconstruct(id, email, "$2a$12$fakehash", aggregate.UserStatusActive, &used, time.Now(), time.Now())

	err := user.ResetPassword("used-token", "$2a$12$newhash")
	if err == nil {
		t.Error("should fail with already used token")
	}
}

func TestResetPassword_EmptyNewHash(t *testing.T) {
	email, _ := valueobject.NewEmail("test@example.com")
	user, _ := aggregate.Register(email, "$2a$12$fakehash")
	token, _ := user.RequestPasswordReset()

	err := user.ResetPassword(token.Token, "")
	if err == nil {
		t.Error("should fail with empty new hash")
	}
}
