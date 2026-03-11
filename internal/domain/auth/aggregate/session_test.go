package aggregate_test

import (
	"testing"
	"time"

	"github.com/atls-academy/engineer-challenge/internal/domain/auth/aggregate"
	"github.com/atls-academy/engineer-challenge/internal/domain/auth/valueobject"
)

func TestCreateSession_Success(t *testing.T) {
	session, err := aggregate.CreateSession("user-123", "refresh-token-abc", "Mozilla/5.0", "192.168.1.1")
	if err != nil {
		t.Fatalf("CreateSession() error: %v", err)
	}
	if session.ID.String() == "" {
		t.Error("session ID should not be empty")
	}
	if session.UserID != "user-123" {
		t.Error("user ID mismatch")
	}
	if session.RefreshToken != "refresh-token-abc" {
		t.Error("refresh token mismatch")
	}
	if !session.IsActive() {
		t.Error("new session should be active")
	}
	events := session.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestCreateSession_EmptyUserID(t *testing.T) {
	_, err := aggregate.CreateSession("", "refresh-token", "ua", "ip")
	if err == nil {
		t.Error("should fail with empty user ID")
	}
}

func TestCreateSession_EmptyRefreshToken(t *testing.T) {
	_, err := aggregate.CreateSession("user-123", "", "ua", "ip")
	if err == nil {
		t.Error("should fail with empty refresh token")
	}
}

func TestSession_Revoke(t *testing.T) {
	session, _ := aggregate.CreateSession("user-123", "rt", "ua", "ip")
	_ = session.Events()

	err := session.Revoke()
	if err != nil {
		t.Fatalf("Revoke() error: %v", err)
	}
	if session.IsActive() {
		t.Error("revoked session should not be active")
	}
	events := session.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 revoke event, got %d", len(events))
	}
}

func TestSession_DoubleRevoke(t *testing.T) {
	session, _ := aggregate.CreateSession("user-123", "rt", "ua", "ip")
	session.Revoke()

	err := session.Revoke()
	if err == nil {
		t.Error("double revoke should return error")
	}
}

func TestSession_RotateRefreshToken(t *testing.T) {
	session, _ := aggregate.CreateSession("user-123", "old-token", "ua", "ip")
	oldExpiry := session.ExpiresAt

	time.Sleep(10 * time.Millisecond)
	session.RotateRefreshToken("new-token")

	if session.RefreshToken != "new-token" {
		t.Error("refresh token should be updated")
	}
	if !session.ExpiresAt.After(oldExpiry) {
		t.Error("expiry should be extended")
	}
}

func TestSession_ExpiredIsInactive(t *testing.T) {
	id := valueobject.NewSessionID()
	now := time.Now()
	session := aggregate.ReconstructSession(
		id, "user-123", "rt", "ua", "ip",
		now.Add(-1*time.Hour), now.Add(-2*time.Hour), nil,
	)
	if session.IsActive() {
		t.Error("expired session should not be active")
	}
}

func TestSession_Constants(t *testing.T) {
	if aggregate.AccessTokenTTL != 15*time.Minute {
		t.Errorf("AccessTokenTTL = %v, want 15m", aggregate.AccessTokenTTL)
	}
	if aggregate.RefreshTokenTTL != 7*24*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want 7d", aggregate.RefreshTokenTTL)
	}
	if aggregate.MaxSessions != 5 {
		t.Errorf("MaxSessions = %d, want 5", aggregate.MaxSessions)
	}
}
