package user

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// maxBodyBytes bounds a request body; a name, an email and a password fit easily.
const maxBodyBytes = 16 << 10

// registrar is what the handler needs from the service; tests replace it
// with a fake.
type registrar interface {
	Register(ctx context.Context, in RegisterInput) (User, error)
}

type Handler struct {
	service registrar
	log     *zap.Logger
}

func NewHandler(service registrar, log *zap.Logger) *Handler {
	return &Handler{service: service, log: log}
}

// Register mounts the account routes. The gateway strips /api/users, so
// POST /api/users/auth/register arrives here as POST /auth/register.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.POST("/auth/register", h.register)
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

// fail turns a service error into a response. It is the one place that
// knows which error means which status.
func (h *Handler) fail(c *gin.Context, err error) {

	switch {
	case errors.Is(err, ErrInvalidInput):
		// The message names the field and is written for the user.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	case errors.Is(err, ErrEmailTaken):
		c.JSON(http.StatusConflict, gin.H{"error": "an account with this email already exists"})

	default:
		h.log.Error("cannot handle account request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
