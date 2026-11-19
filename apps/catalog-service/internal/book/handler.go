package book

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/fizzisme/catalog-service/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// service is what the handler needs from the service layer; tests replace
// it with a fake.
type service interface {
	List(ctx context.Context, in ListInput) (Page, error)
	Get(ctx context.Context, slug string) (Book, error)
	Related(ctx context.Context, slug string, limit int) ([]Book, error)
	BySlugs(ctx context.Context, slugs []string) ([]Book, error)
}

type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{service: service}
}

// Register mounts the book routes on the API group.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/books", h.list)
	api.GET("/books/:slug", h.get)
	api.GET("/books/:slug/related", h.related)
}

// listQuery is the query string of GET /books. Gin validates it, so the
// sort value is always one the repository knows.
type listQuery struct {
	Subject string `form:"subject"`
	Rare    bool   `form:"rare"`
	Sort    string `form:"sort" binding:"omitempty,oneof=new price-asc price-desc year"`
	Q       string `form:"q"    binding:"max=100"`
	Page    int    `form:"page" binding:"omitempty,min=1,max=100000"`

	// Slugs is a comma-separated list. When present the other filters are
	// ignored and the listing is replaced by those books (see bySlugs).
	Slugs string `form:"slugs" binding:"max=4000"`
}

type relatedQuery struct {
	Limit int `form:"limit" binding:"omitempty,min=1"`
}

func (h *Handler) list(c *gin.Context) {

	var q listQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		badRequest(c)
		return
	}

	if q.Slugs != "" {
		h.bySlugs(c, q.Slugs)
		return
	}

	page, err := h.service.List(c.Request.Context(), ListInput{
		Subject: q.Subject,
		Rare:    q.Rare,
		Sort:    q.Sort,
		Q:       q.Q,
		Page:    q.Page,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, page)
}

// bySlugs answers GET /books?slugs=a,b,c, used to show the books of a basket,
// order or wishlist with a single request. The books come in the order asked.
func (h *Handler) bySlugs(c *gin.Context, raw string) {

	slugs := strings.Split(raw, ",")
	if len(slugs) > MaxBatch {
		badRequest(c)
		return
	}
	for i := range slugs {
		slugs[i] = strings.TrimSpace(slugs[i])
	}

	books, err := h.service.BySlugs(c.Request.Context(), slugs)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, Page{
		Items:    books,
		Total:    len(books),
		Page:     1,
		PageSize: len(books),
	})
}

func (h *Handler) get(c *gin.Context) {

	b, err := h.service.Get(c.Request.Context(), c.Param("slug"))
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, b)
}

func (h *Handler) related(c *gin.Context) {

	var q relatedQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		badRequest(c)
		return
	}

	books, err := h.service.Related(c.Request.Context(), c.Param("slug"), q.Limit)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": books})
}

func badRequest(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
}

// respondError maps service errors to HTTP. Anything unexpected is logged
// and answered with a generic 500, so SQL errors never reach the client.
func respondError(c *gin.Context, err error) {

	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	logger.Log.Error(
		"request failed",
		zap.String("path", c.Request.URL.Path),
		zap.Error(err),
	)

	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
