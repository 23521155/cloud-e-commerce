package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// fakeRegistrar returns what the test set and records the input it got.
type fakeRegistrar struct {
	user User
	err  error

	got RegisterInput
}

func (f *fakeRegistrar) Register(ctx context.Context, in RegisterInput) (User, error) {
	f.got = in
	return f.user, f.err
}

// postRegister sends body to POST /auth/register and returns the response.
func postRegister(t *testing.T, svc registrar, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(svc, zap.NewNop()).Register(r.Group(""))

	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

func TestRegisterCreated(t *testing.T) {

	svc := &fakeRegistrar{user: User{
		ID:           "0199f2b0-0000-7000-8000-000000000001",
		Email:        "reader@example.com",
		PasswordHash: "$argon2id$secret",
		Name:         "Reader",
		Role:         RoleCustomer,
	}}

	w := postRegister(t, svc, `{"name":"Reader","email":"reader@example.com","password":"12345678"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", w.Code, w.Body)
	}

	if svc.got != (RegisterInput{Name: "Reader", Email: "reader@example.com", Password: "12345678"}) {
		t.Errorf("service got %+v", svc.got)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["id"] != svc.user.ID || body["email"] != "reader@example.com" || body["name"] != "Reader" {
		t.Errorf("body = %v", body)
	}
	if strings.Contains(w.Body.String(), "argon2id") || strings.Contains(w.Body.String(), "password") {
		t.Errorf("response leaks the password hash: %s", w.Body)
	}
}

func TestRegisterErrors(t *testing.T) {

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"invalid input", fmt.Errorf("%w: email is not a valid address", ErrInvalidInput), http.StatusBadRequest},
		{"email taken", ErrEmailTaken, http.StatusConflict},
		{"anything else", errors.New("connection lost"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := postRegister(t, &fakeRegistrar{err: tt.err}, `{"name":"R","email":"r@example.com","password":"12345678"}`)
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestRegisterHidesInternalErrors(t *testing.T) {

	w := postRegister(t, &fakeRegistrar{err: errors.New("mssql: login failed for user sa")}, `{"name":"R","email":"r@example.com","password":"12345678"}`)

	if strings.Contains(w.Body.String(), "mssql") {
		t.Errorf("response leaks the internal error: %s", w.Body)
	}
}

func TestRegisterRejectsMalformedBody(t *testing.T) {

	svc := &fakeRegistrar{}
	w := postRegister(t, svc, `{"name":`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if svc.got != (RegisterInput{}) {
		t.Error("service was called with a malformed body")
	}
}

func TestRegisterRejectsOversizedBody(t *testing.T) {

	body := `{"name":"R","email":"r@example.com","password":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	w := postRegister(t, &fakeRegistrar{}, body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
