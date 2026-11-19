package order

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fizzisme/order-service/internal/catalogclient"
	"github.com/fizzisme/order-service/internal/payment/vnpay"
	"go.uber.org/zap"
)

// onlineInput is validInput paid by VietQR through the gateway.
func onlineInput() CreateInput {
	in := validInput()
	in.PaymentMethod = PaymentVietQR
	in.ClientIP = "203.0.113.7"
	return in
}

// onlineSetup is a service with a fake store, catalogue and gateway.
type onlineSetup struct {
	svc      *Service
	store    *fakeStore
	catalog  *fakeCatalog
	payments *fakePayments
}

func newOnlineSetup() onlineSetup {
	s := onlineSetup{
		store:    newFakeStore(),
		catalog:  &fakeCatalog{books: twoBooks()},
		payments: &fakePayments{},
	}
	s.svc = NewService(s.store, s.catalog, s.payments, testHold, zap.NewNop())
	return s
}

// place puts an online order for twoBooks and fails the test if it cannot.
func (s onlineSetup) place(t *testing.T) Order {
	t.Helper()

	o, _, err := s.svc.Create(context.Background(), onlineInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return o
}

// paid is what the gateway reports for a successful payment of o.
func paid(o Order) vnpay.Result {
	return vnpay.Result{
		TxnRef:            o.Code,
		AmountVND:         o.SubtotalVND,
		ResponseCode:      "00",
		TransactionStatus: "00",
		TransactionNo:     "14567890",
	}
}

// ---- Create, online ------------------------------------------------------

func TestCreateOnlineAwaitsPayment(t *testing.T) {
	s := newOnlineSetup()

	before := time.Now()
	got := s.place(t)
	after := time.Now()

	if got.Status != StatusAwaitingPayment || got.PaymentStatus != PaymentUnpaid {
		t.Errorf("order = %s / %s, want AWAITING_PAYMENT / UNPAID", got.Status, got.PaymentStatus)
	}

	// The books are held for the payment hold, and nothing is sold yet.
	if len(s.catalog.reserved) != 1 || s.catalog.reserved[0].TTL != testHold {
		t.Errorf("reserve calls = %+v, want one for %v", s.catalog.reserved, testHold)
	}
	if len(s.catalog.sold) != 0 {
		t.Errorf("sell calls = %+v, want none before the payment", s.catalog.sold)
	}

	// ExpiresAt is the end of the hold as the catalogue reports it.
	if got.ExpiresAt == nil ||
		got.ExpiresAt.Before(before.Add(testHold)) || got.ExpiresAt.After(after.Add(testHold)) {
		t.Errorf("ExpiresAt = %v, want the catalogue's ReservedUntil", got.ExpiresAt)
	}

	if got.PaymentURL != "https://pay.example/?ref="+got.Code {
		t.Errorf("PaymentURL = %q, want the gateway's URL for %s", got.PaymentURL, got.Code)
	}
}

func TestCreateOnlineAsksTheGatewayForTheRightPayment(t *testing.T) {
	s := newOnlineSetup()
	got := s.place(t)

	if len(s.payments.requests) != 1 {
		t.Fatalf("BuildPayURL calls = %d, want 1", len(s.payments.requests))
	}
	r := s.payments.requests[0]

	if r.TxnRef != got.Code || r.AmountVND != 200000 || r.IPAddr != "203.0.113.7" {
		t.Errorf("request = %+v, want order %s, 200000 VND, the customer's IP", r, got.Code)
	}

	// The gateway closes a fifth of the hold before the books are released.
	if want := got.ExpiresAt.Add(-testHold / payWindowShare); !r.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", r.ExpiresAt, want)
	}

	if !strings.Contains(r.OrderInfo, got.Code) {
		t.Errorf("OrderInfo = %q, want it to name %s", r.OrderInfo, got.Code)
	}
	for _, c := range r.OrderInfo {
		if c > 127 {
			t.Errorf("OrderInfo = %q, want plain ASCII", r.OrderInfo)
			break
		}
	}
}

func TestCreateOnlineWithoutGateway(t *testing.T) {
	store, catalog := newFakeStore(), &fakeCatalog{books: twoBooks()}
	svc := NewService(store, catalog, nil, testHold, zap.NewNop())

	if _, _, err := svc.Create(context.Background(), onlineInput()); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if len(catalog.reserved) != 0 {
		t.Error("books were reserved for a method that cannot be paid")
	}
}

