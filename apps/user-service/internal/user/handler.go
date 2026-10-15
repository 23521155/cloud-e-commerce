package user

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/fizzisme/user-service/internal/token"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// userIDHeader is the JWT subject, set by the gateway after it verified the
// token. This service is only reachable through the gateway, which strips
// the header from client requests.
const userIDHeader = "X-User-ID"

// maxBodyBytes bounds a request body; a name, an email and a password fit easily.
const maxBodyBytes = 16 << 10

// accounts is what the handler needs from the service; tests replace it
// with a fake.
type accounts interface {
	Register(ctx context.Context, in RegisterInput) (User, error)
	Login(ctx context.Context, email, password string) (User, token.Access, error)
	Me(ctx context.Context, id string) (User, error)
	UpdateProfile(ctx context.Context, id, name string, p Profile) (User, error)
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
	api.GET("/me", h.me)
	api.PUT("/me", h.updateMe)
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

// loginResponse carries the access token to send as Authorization: Bearer.
type loginResponse struct {
	AccessToken string    `json:"accessToken"`
	TokenType   string    `json:"tokenType"`
	ExpiresAt   time.Time `json:"expiresAt"`
	User        User      `json:"user"`
}

func (h *Handler) login(c *gin.Context) {

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	u, access, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, loginResponse{
		AccessToken: access.Token,
		TokenType:   "Bearer",
		ExpiresAt:   access.ExpiresAt,
		User:        u,
	})
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

	default:
		h.log.Error("cannot handle account request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
