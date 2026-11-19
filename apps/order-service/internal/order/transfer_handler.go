package order

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/fizzisme/order-service/internal/payment/sepay"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// transferPayer is what the transfer handler needs from the service; tests
// replace it with a fake.
type transferPayer interface {
	ConfirmTransfer(ctx context.Context, code string, amountVND int64, ref string) (Order, error)
}

// transferGateway checks and reads what SePay sent. The real one is
// *sepay.Client.
type transferGateway interface {
	Authenticate(authorization string) error
	Accepts(w sepay.Webhook) bool
	OrderCode(w sepay.Webhook) (string, bool)
}

var (
	_ transferPayer   = (*Service)(nil)
	_ transferGateway = (*sepay.Client)(nil)
)

// TransferHandler serves the webhook SePay calls when money arrives in the
// shop's bank account. Mounted only when SePay is configured.
type TransferHandler struct {
	service transferPayer
	gateway transferGateway
	log     *zap.Logger
}

func NewTransferHandler(service transferPayer, gateway transferGateway, log *zap.Logger) *TransferHandler {
	return &TransferHandler{service: service, gateway: gateway, log: log}
}

// Register mounts the route. The gateway strips /api/orders, so
// POST /api/orders/payments/sepay/webhook arrives here as
// /payments/sepay/webhook.
func (h *TransferHandler) Register(api *gin.RouterGroup) {
	api.POST("/payments/sepay/webhook", h.webhook)
}

// webhook answers SePay. SePay repeats a call until it gets HTTP 200 or 201
// with {"success": true} (up to 7 times in 5 hours), so:
//   - what is settled, or cannot be settled by repeating, is answered with
//     success, and logged when the owner has to act;
//   - only a failure that may pass (the catalogue or the database is down) is
//     answered with 500, so SePay calls again.
//
// The transfer text holds the sender's account and phone number, so it is
// never logged.
func (h *TransferHandler) webhook(c *gin.Context) {

	if err := h.gateway.Authenticate(c.GetHeader("Authorization")); err != nil {
		h.log.Warn("sepay webhook without a valid key", zap.String("ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"success": false})
		return
	}

	var w sepay.Webhook
	if err := c.ShouldBindJSON(&w); err != nil {
		h.log.Warn("unreadable sepay webhook", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"success": false})
		return
	}

	// Money going out, or coming in on another account: not ours.
	if !h.gateway.Accepts(w) {
		h.log.Info("sepay webhook ignored",
			zap.Int64("id", w.ID), zap.String("type", w.TransferType), zap.String("va", w.SubAccount))
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	code, ok := h.gateway.OrderCode(w)
	if !ok {
		// Repeating the call would not make a code appear.
		h.log.Warn("transfer without an order code, settle by hand",
			zap.Int64("id", w.ID), zap.Int64("amount_vnd", w.AmountVND), zap.String("reference", w.ReferenceCode))
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	ref := "sepay:" + strconv.FormatInt(w.ID, 10)

	_, err := h.service.ConfirmTransfer(c.Request.Context(), code, w.AmountVND, ref)
	switch {
	case err == nil:
		h.log.Info("transfer recorded", zap.String("code", code), zap.String("ref", ref))

	case errors.Is(err, ErrAlreadyProcessed):
		// Normal for a repeated call.
		h.log.Info("transfer already recorded", zap.String("code", code), zap.String("ref", ref))

	case errors.Is(err, ErrPaidTooLate):
		// Recorded by the service: the order is cancelled for a refund.
		h.log.Warn("transfer arrived too late, refund by hand", zap.String("code", code), zap.String("ref", ref))

	case errors.Is(err, ErrNotFound):
		h.log.Warn("transfer for an unknown order, settle by hand",
			zap.String("code", code), zap.String("ref", ref), zap.Int64("amount_vnd", w.AmountVND))

	case errors.Is(err, ErrAmountMismatch):
		h.log.Warn("transfer with a wrong amount, settle by hand",
			zap.String("code", code), zap.String("ref", ref), zap.Int64("amount_vnd", w.AmountVND))

	default:
		// Nothing was recorded: SePay calls again later.
		h.log.Error("cannot record a transfer", zap.String("code", code), zap.String("ref", ref), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
