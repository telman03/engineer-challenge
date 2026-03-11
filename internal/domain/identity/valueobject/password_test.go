package valueobject_test

import (
	"testing"

	"github.com/atls-academy/engineer-challenge/internal/domain/identity/valueobject"
)

func TestNewPassword_Valid(t *testing.T) {
	validPasswords := []string{
		"MyP@ssw0rd!",
		"Abcdefg1!",
		"C0mpl3x!Pass",
		"12345678aA!",
	}

	for _, pw := range validPasswords {
		_, err := valueobject.NewPassword(pw, valueobject.DefaultPasswordPolicy)
		if err != nil {
			t.Errorf("NewPassword(%q) returned error: %v", pw, err)
		}
	}
}

func TestNewPassword_TooShort(t *testing.T) {
	_, err := valueobject.NewPassword("Ab1!", valueobject.DefaultPasswordPolicy)
	if err == nil {
		t.Error("password with < 8 chars should be rejected")
	}
}

func TestNewPassword_TooLong(t *testing.T) {
	long := make([]byte, 129)
	for i := range long {
		long[i] = 'a'
	}
	long[0] = 'A'
	long[1] = '1'
	long[2] = '!'
	_, err := valueobject.NewPassword(string(long), valueobject.DefaultPasswordPolicy)
	if err == nil {
		t.Error("password with > 128 chars should be rejected")
	}
}

func TestNewPassword_MissingUppercase(t *testing.T) {
	_, err := valueobject.NewPassword("abcdefg1!", valueobject.DefaultPasswordPolicy)
	if err == nil {
		t.Error("password without uppercase should be rejected")
	}
}

func TestNewPassword_MissingLowercase(t *testing.T) {
	_, err := valueobject.NewPassword("ABCDEFG1!", valueobject.DefaultPasswordPolicy)
	if err == nil {
		t.Error("password without lowercase should be rejected")
	}
}

func TestNewPassword_MissingDigit(t *testing.T) {
	_, err := valueobject.NewPassword("Abcdefgh!", valueobject.DefaultPasswordPolicy)
	if err == nil {
		t.Error("password without digit should be rejected")
	}
}

func TestNewPassword_MissingSpecial(t *testing.T) {
	_, err := valueobject.NewPassword("Abcdefg1h", valueobject.DefaultPasswordPolicy)
	if err == nil {
		t.Error("password without special char should be rejected")
	}
}

func TestPassword_Plaintext(t *testing.T) {
	pw, _ := valueobject.NewPassword("MyP@ssw0rd!", valueobject.DefaultPasswordPolicy)
	if pw.Plaintext() != "MyP@ssw0rd!" {
		t.Error("Plaintext() should return original password")
	}
}
