package token

import (
	"slices"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// verifyLikeGateway parses a token with the same rules as the gateway's
// auth.JWTService.Verify, so a token it accepts here the gateway accepts too.
func verifyLikeGateway(t *testing.T, tokenString, secret, issuer string) (*Claims, error) {
	t.Helper()

	claims := &Claims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(issuer),
	)

	_, err := parser.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	return claims, err
}

func TestIssueIsAcceptedByTheGatewayRules(t *testing.T) {

	iss, err := NewIssuer("shared-secret", "fiurozz", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	before := time.Now()
	access, err := iss.Issue(Subject{ID: "0199f2b0-0000-7000-8000-000000000001", Email: "reader@example.com", Name: "Reader", Roles: []string{"customer"}})
	if err != nil {
		t.Fatal(err)
	}

	claims, err := verifyLikeGateway(t, access.Token, "shared-secret", "fiurozz")
	if err != nil {
		t.Fatalf("gateway rules refuse the token: %v", err)
	}

	if claims.Subject != "0199f2b0-0000-7000-8000-000000000001" || claims.Email != "reader@example.com" || claims.Name != "Reader" || !slices.Equal(claims.Roles, []string{"customer"}) {
		t.Errorf("claims = %+v", claims)
	}

	if d := access.ExpiresAt.Sub(before); d < 15*time.Minute-time.Second || d > 15*time.Minute+time.Second {
		t.Errorf("expires in %v, want 15m", d)
	}
}

func TestIssueIsRefusedWithAnotherSecretOrIssuer(t *testing.T) {

	iss, _ := NewIssuer("shared-secret", "fiurozz", time.Minute)
	access, _ := iss.Issue(Subject{ID: "id", Email: "a@example.com", Roles: []string{"customer"}})

	if _, err := verifyLikeGateway(t, access.Token, "other-secret", "fiurozz"); err == nil {
		t.Error("token accepted with another secret")
	}
	if _, err := verifyLikeGateway(t, access.Token, "shared-secret", "someone-else"); err == nil {
		t.Error("token accepted with another issuer")
	}
}

func TestIssuedTokenExpires(t *testing.T) {

	iss, _ := NewIssuer("shared-secret", "fiurozz", time.Minute)
	iss.now = func() time.Time { return time.Now().Add(-2 * time.Minute) }

	access, _ := iss.Issue(Subject{ID: "id", Email: "a@example.com", Roles: []string{"customer"}})

	if _, err := verifyLikeGateway(t, access.Token, "shared-secret", "fiurozz"); err == nil {
		t.Error("expired token accepted")
	}
}

func TestNewIssuerRefusesEmptySecret(t *testing.T) {
	if _, err := NewIssuer("", "fiurozz", time.Minute); err == nil {
		t.Error("empty secret accepted")
	}
}