func TestCreateCODHasNoPaymentURL(t *testing.T) {
	s := newOnlineSetup()

	got, _, err := s.svc.Create(context.Background(), validInput())
	if err != nil {
		t.Fatal(err)
	}
	if got.PaymentURL != "" || len(s.payments.requests) != 0 {
		t.Errorf("PaymentURL = %q, %d gateway calls; want none for cash on delivery",
			got.PaymentURL, len(s.payments.requests))
	}
}

func TestCreateOnlineSigningFailureCancelsAndReleases(t *testing.T) {
	s := newOnlineSetup()
	s.payments.err = errors.New("cannot sign")

	if _, _, err := s.svc.Create(context.Background(), onlineInput()); err == nil {
		t.Fatal("err = nil, want the signing error")
	}
	if len(s.store.cancelled) != 1 || len(s.catalog.released) != 1 {
		t.Errorf("cancelled %v, released %+v; want one of each", s.store.cancelled, s.catalog.released)
	}
}

func TestCreateOnlineReplaySignsAgain(t *testing.T) {
	s := newOnlineSetup()
	first := s.place(t)

	again, replayed, err := s.svc.Create(context.Background(), onlineInput())
	if err != nil {
		t.Fatal(err)
	}

	if !replayed || again.Code != first.Code {
		t.Errorf("replayed = %v, code %s; want the first order %s", replayed, again.Code, first.Code)
	}
	if again.PaymentURL == "" {
		t.Error("a replayed order awaiting payment has no PaymentURL")
	}
	if len(s.catalog.reserved) != 1 || len(s.payments.requests) != 2 {
		t.Errorf("%d reserves, %d signatures; want 1 and 2", len(s.catalog.reserved), len(s.payments.requests))
	}
}

func TestCreateOnlineReplayAfterTheWindowClosed(t *testing.T) {
	s := newOnlineSetup()
	first := s.place(t)

	// The payment window closed a minute ago.
	closed := time.Now().Add(testHold/payWindowShare - time.Minute)
	s.store.orders[onlineInput().Key].ExpiresAt = &closed

	again, _, err := s.svc.Create(context.Background(), onlineInput())
	if err != nil {
		t.Fatal(err)
	}
	if again.Code != first.Code || again.PaymentURL != "" {
		t.Errorf("code %s, PaymentURL %q; want order %s with no URL", again.Code, again.PaymentURL, first.Code)
	}
}

// ---- ConfirmPayment ------------------------------------------------------

func TestConfirmPaymentPaid(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)

	got, err := s.svc.ConfirmPayment(context.Background(), paid(o))
	if err != nil {
		t.Fatal(err)
	}

	if got.Status != StatusConfirmed || got.PaymentStatus != PaymentPaid ||
		got.PaymentRef == nil || *got.PaymentRef != "14567890" {
		t.Errorf("order = %s / %s / ref %v, want CONFIRMED / PAID / 14567890",
			got.Status, got.PaymentStatus, got.PaymentRef)
	}
	if len(s.catalog.sold) != 1 || s.catalog.sold[0].Holder != o.Code ||
		!slices.Equal(s.catalog.sold[0].IDs, []int{1, 2}) {
		t.Errorf("sell calls = %+v, want one for both books of %s", s.catalog.sold, o.Code)
	}
	if s.store.paid[o.ID] != "14567890" {
		t.Errorf("paid = %v, want the order marked paid with the gateway's ref", s.store.paid)
	}
	if len(s.catalog.released) != 0 || len(s.store.cancelled) != 0 {
		t.Errorf("released %+v, cancelled %v; want neither", s.catalog.released, s.store.cancelled)
	}
}

func TestConfirmPaymentUnknownOrder(t *testing.T) {
	s := newOnlineSetup()

	r := paid(Order{Code: "MA-000000", SubtotalVND: 200000})
	if _, err := s.svc.ConfirmPayment(context.Background(), r); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestConfirmPaymentWrongAmount(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)

	r := paid(o)
	r.AmountVND = 1000

	if _, err := s.svc.ConfirmPayment(context.Background(), r); !errors.Is(err, ErrAmountMismatch) {
		t.Fatalf("err = %v, want ErrAmountMismatch", err)
	}
	if len(s.catalog.sold) != 0 || len(s.store.paid) != 0 || len(s.store.cancelled) != 0 {
		t.Error("a payment with the wrong amount changed the order")
	}
}

