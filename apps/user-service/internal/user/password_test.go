package user

import (
	"errors"
	"strings"
	"testing"
)

func TestHashPasswordVerifies(t *testing.T) {

	hash, err := hashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("hash = %q, want argon2id PHC string with the configured cost", hash)
	}
	if strings.Contains(hash, "correct horse battery") {
		t.Fatal("hash contains the password")
	}

	ok, err := verifyPassword("correct horse battery", hash)
	if err != nil || !ok {
		t.Fatalf("verify right password = %v, %v; want true, nil", ok, err)
	}

	ok, err = verifyPassword("correct horse batterY", hash)
	if err != nil || ok {
		t.Fatalf("verify wrong password = %v, %v; want false, nil", ok, err)
	}
}

func TestHashPasswordSaltsEachHash(t *testing.T) {

	a, _ := hashPassword("same password")
	b, _ := hashPassword("same password")

	if a == b {
		t.Fatal("two hashes of one password are equal: the salt is not random")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {

	for _, hash := range []string{
		"",
		"plain-text",
		"$2a$10$abcdefghijklmnopqrstuv", // bcrypt
		"$argon2id$v=19$m=19456,t=2,p=1$***$***",
		"$argon2id$v=16$m=19456,t=2,p=1$c2FsdA$aGFzaA",
	} {
		if _, err := verifyPassword("anything", hash); !errors.Is(err, errMalformedHash) {
			t.Errorf("verifyPassword(%q) error = %v, want errMalformedHash", hash, err)
		}
	}
}
