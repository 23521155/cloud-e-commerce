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
	tokens Tokens
	err    error

	got      RegisterInput
	gotEmail string
	gotPass  string
	gotID    string

	gotName    string
	gotProfile Profile

	gotDevice  string
	gotRefresh string
	gotNewPass string
}

func (f *fakeAccounts) Register(ctx context.Context, in RegisterInput) (User, error) {
	f.got = in
	return f.user, f.err
}

func (f *fakeAccounts) Login(ctx context.Context, email, password, device string) (User, Tokens, error) {
	f.gotEmail, f.gotPass, f.gotDevice = email, password, device
	return f.user, f.tokens, f.err
}

func (f *fakeAccounts) Refresh(ctx context.Context, refreshToken, device string) (User, Tokens, error) {
	f.gotRefresh, f.gotDevice = refreshToken, device
	return f.user, f.tokens, f.err
}

func (f *fakeAccounts) ChangePassword(ctx context.Context, id, current, password, device string) (User, Tokens, error) {
	f.gotID, f.gotPass, f.gotNewPass, f.gotDevice = id, current, password, device
	return f.user, f.tokens, f.err
}

func (f *fakeAccounts) VerifyEmail(ctx context.Context, token string) (User, error) {
	f.gotRefresh = token
	return f.user, f.err
}

func (f *fakeAccounts) ResendVerification(ctx context.Context, id string) error {
	f.gotID = id
	return f.err
}

func (f *fakeAccounts) RequestPasswordReset(ctx context.Context, email string) error {
	f.gotEmail = email
	return f.err
}

func (f *fakeAccounts) ResetPassword(ctx context.Context, token, password string) error {
	f.gotRefresh, f.gotNewPass = token, password
	return f.err
}

func (f *fakeAccounts) SetupTOTP(ctx context.Context, id string) (TOTPSetup, error) {
	f.gotID = id
	return TOTPSetup{Secret: "JBSWY3DPEHPK3PXP", URL: "otpauth://totp/Marginalleya:reader@example.com?secret=JBSWY3DPEHPK3PXP"}, f.err
}

func (f *fakeAccounts) ConfirmTOTP(ctx context.Context, id, code string) error {
	f.gotID, f.gotPass = id, code
	return f.err
}

func (f *fakeAccounts) VerifyMFA(ctx context.Context, token, code string) (User, Tokens, error) {
	f.gotRefresh, f.gotPass = token, code
	return f.user, f.tokens, f.err
}

