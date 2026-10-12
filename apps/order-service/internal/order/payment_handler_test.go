package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/fizzisme/order-service/internal/catalogclient"
	"github.com/fizzisme/order-service/internal/payment/vnpay"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// fakeVerifier answers Verify with what the test set and keeps the query.
type fakeVerifier struct {
	result vnpay.Result
	err    error

	got url.Values
}

func (f *fakeVerifier) Verify(query url.Values) (vnpay.Result, error) {
	f.got = query
	return f.result, f.err
}

// fakePayer answers with what the test set and counts the confirmations.
type fakePayer struct {
	order      Order
	confirmErr error
	findErr    error

	confirmed int
}

func (f *fakePayer) ConfirmPayment(ctx context.Context, r vnpay.Result) (Order, error) {
	f.confirmed++
	return f.order, f.confirmErr
}

func (f *fakePayer) FindByCode(ctx context.Context, code string) (Order, error) {
	return f.order, f.findErr
}

// paidResult is a verified, successful payment of order MA-048291.
var paidResult = vnpay.Result{TxnRef: "MA-048291", AmountVND: 200000, ResponseCode: "00", TransactionStatus: "00"}

// getPayment sends GET path to a PaymentHandler and returns the response.
func getPayment(t *testing.T, svc payer, gateway verifier, confirmOnReturn bool, path string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewPaymentHandler(svc, gateway, confirmOnReturn, zap.NewNop()).Register(r.Group(""))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))

	return w
}

func TestIPNAnswers(t *testing.T) {

	tests := []struct {
		name       string
		verifyErr  error
		confirmErr error
		want       string
	}{
		{"recorded", nil, nil, ipnConfirmed},
		{"paid too late is recorded too", nil, ErrPaidTooLate, ipnConfirmed},
		{"unknown order", nil, ErrNotFound, ipnOrderNotFound},
		{"already processed", nil, ErrAlreadyProcessed, ipnAlreadyConfirmed},
		{"wrong amount", nil, ErrAmountMismatch, ipnInvalidAmount},
		{"bad signature", vnpay.ErrInvalidSignature, nil, ipnInvalidSignature},
		{"unreadable", errors.New("vnpay: invalid vnp_Amount"), nil, ipnUnknownError},
		{"catalogue down", nil, catalogclient.ErrCatalogUnavailable, ipnUnknownError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			svc := &fakePayer{confirmErr: tt.confirmErr}
			gateway := &fakeVerifier{result: paidResult, err: tt.verifyErr}

			w := getPayment(t, svc, gateway, false, "/payments/vnpay/ipn?vnp_TxnRef=MA-048291&vnp_SecureHash=abc")

			// VNPay reads only the body: the status is always 200.
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}

			var body struct{ RspCode, Message string }
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("body %s: %v", w.Body, err)
			}
			if body.RspCode != tt.want || body.Message == "" {
				t.Errorf("answer = %+v, want RspCode %s with a message", body, tt.want)
			}

			if tt.verifyErr != nil && svc.confirmed != 0 {
				t.Error("an unverified call reached ConfirmPayment")
			}
		})
	}
}

func TestIPNVerifiesTheWholeQuery(t *testing.T) {

	gateway := &fakeVerifier{result: paidResult}
	getPayment(t, &fakePayer{}, gateway, false, "/payments/vnpay/ipn?vnp_TxnRef=MA-048291&vnp_Amount=20000000&vnp_SecureHash=abc")

	if gateway.got.Get("vnp_Amount") != "20000000" || gateway.got.Get("vnp_SecureHash") != "abc" {
		t.Errorf("verified query = %v, want every parameter VNPay sent", gateway.got)
	}
}

func TestReturnShowsTheOrderWithoutRecording(t *testing.T) {

	svc := &fakePayer{order: Order{Code: "MA-048291", Status: StatusAwaitingPayment}}
	w := getPayment(t, svc, &fakeVerifier{result: paidResult}, false, "/payments/vnpay/return?vnp_TxnRef=MA-048291")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if svc.confirmed != 0 {
		t.Error("the return redirect recorded the payment although only the IPN call may")
	}

	var body returnResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Order.Code != "MA-048291" || !body.PaymentSucceeded {
		t.Errorf("body = %+v, want the order and paymentSucceeded", body)
	}
}

func TestReturnRecordsWhenAllowed(t *testing.T) {

	// A repeat after the IPN call is no error for the customer.
	svc := &fakePayer{order: Order{Code: "MA-048291", Status: StatusConfirmed}, confirmErr: ErrAlreadyProcessed}
	w := getPayment(t, svc, &fakeVerifier{result: paidResult}, true, "/payments/vnpay/return?vnp_TxnRef=MA-048291")

	if w.Code != http.StatusOK || svc.confirmed != 1 {
		t.Errorf("status %d, %d confirmations; want 200 and 1", w.Code, svc.confirmed)
	}
}

func TestReturnRefuses(t *testing.T) {

	tests := []struct {
		name       string
		verifyErr  error
		confirmErr error
		findErr    error
		want       int
	}{
		{"bad signature", vnpay.ErrInvalidSignature, nil, nil, http.StatusBadRequest},
		{"wrong amount", nil, ErrAmountMismatch, nil, http.StatusBadRequest},
		{"unknown order", nil, nil, ErrNotFound, http.StatusNotFound},
		{"database down", nil, nil, errors.New("connection lost"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			svc := &fakePayer{confirmErr: tt.confirmErr, findErr: tt.findErr}
			gateway := &fakeVerifier{result: paidResult, err: tt.verifyErr}

			w := getPayment(t, svc, gateway, true, "/payments/vnpay/return?vnp_TxnRef=MA-048291")
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}
