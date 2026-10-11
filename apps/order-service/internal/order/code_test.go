package order

import (
	"regexp"
	"testing"
)

func TestNewCode(t *testing.T) {

	pattern := regexp.MustCompile(`^MA-[0-9]{6}$`)

	for range 1000 {
		if code := newCode(); !pattern.MatchString(code) {
			t.Fatalf("code %q does not match %s", code, pattern)
		}
	}
}
