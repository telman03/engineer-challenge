package valueobject_test

import (
	"testing"

	"github.com/atls-academy/engineer-challenge/internal/domain/identity/valueobject"
)

func TestNewEmail_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user@example.com", "user@example.com"},
		{"USER@EXAMPLE.COM", "user@example.com"},
		{"  User@Example.Com  ", "user@example.com"},
	}
	for _, tc := range tests {
		email, err := valueobject.NewEmail(tc.input)
		if err != nil {
			t.Errorf("NewEmail(%q) returned error: %v", tc.input, err)
			continue
		}
		if email.String() != tc.expected {
			t.Errorf("NewEmail(%q).String() = %q, want %q", tc.input, email.String(), tc.expected)
		}
	}
}

func TestNewEmail_Invalid(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"not-an-email",
		"@example.com",
		"user@",
		"user@.com",
	}
	for _, input := range tests {
		_, err := valueobject.NewEmail(input)
		if err == nil {
			t.Errorf("NewEmail(%q) should have returned error", input)
		}
	}
}

func TestEmail_Equals(t *testing.T) {
	e1, _ := valueobject.NewEmail("user@example.com")
	e2, _ := valueobject.NewEmail("USER@EXAMPLE.COM")
	e3, _ := valueobject.NewEmail("other@example.com")

	if !e1.Equals(e2) {
		t.Error("same normalized email should be equal")
	}
	if e1.Equals(e3) {
		t.Error("different emails should not be equal")
	}
}
