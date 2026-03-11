package crypto_test

import (
	"testing"

	"github.com/atls-academy/engineer-challenge/internal/infrastructure/crypto"
)

func TestBcryptHasher_HashAndVerify(t *testing.T) {
	hasher := crypto.NewBcryptHasher()

	password := "MyP@ssw0rd!"
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash() returned error: %v", err)
	}
	if hash == "" {
		t.Error("hash should not be empty")
	}
	if hash == password {
		t.Error("hash should not equal plaintext")
	}
	if !hasher.Verify(password, hash) {
		t.Error("Verify() should return true for correct password")
	}
}

func TestBcryptHasher_WrongPassword(t *testing.T) {
	hasher := crypto.NewBcryptHasher()
	hash, _ := hasher.Hash("CorrectP@ss1!")
	if hasher.Verify("WrongP@ss1!", hash) {
		t.Error("Verify() should return false for wrong password")
	}
}

func TestBcryptHasher_UniqueHashes(t *testing.T) {
	hasher := crypto.NewBcryptHasher()
	h1, _ := hasher.Hash("SameP@ss1!")
	h2, _ := hasher.Hash("SameP@ss1!")
	if h1 == h2 {
		t.Error("same password should produce different hashes (different salts)")
	}
	if !hasher.Verify("SameP@ss1!", h1) {
		t.Error("first hash should verify")
	}
	if !hasher.Verify("SameP@ss1!", h2) {
		t.Error("second hash should verify")
	}
}
