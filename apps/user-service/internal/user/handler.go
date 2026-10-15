package user

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// userIDHeader is the JWT subject, set by the gateway after it verified the
// token. This service is only reachable through the gateway, which strips
// the header from client requests.
const userIDHeader = "X-User-ID"

// deviceHeader is the browser and OS the gateway reads from the User-Agent,
// kept with each session.
const deviceHeader = "X-Device-Name"

// maxBodyBytes bounds a request body; a name, an email and a password fit easily.
const maxBodyBytes = 16 << 10

// accounts is what the handler needs from the service; tests replace it
// with a fake.
type accounts interface {
	Register(ctx context.Context, in RegisterInput) (User, error)
	Login(ctx context.Context, email, password, device string) (User, Tokens, error)
	Refresh(ctx context.Context, refreshToken, device string) (User, Tokens, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, id string) (User, error)
	UpdateProfile(ctx context.Context, id, name string, p Profile) (User, error)
	ChangePassword(ctx context.Context, id, current, password, device string) (User, Tokens, error)
	VerifyEmail(ctx context.Context, token string) (User, error)
	ResendVerification(ctx context.Context, id string) error
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, password string) error
	SetupTOTP(ctx context.Context, id string) (TOTPSetup, error)
	ConfirmTOTP(ctx context.Context, id, code string) error
	VerifyMFA(ctx context.Context, token, code string) (User, Tokens, error)
}

type Handler struct {
	service accounts
	log     *zap.Logger
}

func NewHandler(service accounts, log *zap.Logger) *Handler {
	return &Handler{service: service, log: log}
}

// Register mounts the account routes. The gateway strips /api/users, so
// POST /api/users/auth/register arrives here as POST /auth/register.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.POST("/auth/register", h.register)
	api.POST("/auth/login", h.login)
	api.POST("/auth/refresh", h.refresh)
	api.POST("/auth/logout", h.logout)
	api.GET("/me", h.me)
	api.PUT("/me", h.updateMe)
	api.POST("/me/password", h.changePassword)
	api.POST("/auth/verify-email", h.verifyEmail)
	api.POST("/me/verify-email", h.resendVerification)
	api.POST("/auth/forgot-password", h.forgotPassword)
	api.POST("/auth/reset-password", h.resetPassword)
	api.POST("/me/mfa/totp/setup", h.setupTOTP)
	api.POST("/me/mfa/totp/confirm", h.confirmTOTP)
	api.POST("/auth/mfa/verify", h.verifyMFA)
}

// registerRequest is the body of POST /auth/register. Fields are checked by
// the service, so a bad value gets a message naming it.
type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) register(c *gin.Context) {

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	u, err := h.service.Register(c.Request.Context(), RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.fail(c, err)
		return
	}

	c.JSON(http.StatusCreated, u)
}

// loginRequest is the body of POST /auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginResponse carries the access token to send as Authorization: Bearer,
// and the refresh token that gets the next one. Refresh answers the same.
type loginResponse struct {
	AccessToken      string    `json:"accessToken"`
	TokenType        string    `json:"tokenType"`
	ExpiresAt        time.Time `json:"expiresAt"`
	RefreshToken     string    `json:"refreshToken"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
	User             User      `json:"user"`
}

func newLoginResponse(u User, t Tokens) loginResponse {
	return loginResponse{
		AccessToken:      t.Access.Token,
		TokenType:        "Bearer",
		ExpiresAt:        t.Access.ExpiresAt,
		RefreshToken:     t.Refresh,
		RefreshExpiresAt: t.RefreshExpiresAt,
		User:             u,
	}
}

func (h *Handler) login(c *gin.Context) {

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	u, tokens, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.GetHeader(deviceHeader))

	// Right password, and a code is due: no tokens yet.
	var mfa *MFARequiredError
	if errors.As(err, &mfa) {
		c.JSON(http.StatusOK, gin.H{"mfaRequired": true, "mfaToken": mfa.Token, "mfaExpiresAt": mfa.ExpiresAt})
		return
	}
	if err != nil {
		h.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, newLoginResponse(u, tokens))
}

// refreshRequest is the body of POST /auth/refresh and POST /auth/logout.
type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (h *Handler) refresh(c *gin.Context) {

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	u, tokens, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.GetHeader(deviceHeader))
	if err != nil {
		h.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, newLoginResponse(u, tokens))
}

// logout ends the session of a refresh token. It always answers 204: a
// token that is not valid has no session to end.
func (h *Handler) logout(c *gin.Context) {

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// me answers GET /me with the signed-in account.
func (h *Handler) me(c *gin.Context) {

	id := c.GetHeader(userIDHeader)
	if id == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in required"})
		return
	}

	u, err := h.service.Me(c.Request.Context(), id)
	if errors.Is(err, ErrNotFound) {
		// A valid token for an account that no longer exists.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in required"})
		return
	}
	if err != nil {
		h.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, u)
}

// updateMeRequest is the body of PUT /me: the whole editable profile. The
// email is not part of it: changing it needs the new address confirmed.
type updateMeRequest struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Address    string `json:"address"`
	City       string `json:"city"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

func (h *Handler) updateMe(c *gin.Context) {

	id := c.GetHeader(userIDHeader)
	if id == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in required"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	u, err := h.service.UpdateProfile(c.Request.Context(), id, req.Name, Profile{
		Phone:      req.Phone,
		Address:    req.Address,
		City:       req.City,
		PostalCode: req.PostalCode,
		Country:    req.Country,
	})
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in required"})
		return
	}
	if err != nil {
		h.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, u)
}

