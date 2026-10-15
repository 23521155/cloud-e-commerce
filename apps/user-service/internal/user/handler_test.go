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
	"time"

	"github.com/fizzisme/user-service/internal/token"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// fakeAccounts returns what the test set and records the input it got.
type fakeAccounts struct {
	user   User
	access token.Access
	err    error

	got      RegisterInput
	gotEmail string
	gotPass  string
	gotID    string

	gotName    string
	gotProfile Profile
}

func (f *fakeAccounts) Register(ctx context.Context, in RegisterInput) (User, error) {
	f.got = in
	return f.user, f.err
}

func (f *fakeAccounts) Login(ctx context.Context, email, password string) (User, token.Access, error) {
	f.gotEmail, f.gotPass = email, password
	return f.user, f.access, f.err
}

func (f *fakeAccounts) Me(ctx context.Context, id string) (User, error) {
	f.gotID = id
	return f.user, f.err
}

func (f *fakeAccounts) UpdateProfile(ctx context.Context, id, name string, p Profile) (User, error) {
	f.gotID, f.gotName, f.gotProfile = id, name, p
	return f.user, f.err
}

// send makes one request to the handler's routes and returns the response.
func send(t *testing.T, svc accounts, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(svc, zap.NewNop()).Register(r.Group(""))

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

// postRegister sends body to POST /auth/register and returns the response.
func postRegister(t *testing.T, svc accounts, body string) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, svc, http.MethodPost, "/auth/register", body, nil)
}

func TestRegisterCreated(t *testing.T) {

	svc := &fakeAccounts{user: User{
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
			w := postRegister(t, &fakeAccounts{err: tt.err}, `{"name":"R","email":"r@example.com","password":"12345678"}`)
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestRegisterHidesInternalErrors(t *testing.T) {

	w := postRegister(t, &fakeAccounts{err: errors.New("mssql: login failed for user sa")}, `{"name":"R","email":"r@example.com","password":"12345678"}`)

	if strings.Contains(w.Body.String(), "mssql") {
		t.Errorf("response leaks the internal error: %s", w.Body)
	}
}

func TestRegisterRejectsMalformedBody(t *testing.T) {

	svc := &fakeAccounts{}
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
	w := postRegister(t, &fakeAccounts{}, body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestLoginReturnsToken(t *testing.T) {

	expires := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	svc := &fakeAccounts{
		user:   User{ID: "u1", Email: "reader@example.com", Name: "Reader", Role: RoleCustomer, PasswordHash: "$argon2id$secret"},
		access: token.Access{Token: "signed.jwt.token", ExpiresAt: expires},
	}

	w := send(t, svc, http.MethodPost, "/auth/login", `{"email":"Reader@example.com","password":"12345678"}`, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body)
	}
	if svc.gotEmail != "Reader@example.com" || svc.gotPass != "12345678" {
		t.Errorf("service got %q / %q", svc.gotEmail, svc.gotPass)
	}

	var body struct {
		AccessToken string    `json:"accessToken"`
		TokenType   string    `json:"tokenType"`
		ExpiresAt   time.Time `json:"expiresAt"`
		User        struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.AccessToken != "signed.jwt.token" || body.TokenType != "Bearer" || !body.ExpiresAt.Equal(expires) || body.User.ID != "u1" {
		t.Errorf("body = %+v", body)
	}
	if strings.Contains(w.Body.String(), "argon2id") {
		t.Errorf("response leaks the password hash: %s", w.Body)
	}
}

func TestLoginWrongCredentials(t *testing.T) {

	w := send(t, &fakeAccounts{err: ErrInvalidCredentials}, http.MethodPost, "/auth/login", `{"email":"a@example.com","password":"x"}`, nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestMeReadsTheGatewayIdentity(t *testing.T) {

	svc := &fakeAccounts{user: User{ID: "u1", Email: "reader@example.com", Name: "Reader", Role: RoleCustomer}}

	w := send(t, svc, http.MethodGet, "/me", "", map[string]string{"X-User-ID": "u1"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body)
	}
	if svc.gotID != "u1" {
		t.Errorf("service got id %q, want u1", svc.gotID)
	}
}

func TestMeWithoutIdentity(t *testing.T) {

	tests := []struct {
		name    string
		headers map[string]string
		err     error
	}{
		{"no token", nil, nil},
		{"account gone", map[string]string{"X-User-ID": "u1"}, ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := send(t, &fakeAccounts{err: tt.err}, http.MethodGet, "/me", "", tt.headers)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", w.Code)
			}
		})
	}
}

func TestUpdateMe(t *testing.T) {

	svc := &fakeAccounts{user: User{ID: "u1", Name: "New Name", Profile: Profile{City: "Hà Nội"}}}

	w := send(t, svc, http.MethodPut, "/me",
		`{"name":"New Name","phone":"0901","address":"1 Lê Lợi","city":"Hà Nội","postalCode":"100000","country":"Vietnam","email":"ignored@example.com"}`,
		map[string]string{"X-User-ID": "u1"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body)
	}
	want := Profile{Phone: "0901", Address: "1 Lê Lợi", City: "Hà Nội", PostalCode: "100000", Country: "Vietnam"}
	if svc.gotID != "u1" || svc.gotName != "New Name" || svc.gotProfile != want {
		t.Errorf("service got %q %q %+v", svc.gotID, svc.gotName, svc.gotProfile)
	}
	if !strings.Contains(w.Body.String(), `"city":"Hà Nội"`) {
		t.Errorf("body = %s, want the updated profile", w.Body)
	}
}

func TestUpdateMeNeedsSignIn(t *testing.T) {

	svc := &fakeAccounts{}
	w := send(t, svc, http.MethodPut, "/me", `{"name":"X"}`, nil)

	if w.Code != http.StatusUnauthorized || svc.gotName != "" {
		t.Errorf("status = %d, service called = %v", w.Code, svc.gotName != "")
	}
}