func (f *fakeAccounts) Logout(ctx context.Context, refreshToken string) error {
	f.gotRefresh = refreshToken
	return f.err
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
		tokens: Tokens{Access: token.Access{Token: "signed.jwt.token", ExpiresAt: expires}, Refresh: "opaque-refresh", RefreshExpiresAt: expires.Add(720 * time.Hour)},
	}

	w := send(t, svc, http.MethodPost, "/auth/login", `{"email":"Reader@example.com","password":"12345678"}`, map[string]string{"X-Device-Name": "Firefox on Windows"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body)
	}
	if svc.gotEmail != "Reader@example.com" || svc.gotPass != "12345678" || svc.gotDevice != "Firefox on Windows" {
		t.Errorf("service got %q / %q / %q", svc.gotEmail, svc.gotPass, svc.gotDevice)
	}

	var body struct {
		AccessToken  string    `json:"accessToken"`
		TokenType    string    `json:"tokenType"`
		ExpiresAt    time.Time `json:"expiresAt"`
		RefreshToken string    `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.AccessToken != "signed.jwt.token" || body.TokenType != "Bearer" || !body.ExpiresAt.Equal(expires) || body.RefreshToken != "opaque-refresh" || body.User.ID != "u1" {
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

func TestRefreshHandler(t *testing.T) {

	svc := &fakeAccounts{user: User{ID: "u1"}, tokens: Tokens{Access: token.Access{Token: "new.jwt"}, Refresh: "next-refresh"}}

	w := send(t, svc, http.MethodPost, "/auth/refresh", `{"refreshToken":"old-refresh"}`, map[string]string{"X-Device-Name": "Safari on iOS"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body)
	}
	if svc.gotRefresh != "old-refresh" || svc.gotDevice != "Safari on iOS" {
		t.Errorf("service got %q / %q", svc.gotRefresh, svc.gotDevice)
	}
	if !strings.Contains(w.Body.String(), `"refreshToken":"next-refresh"`) || !strings.Contains(w.Body.String(), `"accessToken":"new.jwt"`) {
		t.Errorf("body = %s", w.Body)
	}
}

func TestRefreshHandlerRefused(t *testing.T) {

	w := send(t, &fakeAccounts{err: ErrInvalidRefreshToken}, http.MethodPost, "/auth/refresh", `{"refreshToken":"old"}`, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}

	w = send(t, &fakeAccounts{}, http.MethodPost, "/auth/refresh", `{}`, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("empty token: status = %d, want 400", w.Code)
	}
}

func TestLogoutHandler(t *testing.T) {

	svc := &fakeAccounts{}
	w := send(t, svc, http.MethodPost, "/auth/logout", `{"refreshToken":"the-refresh"}`, nil)

	if w.Code != http.StatusNoContent || svc.gotRefresh != "the-refresh" {
		t.Errorf("status = %d, service got %q", w.Code, svc.gotRefresh)
	}
}

func TestChangePasswordHandler(t *testing.T) {

	svc := &fakeAccounts{user: User{ID: "u1"}, tokens: Tokens{Access: token.Access{Token: "fresh.jwt"}, Refresh: "fresh-refresh"}}

	w := send(t, svc, http.MethodPost, "/me/password", `{"current":"old password","password":"new password"}`,
		map[string]string{"X-User-ID": "u1", "X-Device-Name": "Firefox on Windows"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body)
	}
	if svc.gotID != "u1" || svc.gotPass != "old password" || svc.gotNewPass != "new password" || svc.gotDevice != "Firefox on Windows" {
		t.Errorf("service got %q %q %q %q", svc.gotID, svc.gotPass, svc.gotNewPass, svc.gotDevice)
	}
	if !strings.Contains(w.Body.String(), `"refreshToken":"fresh-refresh"`) {
		t.Errorf("body = %s, want the new tokens", w.Body)
	}
}

func TestChangePasswordHandlerErrors(t *testing.T) {

	tests := []struct {
		name    string
		headers map[string]string
		err     error
		want    int
	}{
		{"signed out", nil, nil, http.StatusUnauthorized},
		{"wrong current password", map[string]string{"X-User-ID": "u1"}, ErrWrongPassword, http.StatusForbidden},
		{"short new password", map[string]string{"X-User-ID": "u1"}, fmt.Errorf("%w: password must be 8 to 128 characters", ErrInvalidInput), http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := send(t, &fakeAccounts{err: tt.err}, http.MethodPost, "/me/password", `{"current":"a","password":"b"}`, tt.headers)
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestVerifyEmailHandler(t *testing.T) {

	verified := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	svc := &fakeAccounts{user: User{ID: "u1", Email: "reader@example.com", EmailVerifiedAt: &verified}}

	w := send(t, svc, http.MethodPost, "/auth/verify-email", `{"token":"from-the-link"}`, nil)

	if w.Code != http.StatusOK || svc.gotRefresh != "from-the-link" {
		t.Fatalf("status = %d, token %q; body %s", w.Code, svc.gotRefresh, w.Body)
	}
	if !strings.Contains(w.Body.String(), `"emailVerifiedAt":"2026-10-16T09:00:00Z"`) {
		t.Errorf("body = %s", w.Body)
	}

	w = send(t, &fakeAccounts{err: ErrInvalidEmailToken}, http.MethodPost, "/auth/verify-email", `{"token":"old"}`, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("used link: status = %d, want 400", w.Code)
	}
}

func TestResendVerificationHandler(t *testing.T) {

	tests := []struct {
		name    string
		headers map[string]string
		err     error
		want    int
	}{
		{"sent", map[string]string{"X-User-ID": "u1"}, nil, http.StatusAccepted},
		{"signed out", nil, nil, http.StatusUnauthorized},
		{"already confirmed", map[string]string{"X-User-ID": "u1"}, ErrAlreadyVerified, http.StatusConflict},
		{"too soon", map[string]string{"X-User-ID": "u1"}, ErrTooSoon, http.StatusTooManyRequests},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := send(t, &fakeAccounts{err: tt.err}, http.MethodPost, "/me/verify-email", "", tt.headers)
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestForgotPasswordHandler(t *testing.T) {

	svc := &fakeAccounts{}
	w := send(t, svc, http.MethodPost, "/auth/forgot-password", `{"email":"reader@example.com"}`, nil)

	if w.Code != http.StatusAccepted || svc.gotEmail != "reader@example.com" {
		t.Errorf("status = %d, service got %q", w.Code, svc.gotEmail)
	}
}

func TestResetPasswordHandler(t *testing.T) {

	svc := &fakeAccounts{}
	w := send(t, svc, http.MethodPost, "/auth/reset-password", `{"token":"from-link","password":"a new password"}`, nil)
	if w.Code != http.StatusNoContent || svc.gotRefresh != "from-link" || svc.gotNewPass != "a new password" {
		t.Errorf("status = %d, service got %q %q", w.Code, svc.gotRefresh, svc.gotNewPass)
	}

	w = send(t, &fakeAccounts{err: ErrInvalidEmailToken}, http.MethodPost, "/auth/reset-password", `{"token":"used","password":"a new password"}`, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("used link: status = %d, want 400", w.Code)
	}
}

func TestLoginAsksForACode(t *testing.T) {

	svc := &fakeAccounts{err: &MFARequiredError{Token: "challenge", ExpiresAt: time.Date(2026, 10, 16, 9, 5, 0, 0, time.UTC)}}

	w := send(t, svc, http.MethodPost, "/auth/login", `{"email":"a@example.com","password":"right"}`, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body %s", w.Code, w.Body)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"mfaRequired":true`) || !strings.Contains(body, `"mfaToken":"challenge"`) || strings.Contains(body, "accessToken") {
		t.Errorf("body = %s, want the code step and no tokens", body)
	}
}

