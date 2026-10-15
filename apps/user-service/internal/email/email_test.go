package email

import (
	"strings"
	"testing"
)

func TestVerifyEmail(t *testing.T) {

	e, err := VerifyEmail("reader@example.com", "Trần <Bình>", "http://localhost:3000/verify-email?token=abc&x=1")
	if err != nil {
		t.Fatal(err)
	}

	if e.To != "reader@example.com" || !strings.Contains(e.Subject, "Confirm your email") {
		t.Errorf("email = %+v", e)
	}
	if !strings.Contains(e.Text, "http://localhost:3000/verify-email?token=abc&x=1") || !strings.Contains(e.Text, "Trần <Bình>") {
		t.Errorf("text = %q, want the name and link as they are", e.Text)
	}
	// The HTML escapes the name, and the link stays usable.
	if !strings.Contains(e.HTML, "Trần &lt;Bình&gt;") || !strings.Contains(e.HTML, `href="http://localhost:3000/verify-email?token=abc&amp;x=1"`) {
		t.Errorf("html = %q", e.HTML)
	}
}