// changePasswordRequest is the body of POST /me/password.
type changePasswordRequest struct {
	Current  string `json:"current"`
	Password string `json:"password"`
}

// changePassword answers with new tokens like a sign-in: every earlier
// session of the account has ended.
func (h *Handler) changePassword(c *gin.Context) {

	id := c.GetHeader(userIDHeader)
	if id == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in required"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	u, tokens, err := h.service.ChangePassword(c.Request.Context(), id, req.Current, req.Password, c.GetHeader(deviceHeader))
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in required"})
		return
	}
	if err != nil {
		h.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, newLoginResponse(u, tokens))
}

// verifyEmailRequest is the body of POST /auth/verify-email: the token from
// the link in the email. No sign-in is needed: the token proves the reader
// holds the mailbox.
type verifyEmailRequest struct {
	Token string `json:"token"`
}

func (h *Handler) verifyEmail(c *gin.Context) {

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	u, err := h.service.VerifyEmail(c.Request.Context(), req.Token)
	if err != nil {
		h.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"email": u.Email, "emailVerifiedAt": u.EmailVerifiedAt})
}

// resendVerification answers POST /me/verify-email: a new confirmation
// link to the signed-in account's address.
func (h *Handler) resendVerification(c *gin.Context) {

	id := c.GetHeader(userIDHeader)
	if id == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in required"})
		return
	}

	err := h.service.ResendVerification(c.Request.Context(), id)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in required"})
		return
	}
	if err != nil {
		h.fail(c, err)
		return
	}

	c.Status(http.StatusAccepted)
}

// forgotPasswordRequest is the body of POST /auth/forgot-password.
type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// forgotPassword always answers 202, account or not, so it cannot be used
// to find out who has an account.
func (h *Handler) forgotPassword(c *gin.Context) {

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		h.fail(c, err)
		return
	}

	c.Status(http.StatusAccepted)
}

// resetPasswordRequest is the body of POST /auth/reset-password: the token
// from the emailed link and the new password.
type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *Handler) resetPassword(c *gin.Context) {

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		h.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// setupTOTP answers POST /me/mfa/totp/setup with a new secret for the
// reader's authenticator app.
func (h *Handler) setupTOTP(c *gin.Context) {

	id := c.GetHeader(userIDHeader)
	if id == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in required"})
		return
	}

	setup, err := h.service.SetupTOTP(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"secret": setup.Secret, "otpauthUrl": setup.URL})
}

// codeRequest is the body of POST /me/mfa/totp/confirm and, with the token
// from the sign-in, of POST /auth/mfa/verify.
type codeRequest struct {
	MFAToken string `json:"mfaToken"`
	Code     string `json:"code"`
}

func (h *Handler) confirmTOTP(c *gin.Context) {

	id := c.GetHeader(userIDHeader)
	if id == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in required"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req codeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.service.ConfirmTOTP(c.Request.Context(), id, req.Code); err != nil {
		h.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) verifyMFA(c *gin.Context) {

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req codeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.MFAToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	u, tokens, err := h.service.VerifyMFA(c.Request.Context(), req.MFAToken, req.Code)
	if err != nil {
		h.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, newLoginResponse(u, tokens))
}

// fail turns a service error into a response. It is the one place that
// knows which error means which status.
func (h *Handler) fail(c *gin.Context, err error) {

	switch {
	case errors.Is(err, ErrInvalidInput):
		// The message names the field and is written for the user.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	case errors.Is(err, ErrEmailTaken):
		c.JSON(http.StatusConflict, gin.H{"error": "an account with this email already exists"})

	case errors.Is(err, ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email or password is incorrect"})

	case errors.Is(err, ErrWrongPassword):
		// 403, not 401: the reader is signed in, the confirmation is wrong.
		c.JSON(http.StatusForbidden, gin.H{"error": "current password is incorrect"})

	case errors.Is(err, ErrInvalidEmailToken):
		c.JSON(http.StatusBadRequest, gin.H{"error": "this link has been used or has expired"})

	case errors.Is(err, ErrAlreadyVerified):
		c.JSON(http.StatusConflict, gin.H{"error": "the email address is already confirmed"})

	case errors.Is(err, ErrTooSoon):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "an email was sent a moment ago, please wait a minute"})

	case errors.Is(err, ErrWrongCode):
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is incorrect"})

	case errors.Is(err, ErrInvalidChallenge):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in again"})

	case errors.Is(err, ErrMFAAlreadyOn), errors.Is(err, ErrMFANotSetUp):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	case errors.Is(err, ErrMFAUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})

	case errors.Is(err, ErrInvalidRefreshToken):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in required"})

	default:
		h.log.Error("cannot handle account request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
