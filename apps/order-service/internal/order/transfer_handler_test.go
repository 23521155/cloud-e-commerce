package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fizzisme/order-service/internal/catalogclient"
	"github.com/fizzisme/order-service/internal/payment/sepay"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// fakeTransferPayer answers with what the test set and keeps the last call.
type fakeTransferPayer struct {
	err error

	calls  int
	code   string
	amount int64
	ref    string
}

func (f *fakeTransferPayer) ConfirmTransfer(ctx context.Context, code string, amountVND int64, ref string) (Order, error) {
	f.calls++
	f.code, f.amount, f.ref = code, amountVND, ref
	return Order{}, f.err
}

const (
	transferKey  = "TESTKEY"
	transferAuth = "Apikey " + transferKey
)

// transferBody is a transfer of 200,000 VND to the shop's VA, as BIDV words it.
const transferBody = `{
	"id": 92704,
	"gateway": "BIDV",
	"accountNumber": "1234567890",
	"subAccount": "96247SHOPVA",
	"code": null,
	"content": "111122223333 0900000000 MA048291",
	"transferType": "in",
	"transferAmount": 200000,
	"referenceCode": "FT26291"
}`

func newTransferGateway(t *testing.T) *sepay.Client {
	t.Helper()

	c, err := sepay.New(sepay.Config{
		APIKey:      transferKey,
		Bank:        "BIDV",
		AccountNo:   "96247SHOPVA",
		AccountName: "NGUYEN VAN A",
		CodePrefix:  "MA",
	})
	if err != nil {
		t.Fatalf("sepay.New: %v", err)
	}
	return c
}

// postTransfer sends a webhook call to a TransferHandler with the real SePay
// checks and returns the response.
func postTransfer(t *testing.T, svc transferPayer, auth, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewTransferHandler(svc, newTransferGateway(t), zap.NewNop()).Register(r.Group(""))

	req := httptest.NewRequest(http.MethodPost, "/payments/sepay/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

func wantSuccess(t *testing.T, w *httptest.ResponseRecorder, status int, success bool) {
	t.Helper()

	if w.Code != status {
		t.Fatalf("status = %d, want %d (body %s)", w.Code, status, w.Body)
	}

	var body struct{ Success bool }
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %s: %v", w.Body, err)
	}
	if body.Success != success {
		t.Errorf("success = %v, want %v", body.Success, success)
	}
}

func TestTransferWebhookRecordsThePayment(t *testing.T) {
	svc := &fakeTransferPayer{}

	w := postTransfer(t, svc, transferAuth, transferBody)

	wantSuccess(t, w, http.StatusOK, true)
	if svc.calls != 1 || svc.code != "MA-048291" || svc.amount != 200000 || svc.ref != "sepay:92704" {
		t.Errorf("ConfirmTransfer = %d call(s) with %q, %d, %q; want one with MA-048291, 200000, sepay:92704",
			svc.calls, svc.code, svc.amount, svc.ref)
	}
}

// SePay stops repeating a call only when it gets 200 with success: true.
func TestTransferWebhookAnswers(t *testing.T) {

	tests := []struct {
		name    string
		err     error
		status  int
		success bool
	}{
		{"recorded", nil, http.StatusOK, true},
		{"repeated call", ErrAlreadyProcessed, http.StatusOK, true},
		{"paid too late is recorded too", ErrPaidTooLate, http.StatusOK, true},
		{"unknown order", ErrNotFound, http.StatusOK, true},
		{"wrong amount", ErrAmountMismatch, http.StatusOK, true},
		{"catalogue down", catalogclient.ErrCatalogUnavailable, http.StatusInternalServerError, false},
		{"database down", errors.New("sql: connection refused"), http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeTransferPayer{err: tt.err}

			w := postTransfer(t, svc, transferAuth, transferBody)

			wantSuccess(t, w, tt.status, tt.success)
		})
	}
}

func TestTransferWebhookRefusesABadKey(t *testing.T) {

	for _, auth := range []string{"", "Apikey WRONG", "Bearer " + transferKey} {
		svc := &fakeTransferPayer{}

		w := postTransfer(t, svc, auth, transferBody)

		wantSuccess(t, w, http.StatusUnauthorized, false)
		if svc.calls != 0 {
			t.Errorf("auth %q reached ConfirmTransfer", auth)
		}
	}
}

func TestTransferWebhookRefusesUnreadableJSON(t *testing.T) {
	svc := &fakeTransferPayer{}

	w := postTransfer(t, svc, transferAuth, `{"id": "not a number"`)

	wantSuccess(t, w, http.StatusBadRequest, false)
	if svc.calls != 0 {
		t.Error("an unreadable call reached ConfirmTransfer")
	}
}

// Not our business, or nothing to look up: answered with success so SePay
// stops, and never confirmed.
func TestTransferWebhookIgnores(t *testing.T) {

	tests := []struct{ name, body string }{
		{"money out", strings.Replace(transferBody, `"in"`, `"out"`, 1)},
		{"another VA", strings.Replace(transferBody, "96247SHOPVA", "96247OTHER", 1)},
		{"no order code", strings.Replace(transferBody, "MA048291", "ung ho", 1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeTransferPayer{}

			w := postTransfer(t, svc, transferAuth, tt.body)

			wantSuccess(t, w, http.StatusOK, true)
			if svc.calls != 0 {
				t.Error("ConfirmTransfer was called")
			}
		})
	}
}
