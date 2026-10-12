package order

import (
	"context"
	"errors"
	"net/http"

	"github.com/fizzisme/order-service/internal/catalogclient"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// idempotencyHeader carries the key the web made for one checkout.
const idempotencyHeader = "Idempotency-Key"

// maxBodyBytes bounds a request body; 20 books and an address fit easily.
const maxBodyBytes = 64 << 10

// creator is what the handler needs from the service; tests replace it with
// a fake.
type creator interface {
	Create(ctx context.Context, in CreateInput) (Order, bool, error)
}

type Handler struct {
	service creator
	log     *zap.Logger
}

func NewHandler(service creator, log *zap.Logger) *Handler {
	return &Handler{service: service, log: log}
}

// Register mounts the order routes. The gateway strips /api/orders, so
// POST /api/orders arrives here as POST /.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.POST("/", h.create)
}

// createRequest is the body of POST /api/orders. Fields are checked by the
// service, so a bad value gets a message naming it.
type createRequest struct {
	Items []struct {
		Slug string `json:"slug"`
	} `json:"items"`

	PaymentMethod string  `json:"paymentMethod"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	FullName      string  `json:"fullName"`
	Address       string  `json:"address"`
	City          string  `json:"city"`
	PostalCode    *string `json:"postalCode"`
	Country       string  `json:"country"`
}

// unavailableBook is one book that stopped an order. Ids are not part of the
// public API, so only the slug is shown.
type unavailableBook struct {
	Slug   string `json:"slug"`
	Status string `json:"status"`
}

func (h *Handler) create(c *gin.Context) {

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	slugs := make([]string, len(req.Items))
	for i, it := range req.Items {
		slugs[i] = it.Slug
	}

	order, replayed, err := h.service.Create(c.Request.Context(), CreateInput{
		Key:           c.GetHeader(idempotencyHeader),
		Slugs:         slugs,
		PaymentMethod: req.PaymentMethod,
		Email:         req.Email,
		Phone:         req.Phone,
		FullName:      req.FullName,
		Address:       req.Address,
		City:          req.City,
		PostalCode:    req.PostalCode,
		Country:       req.Country,
		ClientIP:      c.ClientIP(),
	})
	if err != nil {
		h.fail(c, err)
		return
	}

	// A replay answers with the order the key already placed.
	if replayed {
		c.JSON(http.StatusOK, order)
		return
	}

	c.JSON(http.StatusCreated, order)
}

// fail turns a service error into a response. It is the one place that
// knows which error means which status.
func (h *Handler) fail(c *gin.Context, err error) {

	var unavailable *catalogclient.UnavailableError

	switch {
	case errors.Is(err, ErrInvalidInput):
		// The message names the field and is written for the customer.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	case errors.Is(err, ErrKeyReused):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "idempotency key already used for another order"})

	case errors.As(err, &unavailable):
		books := make([]unavailableBook, len(unavailable.Books))
		for i, b := range unavailable.Books {
			books[i] = unavailableBook{Slug: b.Slug, Status: b.Status}
		}
		c.JSON(http.StatusConflict, gin.H{"error": "some books are no longer available", "books": books})

	case errors.Is(err, catalogclient.ErrCatalogUnavailable), errors.Is(err, ErrNoFreeCode):
		h.log.Warn("cannot place order now", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "cannot place the order right now, please try again"})

	default:
		h.log.Error("cannot place order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
