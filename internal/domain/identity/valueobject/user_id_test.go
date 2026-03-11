package valueobject_test

import (
	"testing"

	"github.com/atls-academy/engineer-challenge/internal/domain/identity/valueobject"
)

func TestNewUserID(t *testing.T) {
	id := valueobject.NewUserID()
	if id.IsZero() {
		t.Error("new UserID should not be zero")
	}
	if id.String() == "" {
		t.Error("UserID.String() should not be empty")
	}
}

func TestUserID_Uniqueness(t *testing.T) {
	id1 := valueobject.NewUserID()
	id2 := valueobject.NewUserID()
	if id1.Equals(id2) {
		t.Error("two new UserIDs should be different")
	}
}

func TestUserIDFromString_Valid(t *testing.T) {
	original := valueobject.NewUserID()
	parsed, err := valueobject.UserIDFromString(original.String())
	if err != nil {
		t.Fatalf("UserIDFromString(%q) returned error: %v", original.String(), err)
	}
	if !original.Equals(parsed) {
		t.Error("parsed ID should equal original")
	}
}

func TestUserIDFromString_Invalid(t *testing.T) {
	_, err := valueobject.UserIDFromString("not-a-uuid")
	if err == nil {
		t.Error("invalid UUID string should return error")
	}
}
