package sell

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/fizzisme/catalog-service/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.uber.org/zap"
)

type creator interface {
	Create(ctx context.Context, in CreateInput) (Request, error)
}

type Handler struct {
	service creator
}

func NewHandler(service creator) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(api *gin.RouterGroup) {
	// The gateway will strip the /api/sell prefix.
	api.POST("/requests", h.create)
}

func (h *Handler) create(c *gin.Context) {
	if c.ContentType() != "multipart/form-data" {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"error": "expected multipart/form-data",
		})
		return
	}

	// Text-only limit until photo storage is implemented.
	const maxBodyBytes = 64 * 1024
	c.Request.Body = http.MaxBytesReader(
		c.Writer, c.Request.Body, maxBodyBytes,
	)

	err := c.Request.ParseMultipartForm(maxBodyBytes)

	// Clean up temporary files even if parsing failed.
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}

	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "request body is too large",
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid multipart form",
		})
		return
	}

	form := c.Request.MultipartForm

	// Do not accept photos until they can be stored.
	if len(form.File) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "photo uploads are not available yet",
		})
		return
	}

	for key, values := range form.Value {
		if len(values) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "each field must have a single value",
			})
			return
		}

		form.Value[key][0] = strings.TrimSpace(values[0])
	}

	// Empty optional numbers must remain nil instead of becoming zero.
	for _, key := range []string{"year", "asking"} {
		if values := form.Value[key]; len(values) == 1 && values[0] == "" {
			delete(form.Value, key)
		}
	}

	var in CreateInput
	if err := c.ShouldBindWith(&in, binding.FormMultipart); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid form fields",
		})
		return
	}

	request, err := h.service.Create(c.Request.Context(), in)
	if errors.Is(err, ErrInvalidInput) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if err != nil {
		logger.Log.Error(
			"cannot create sell request",
			zap.String("request_id", c.GetHeader("X-Request-ID")),
			zap.Error(err),
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        request.ID,
		"status":    request.Status,
		"createdAt": request.CreatedAt,
	})
}