func TestConfirmPaymentFailed(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)

	r := paid(o)
	r.ResponseCode, r.TransactionStatus = "24", "02" // the customer cancelled

	got, err := s.svc.ConfirmPayment(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}

	if got.Status != StatusCancelled {
		t.Errorf("status = %s, want CANCELLED", got.Status)
	}
	if len(s.catalog.released) != 1 || !slices.Equal(s.catalog.released[0].IDs, []int{1, 2}) {
		t.Errorf("release calls = %+v, want one for both books", s.catalog.released)
	}
	if len(s.catalog.sold) != 0 || len(s.store.paid) != 0 {
		t.Error("a failed payment sold books or marked the order paid")
	}
}

func TestConfirmPaymentTwice(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)

	if _, err := s.svc.ConfirmPayment(context.Background(), paid(o)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.svc.ConfirmPayment(context.Background(), paid(o)); !errors.Is(err, ErrAlreadyProcessed) {
		t.Fatalf("second call: err = %v, want ErrAlreadyProcessed", err)
	}
	if len(s.catalog.sold) != 1 {
		t.Errorf("sell calls = %d, want 1", len(s.catalog.sold))
	}
}

// A failure that arrives after the payment was recorded changes nothing.
func TestConfirmPaymentFailureAfterPaid(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)

	if _, err := s.svc.ConfirmPayment(context.Background(), paid(o)); err != nil {
		t.Fatal(err)
	}

	r := paid(o)
	r.ResponseCode = "24"
	if _, err := s.svc.ConfirmPayment(context.Background(), r); !errors.Is(err, ErrAlreadyProcessed) {
		t.Fatalf("err = %v, want ErrAlreadyProcessed", err)
	}
	if len(s.catalog.released) != 0 {
		t.Error("the books of a paid order were released")
	}
}

func TestConfirmPaymentCatalogueDown(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)
	s.catalog.sellErr = catalogclient.ErrCatalogUnavailable

	if _, err := s.svc.ConfirmPayment(context.Background(), paid(o)); !errors.Is(err, catalogclient.ErrCatalogUnavailable) {
		t.Fatalf("err = %v, want ErrCatalogUnavailable", err)
	}

	// Nothing is recorded or undone: the gateway calls again.
	if len(s.store.paid) != 0 || len(s.store.cancelled) != 0 || len(s.catalog.released) != 0 {
		t.Errorf("paid %v, cancelled %v, released %+v; want nothing changed",
			s.store.paid, s.store.cancelled, s.catalog.released)
	}
	if len(s.catalog.sold) != 2 {
		t.Errorf("sell calls = %d, want 2 (one retry)", len(s.catalog.sold))
	}
}

func TestConfirmPaymentBooksGone(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)
	s.catalog.sellErr = &catalogclient.UnavailableError{
		Books: []catalogclient.Unavailable{{ID: 1, Status: catalogclient.StatusSold}},
	}

	if _, err := s.svc.ConfirmPayment(context.Background(), paid(o)); !errors.Is(err, ErrPaidTooLate) {
		t.Fatalf("err = %v, want ErrPaidTooLate", err)
	}

	reason, ok := s.store.cancelled[o.ID]
	if !ok || !strings.Contains(reason, "14567890") {
		t.Errorf("cancel reason = %q, want the order cancelled naming the payment to refund", reason)
	}
	if len(s.store.paid) != 0 {
		t.Error("an order without books was marked paid")
	}
}

func TestConfirmPaymentParallelCallWon(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)

	// Another call marked the order paid between our read and our update.
	s.store.markPaidErr = ErrNotAwaitingPayment

	if _, err := s.svc.ConfirmPayment(context.Background(), paid(o)); !errors.Is(err, ErrAlreadyProcessed) {
		t.Fatalf("err = %v, want ErrAlreadyProcessed", err)
	}
}

func TestConfirmPaymentForACancelledOrder(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)

	// The hold ran out and the order was cancelled; the payment comes later.
	if err := s.store.CancelAwaiting(context.Background(), o.ID, "hold expired"); err != nil {
		t.Fatal(err)
	}

	if _, err := s.svc.ConfirmPayment(context.Background(), paid(o)); !errors.Is(err, ErrAlreadyProcessed) {
		t.Fatalf("err = %v, want ErrAlreadyProcessed", err)
	}
	if len(s.catalog.sold) != 0 {
		t.Error("books were sold for a cancelled order")
	}
}
