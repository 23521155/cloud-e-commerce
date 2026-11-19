package order

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/fizzisme/order-service/internal/payment/vnpay"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// payer is what the payment handler needs from the service; tests replace it
// with a fake.
type payer interface {
	ConfirmPayment(ctx context.Context, r vnpay.Result) (Order, error)
	FindByCode(ctx context.Context, code string) (Order, error)
}

// verifier checks the signature of what the gateway sent. The real one is
// *vnpay.Client.
type verifier interface {
	Verify(query url.Values) (vnpay.Result, error)
}

var (
	_ payer    = (*Service)(nil)
	_ verifier = (*vnpay.Client)(nil)
)

// Answer codes of the VNPay IPN call. VNPay calls again until it gets 00, 01,
// 02, 04 or 97.
const (
	ipnConfirmed        = "00"
	ipnOrderNotFound    = "01"
	ipnAlreadyConfirmed = "02"
	ipnInvalidAmount    = "04"
	ipnInvalidSignature = "97"
	ipnUnknownError     = "99"
)

// PaymentHandler serves the two ways VNPay reports a payment: the IPN call,
// server to server, and the return redirect the web forwards. Mounted only
// when a VNPay account is configured.
type PaymentHandler struct {
	service payer
	gateway verifier
	log     *zap.Logger

	// confirmOnReturn lets the return redirect record the payment too, for
	// local runs that VNPay's IPN call cannot reach.
	confirmOnReturn bool
}

func NewPaymentHandler(service payer, gateway verifier, confirmOnReturn bool, log *zap.Logger) *PaymentHandler {
	return &PaymentHandler{service: service, gateway: gateway, confirmOnReturn: confirmOnReturn, log: log}
}

// Register mounts the payment routes. The gateway strips /api/orders, so
// GET /api/orders/payments/vnpay/ipn arrives here as /payments/vnpay/ipn.
func (h *PaymentHandler) Register(api *gin.RouterGroup) {
	api.GET("/payments/vnpay/ipn", h.ipn)
	api.GET("/payments/vnpay/return", h.vnpayReturn)
}

// ipn answers VNPay's server-to-server call. The answer is always HTTP 200
// with a JSON body: VNPay reads only RspCode.
func (h *PaymentHandler) ipn(c *gin.Context) {

	code, message := h.confirm(c)

	h.log.Info("vnpay ipn",
		zap.String("txn_ref", c.Query("vnp_TxnRef")),
		zap.String("response_code", c.Query("vnp_ResponseCode")),
		zap.String("answer", code))

	c.JSON(http.StatusOK, gin.H{"RspCode": code, "Message": message})
}

// confirm verifies the IPN call, records the payment and picks the answer.
func (h *PaymentHandler) confirm(c *gin.Context) (code, message string) {

	r, err := h.gateway.Verify(c.Request.URL.Query())
	switch {
	case errors.Is(err, vnpay.ErrInvalidSignature):
		h.log.Warn("vnpay ipn with an invalid signature", zap.String("ip", c.ClientIP()))
		return ipnInvalidSignature, "Invalid signature"
	case err != nil:
		h.log.Warn("unreadable vnpay ipn", zap.Error(err))
		return ipnUnknownError, "Unknown error"
	}

	_, err = h.service.ConfirmPayment(c.Request.Context(), r)
	switch {
	case err == nil:
		return ipnConfirmed, "Confirm Success"

	case errors.Is(err, ErrPaidTooLate):
		// Recorded: the order is cancelled for a refund. Calling again
		// would change nothing.
		return ipnConfirmed, "Confirm Success"

	case errors.Is(err, ErrNotFound):
		return ipnOrderNotFound, "Order not found"

	case errors.Is(err, ErrAlreadyProcessed):
		return ipnAlreadyConfirmed, "Order already confirmed"

	case errors.Is(err, ErrAmountMismatch):
		h.log.Warn("vnpay ipn with a wrong amount",
			zap.String("txn_ref", r.TxnRef), zap.Int64("amount_vnd", r.AmountVND))
		return ipnInvalidAmount, "Invalid amount"
	}

	// Nothing was recorded: VNPay calls again later.
	h.log.Error("cannot record a vnpay payment", zap.String("txn_ref", r.TxnRef), zap.Error(err))
	return ipnUnknownError, "Unknown error"
}

// returnResponse is the body of GET /payments/vnpay/return.
type returnResponse struct {
	Order Order `json:"order"`

	// PaymentSucceeded is what VNPay told the customer. The order may still
	// be AWAITING_PAYMENT while the IPN call has not arrived; the web then
	// asks again.
	PaymentSucceeded bool `json:"paymentSucceeded"`
}

// vnpayReturn answers the web, which forwards the query string VNPay put on
// the customer's return redirect. It shows the order as it is; it records
// the payment only when confirmOnReturn is set.
func (h *PaymentHandler) vnpayReturn(c *gin.Context) {

	r, err := h.gateway.Verify(c.Request.URL.Query())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment result"})
		return
	}

	if h.confirmOnReturn {
		_, err := h.service.ConfirmPayment(c.Request.Context(), r)
		switch {
		case err == nil, errors.Is(err, ErrAlreadyProcessed), errors.Is(err, ErrPaidTooLate),
			errors.Is(err, ErrNotFound):
			// Shown below as the order now is.
		case errors.Is(err, ErrAmountMismatch):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment result"})
			return
		default:
			// The order is shown as it is; the IPN call can still record it.
			h.log.Warn("cannot record a vnpay payment on return",
				zap.String("txn_ref", r.TxnRef), zap.Error(err))
		}
	}

	o, err := h.service.FindByCode(c.Request.Context(), r.TxnRef)
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	case err != nil:
		h.log.Error("cannot read an order on vnpay return", zap.String("txn_ref", r.TxnRef), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, returnResponse{Order: o, PaymentSucceeded: r.Succeeded()})
}
