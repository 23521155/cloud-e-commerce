package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fizzisme/order-service/internal/payment/sepay"
	"go.uber.org/zap"
)

// fakeTransfers answers with what the test set and keeps the requests.
type fakeTransfers struct {
	err error

	requests []transferRequest
}

type transferRequest struct {
	Code      string
	AmountVND int64
}

func (f *fakeTransfers) Instructions(orderCode string, amountVND int64) (sepay.Instructions, error) {
	f.requests = append(f.requests, transferRequest{orderCode, amountVND})
	if f.err != nil {
		return sepay.Instructions{}, f.err
	}
	return sepay.Instructions{ImageURL: "https://qr.example/?ref=" + orderCode, AmountVND: amountVND}, nil
}

// newTransferSetup is an online setup whose VietQR orders are paid by bank
// transfer.
func newTransferSetup() (onlineSetup, *fakeTransfers) {
	s := newOnlineSetup()
	transfers := &fakeTransfers{}
	s.svc.WithTransfers(transfers)
	return s, transfers
}

func TestCreateVietQRIsPaidByTransfer(t *testing.T) {
	s, transfers := newTransferSetup()

	got := s.place(t)

	if got.Status != StatusAwaitingPayment {
		t.Errorf("status = %s, want AWAITING_PAYMENT", got.Status)
	}
	if got.Transfer == nil || got.Transfer.ImageURL != "https://qr.example/?ref="+got.Code {
		t.Errorf("Transfer = %+v, want the instructions for %s", got.Transfer, got.Code)
	}
	if got.PaymentURL != "" || len(s.payments.requests) != 0 {
		t.Errorf("PaymentURL = %q, %d gateway calls; want none for a transfer",
			got.PaymentURL, len(s.payments.requests))
	}
	if len(transfers.requests) != 1 || transfers.requests[0] != (transferRequest{got.Code, 200000}) {
		t.Errorf("requests = %+v, want one for %s, 200000 VND", transfers.requests, got.Code)
	}
}

// Only VietQR changes: a card is still paid on the gateway's page.
func TestCreateCardStaysOnTheGateway(t *testing.T) {
	s, transfers := newTransferSetup()

	in := onlineInput()
	in.PaymentMethod = PaymentCard

	got, _, err := s.svc.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}

	if got.PaymentURL == "" || got.Transfer != nil || len(transfers.requests) != 0 {
		t.Errorf("PaymentURL = %q, Transfer = %+v; want the gateway's URL only", got.PaymentURL, got.Transfer)
	}
}

// A shop that takes transfers needs no VNPay account for VietQR.
func TestCreateVietQRWithTransfersButWithoutGateway(t *testing.T) {
	store, catalog := newFakeStore(), &fakeCatalog{books: twoBooks()}
	svc := NewService(store, catalog, nil, testHold, zap.NewNop()).WithTransfers(&fakeTransfers{})

	got, _, err := svc.Create(context.Background(), onlineInput())
	if err != nil {
		t.Fatal(err)
	}
	if got.Transfer == nil {
		t.Error("Transfer = nil, want the instructions")
	}

	// A card still has no gateway to be paid on.
	in := onlineInput()
	in.Key = "2f2f6f46-6f8b-4a45-9a5b-4a3c7c5d9e11"
	in.PaymentMethod = PaymentCard
	if _, _, err := svc.Create(context.Background(), in); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("card without a gateway: err = %v, want ErrInvalidInput", err)
	}
}

func TestCreateVietQRInstructionsFailureCancelsAndReleases(t *testing.T) {
	s, transfers := newTransferSetup()
	transfers.err = errors.New("no QR")

	if _, _, err := s.svc.Create(context.Background(), onlineInput()); err == nil {
		t.Fatal("err = nil, want the instructions error")
	}
	if len(s.store.cancelled) != 1 || len(s.catalog.released) != 1 {
		t.Errorf("cancelled %v, released %+v; want one of each", s.store.cancelled, s.catalog.released)
	}
}

func TestCreateVietQRReplayGivesTheInstructionsAgain(t *testing.T) {
	s, transfers := newTransferSetup()
	first := s.place(t)

	again, replayed, err := s.svc.Create(context.Background(), onlineInput())
	if err != nil {
		t.Fatal(err)
	}

	if !replayed || again.Code != first.Code || again.Transfer == nil {
		t.Errorf("replayed = %v, code %s, Transfer %+v; want the first order %s with its instructions",
			replayed, again.Code, again.Transfer, first.Code)
	}
	if len(s.catalog.reserved) != 1 || len(transfers.requests) != 2 {
		t.Errorf("%d reserves, %d instructions; want 1 and 2", len(s.catalog.reserved), len(transfers.requests))
	}
}

func TestCreateVietQRReplayAfterTheWindowClosed(t *testing.T) {
	s, _ := newTransferSetup()
	first := s.place(t)

	// The payment window closed a minute ago.
	closed := time.Now().Add(testHold/payWindowShare - time.Minute)
	s.store.orders[onlineInput().Key].ExpiresAt = &closed

	again, _, err := s.svc.Create(context.Background(), onlineInput())
	if err != nil {
		t.Fatal(err)
	}
	if again.Code != first.Code || again.Transfer != nil {
		t.Errorf("code %s, Transfer %+v; want order %s with no instructions", again.Code, again.Transfer, first.Code)
	}
}
