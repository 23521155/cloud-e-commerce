package subject

import (
	"context"
	"net/http"

	"github.com/fizzisme/catalog-service/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// lister is what the handler needs from the repository; tests replace it
// with a fake.
type lister interface {
	List(ctx context.Context) ([]Subject, error)
}

type Handler struct {
	subjects lister
}

func NewHandler(subjects lister) *Handler {
	return &Handler{subjects: subjects}
}

// Register mounts the subject routes on the /api/v1 group.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/subjects", h.list)
}

func (h *Handler) list(c *gin.Context) {

	subjects, err := h.subjects.List(c.Request.Context())
	if err != nil {
		logger.Log.Error(
			"request failed",
			zap.String("path", c.Request.URL.Path),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": subjects})
}