func TestTOTPHandlers(t *testing.T) {

	signedIn := map[string]string{"X-User-ID": "u1"}

	w := send(t, &fakeAccounts{}, http.MethodPost, "/me/mfa/totp/setup", "", signedIn)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"otpauthUrl":"otpauth://totp/`) {
		t.Errorf("setup: %d %s", w.Code, w.Body)
	}

	svc := &fakeAccounts{}
	w = send(t, svc, http.MethodPost, "/me/mfa/totp/confirm", `{"code":"123456"}`, signedIn)
	if w.Code != http.StatusNoContent || svc.gotPass != "123456" {
		t.Errorf("confirm: %d, code %q", w.Code, svc.gotPass)
	}

	w = send(t, &fakeAccounts{err: ErrWrongCode}, http.MethodPost, "/me/mfa/totp/confirm", `{"code":"000000"}`, signedIn)
	if w.Code != http.StatusBadRequest {
		t.Errorf("wrong code: %d, want 400", w.Code)
	}

	w = send(t, &fakeAccounts{}, http.MethodPost, "/me/mfa/totp/setup", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("setup signed out: %d, want 401", w.Code)
	}
}

func TestVerifyMFAHandler(t *testing.T) {

	svc := &fakeAccounts{user: User{ID: "u1"}, tokens: Tokens{Access: token.Access{Token: "jwt"}, Refresh: "r"}}
	w := send(t, svc, http.MethodPost, "/auth/mfa/verify", `{"mfaToken":"challenge","code":"123456"}`, nil)
	if w.Code != http.StatusOK || svc.gotRefresh != "challenge" || svc.gotPass != "123456" || !strings.Contains(w.Body.String(), `"accessToken":"jwt"`) {
		t.Errorf("verify: %d %s", w.Code, w.Body)
	}

	w = send(t, &fakeAccounts{err: ErrInvalidChallenge}, http.MethodPost, "/auth/mfa/verify", `{"mfaToken":"old","code":"1"}`, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expired step: %d, want 401", w.Code)
	}
}
