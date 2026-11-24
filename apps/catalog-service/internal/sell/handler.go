package sell

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/fizzisme/catalog-service/internal/blobstore"
	"github.com/fizzisme/catalog-service/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.uber.org/zap"
)

type service interface {
	Create(ctx context.Context, in CreateInput, uploads []Upload) (Request, error)
	List(ctx context.Context) ([]Request, error)
	OpenPhoto(ctx context.Context, requestID int64, position int) (io.ReadCloser, string, int64, error)
}

type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(api *gin.RouterGroup) {
	// The gateway will strip the /api/sell prefix.
	api.POST("/requests", h.create)
	api.GET("/requests", requireStaff, h.list)
	api.GET("/requests/:id/photos/:position", requireStaff, h.photo)
}

const headerUserRoles = "X-User-Roles"

func requireStaff(c *gin.Context) {
	for _, role := range strings.Split(c.GetHeader(headerUserRoles), ",") {
		if strings.TrimSpace(role) == "staff" {
			c.Next()
			return
		}
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "staff only"})
}

func (h *Handler) list(c *gin.Context) {
	requests, err := h.service.List(c.Request.Context())
	if err != nil {
		logger.Log.Error("cannot list sell requests", zap.String("request_id", c.GetHeader("X-Request-ID")), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	if requests == nil {
		requests = []Request{}
	}
	c.JSON(http.StatusOK, gin.H{"items": requests})
}

func (h *Handler) photo(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	position, posErr := strconv.Atoi(c.Param("position"))
	if err != nil || posErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "photo not found"})
		return
	}

	body, contentType, size, err := h.service.OpenPhoto(c.Request.Context(), id, position)
	if errors.Is(err, ErrPhotoNotFound) || errors.Is(err, blobstore.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "photo not found"})
		return
	}
	if err != nil {
		logger.Log.Error("cannot read sell photo", zap.Int64("sell_request_id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	defer body.Close()

	c.Header("Cache-Control", "private, max-age=3600")
	c.DataFromReader(http.StatusOK, size, contentType, body, nil)
}

func (h *Handler) create(c *gin.Context) {
	if c.ContentType() != "multipart/form-data" {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"error": "expected multipart/form-data",
		})
		return
	}

	const maxBodyBytes = MaxPhotos*MaxPhotoBytes + 64*1024
	c.Request.Body = http.MaxBytesReader(
		c.Writer, c.Request.Body, maxBodyBytes,
	)

	err := c.Request.ParseMultipartForm(8 << 20)

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

	for key := range form.File {
		if key != "photos" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unexpected file field"})
			return
		}
	}

	uploads, err := Uploads(form.File["photos"])
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrPhotoTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		c.JSON(status, gin.H{"error": err.Error()})
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

	request, err := h.service.Create(c.Request.Context(), in, uploads)
	if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrPhotosDisabled) {
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
		"photos":    len(uploads),
	})
}