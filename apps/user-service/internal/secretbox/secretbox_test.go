package secretbox

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"
)

func newKey(t *testing.T) string {
	t.Helper()
	k := make([]byte, 32)
	rand.Read(k)
	return base64.StdEncoding.EncodeToString(k)
}

func TestSealOpen(t *testing.T) {

	box, err := New(newKey(t))
	if err != nil {
		t.Fatal(err)
	}

	sealed, err := box.Seal([]byte("JBSWY3DPEHPK3PXP"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("JBSWY3DPEHPK3PXP")) {
		t.Fatal("the sealed value contains the secret")
	}

	again, _ := box.Seal([]byte("JBSWY3DPEHPK3PXP"))
	if bytes.Equal(sealed, again) {
		t.Error("two seals of one value are equal: the nonce is not random")
	}

	got, err := box.Open(sealed)
	if err != nil || string(got) != "JBSWY3DPEHPK3PXP" {
		t.Errorf("Open = %q, %v", got, err)
	}
}

func TestOpenRefusesAnotherKeyOrAChange(t *testing.T) {

	box, _ := New(newKey(t))
	other, _ := New(newKey(t))
	sealed, _ := box.Seal([]byte("secret"))

	if _, err := other.Open(sealed); !errors.Is(err, ErrOpen) {
		t.Errorf("another key: %v, want ErrOpen", err)
	}

	sealed[len(sealed)-1] ^= 1
	if _, err := box.Open(sealed); !errors.Is(err, ErrOpen) {
		t.Errorf("changed value: %v, want ErrOpen", err)
	}
	if _, err := box.Open([]byte("x")); !errors.Is(err, ErrOpen) {
		t.Errorf("short value: %v, want ErrOpen", err)
	}
}

func TestNewRefusesBadKeys(t *testing.T) {
	for _, k := range []string{"", "not base64!", base64.StdEncoding.EncodeToString(make([]byte, 16))} {
		if _, err := New(k); err == nil {
			t.Errorf("New(%q) accepted", k)
		}
	}
}
