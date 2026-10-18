package order

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/fizzisme/order-service/internal/catalogclient"
)

const transferRef = "sepay:92704"

func TestConfirmTransferPaid(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)

	got, err := s.svc.ConfirmTransfer(context.Background(), o.Code, o.SubtotalVND, transferRef)
	if err != nil {
		t.Fatal(err)
	}

	if got.Status != StatusConfirmed || got.PaymentStatus != PaymentPaid ||
		got.PaymentRef == nil || *got.PaymentRef != transferRef || got.PaidAt == nil {
		t.Errorf("order = %s / %s / ref %v / paidAt %v, want CONFIRMED / PAID / %s / set",
			got.Status, got.PaymentStatus, got.PaymentRef, got.PaidAt, transferRef)
	}
	if len(s.catalog.sold) != 1 || s.catalog.sold[0].Holder != o.Code ||
		!slices.Equal(s.catalog.sold[0].IDs, []int{1, 2}) {
		t.Errorf("sell calls = %+v, want one for both books of %s", s.catalog.sold, o.Code)
	}
	if s.store.paid[o.ID] != transferRef {
		t.Errorf("paid = %v, want the order marked paid with %s", s.store.paid, transferRef)
	}
	if len(s.catalog.released) != 0 || len(s.store.cancelled) != 0 {
		t.Errorf("released %+v, cancelled %v; want neither", s.catalog.released, s.store.cancelled)
	}
}

func TestConfirmTransferUnknownOrder(t *testing.T) {
	s := newOnlineSetup()

	if _, err := s.svc.ConfirmTransfer(context.Background(), "MA-000000", 200000, transferRef); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// A customer who sends too little or too much changes nothing: the order
// stays open and the owner settles it by hand.
func TestConfirmTransferWrongAmount(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)

	for _, amount := range []int64{1000, o.SubtotalVND + 1} {
		_, err := s.svc.ConfirmTransfer(context.Background(), o.Code, amount, transferRef)
		if !errors.Is(err, ErrAmountMismatch) {
			t.Fatalf("amount %d: err = %v, want ErrAmountMismatch", amount, err)
		}
	}

	if len(s.catalog.sold) != 0 || len(s.store.paid) != 0 || len(s.store.cancelled) != 0 {
		t.Error("a transfer with the wrong amount changed the order")
	}
}

// SePay repeats a webhook until it gets an answer.
func TestConfirmTransferTwice(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)

	if _, err := s.svc.ConfirmTransfer(context.Background(), o.Code, o.SubtotalVND, transferRef); err != nil {
		t.Fatal(err)
	}
	if _, err := s.svc.ConfirmTransfer(context.Background(), o.Code, o.SubtotalVND, transferRef); !errors.Is(err, ErrAlreadyProcessed) {
		t.Fatalf("second call: err = %v, want ErrAlreadyProcessed", err)
	}
	if len(s.catalog.sold) != 1 {
		t.Errorf("sell calls = %d, want 1", len(s.catalog.sold))
	}
}

func TestConfirmTransferCatalogueDown(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)
	s.catalog.sellErr = catalogclient.ErrCatalogUnavailable

	_, err := s.svc.ConfirmTransfer(context.Background(), o.Code, o.SubtotalVND, transferRef)
	if !errors.Is(err, catalogclient.ErrCatalogUnavailable) {
		t.Fatalf("err = %v, want ErrCatalogUnavailable", err)
	}

	// Nothing is recorded or undone: SePay calls again.
	if len(s.store.paid) != 0 || len(s.store.cancelled) != 0 || len(s.catalog.released) != 0 {
		t.Errorf("paid %v, cancelled %v, released %+v; want nothing changed",
			s.store.paid, s.store.cancelled, s.catalog.released)
	}
}

func TestConfirmTransferBooksGone(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)
	s.catalog.sellErr = &catalogclient.UnavailableError{
		Books: []catalogclient.Unavailable{{ID: 1, Status: catalogclient.StatusSold}},
	}

	_, err := s.svc.ConfirmTransfer(context.Background(), o.Code, o.SubtotalVND, transferRef)
	if !errors.Is(err, ErrPaidTooLate) {
		t.Fatalf("err = %v, want ErrPaidTooLate", err)
	}

	reason, ok := s.store.cancelled[o.ID]
	if !ok || !strings.Contains(reason, transferRef) {
		t.Errorf("cancel reason = %q, want the order cancelled naming the transfer to refund", reason)
	}
	if len(s.store.paid) != 0 {
		t.Error("an order without books was marked paid")
	}
}

func TestConfirmTransferParallelCallWon(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)

	// Another call marked the order paid between our read and our update.
	s.store.markPaidErr = ErrNotAwaitingPayment

	_, err := s.svc.ConfirmTransfer(context.Background(), o.Code, o.SubtotalVND, transferRef)
	if !errors.Is(err, ErrAlreadyProcessed) {
		t.Fatalf("err = %v, want ErrAlreadyProcessed", err)
	}
}

// The hold ran out and the order was cancelled; the transfer comes later.
func TestConfirmTransferForACancelledOrder(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)

	if err := s.store.CancelAwaiting(context.Background(), o.ID, "hold expired"); err != nil {
		t.Fatal(err)
	}

	_, err := s.svc.ConfirmTransfer(context.Background(), o.Code, o.SubtotalVND, transferRef)
	if !errors.Is(err, ErrAlreadyProcessed) {
		t.Fatalf("err = %v, want ErrAlreadyProcessed", err)
	}
	if len(s.catalog.sold) != 0 {
		t.Error("books were sold for a cancelled order")
	}
}
