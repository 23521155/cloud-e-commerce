package order

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/fizzisme/order-service/internal/catalogclient"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// idempotencyHeader carries the key the web made for one checkout.
const idempotencyHeader = "Idempotency-Key"

// maxBodyBytes bounds a request body; 20 books and an address fit easily.
const maxBodyBytes = 64 << 10

// userIDHeader is the signed-in customer, set by the gateway from a verified
// JWT and stripped from client requests. Absent for a guest.
const userIDHeader = "X-User-ID"

// creator is what the handler needs from the service; tests replace it with
// a fake.
type creator interface {
	Create(ctx context.Context, in CreateInput) (Order, bool, error)
	ListMine(ctx context.Context, userID string) ([]Order, error)
	Lookup(ctx context.Context, code, email string) (Order, error)
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
	api.GET("/", h.listMine)
	api.POST("/lookup", h.lookup)
}

// lookupRequest is the body of POST /api/orders/lookup. It is a POST so the
// email stays out of URLs and access logs.
type lookupRequest struct {
	Code  string `json:"code"`
	Email string `json:"email"`
}

// lookup answers POST /api/orders/lookup: a guest's order, found by its code
// and the email it was placed with.
func (h *Handler) lookup(c *gin.Context) {

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req lookupRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Email) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order code and email are required"})
		return
	}

	order, err := h.service.Lookup(c.Request.Context(), req.Code, req.Email)
	switch {
	case errors.Is(err, ErrNotFound):
		// One answer for an unknown code and a wrong email.
		c.JSON(http.StatusNotFound, gin.H{"error": "no order with this code and email"})
	case err != nil:
		h.log.Error("cannot look up order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	default:
		c.JSON(http.StatusOK, order)
	}
}

// listMine answers GET /api/orders with the signed-in customer's orders.
// Guests look an order up by code and email instead.
func (h *Handler) listMine(c *gin.Context) {

	userID := c.GetHeader(userIDHeader)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in required"})
		return
	}

	orders, err := h.service.ListMine(c.Request.Context(), userID)
	if err != nil {
		h.log.Error("cannot list orders", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": orders})
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
		UserID:        optional(c.GetHeader(userIDHeader)),
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

// optional returns nil for an empty string.
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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
