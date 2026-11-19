package order

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

// expire moves the hold of o, in the fake store, so that it ended a minute ago.
func (s onlineSetup) expire(o Order) {
	ended := time.Now().Add(-time.Minute)
	s.store.byID(o.ID).ExpiresAt = &ended
}

func TestExpireUnpaidCancelsAndReleases(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)
	s.expire(o)

	n, err := s.svc.ExpireUnpaid(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if n != 1 || s.store.byID(o.ID).Status != StatusCancelled || s.store.cancelled[o.ID] != expiredReason {
		t.Errorf("cancelled %d, status %s, reason %q; want 1, CANCELLED, %q",
			n, s.store.byID(o.ID).Status, s.store.cancelled[o.ID], expiredReason)
	}
	if len(s.catalog.released) != 1 || s.catalog.released[0].Holder != o.Code ||
		!slices.Equal(s.catalog.released[0].IDs, []int{1, 2}) {
		t.Errorf("release calls = %+v, want one for both books of %s", s.catalog.released, o.Code)
	}
}

func TestExpireUnpaidLeavesOrdersStillOnHold(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)

	if n, err := s.svc.ExpireUnpaid(context.Background()); err != nil || n != 0 {
		t.Fatalf("ExpireUnpaid = %d, %v; want 0, nil", n, err)
	}
	if s.store.byID(o.ID).Status != StatusAwaitingPayment || len(s.catalog.released) != 0 {
		t.Error("an order still on hold was cancelled or released")
	}
}

func TestExpireUnpaidLeavesPaidOrders(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)
	if _, err := s.svc.ConfirmPayment(context.Background(), paid(o)); err != nil {
		t.Fatal(err)
	}
	s.expire(o)

	if n, err := s.svc.ExpireUnpaid(context.Background()); err != nil || n != 0 {
		t.Fatalf("ExpireUnpaid = %d, %v; want 0, nil", n, err)
	}
	if s.store.byID(o.ID).Status != StatusConfirmed || len(s.catalog.released) != 0 {
		t.Error("a paid order was cancelled or its books released")
	}
}

// lostRaceStore lists an expired order, but a parallel call (a payment, or
// another instance's sweep) changes it before the cancel.
type lostRaceStore struct{ *fakeStore }

func (s lostRaceStore) CancelAwaiting(ctx context.Context, id int64, reason string) error {
	return ErrNotAwaitingPayment
}

func TestExpireUnpaidLosesTheRaceQuietly(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)
	s.expire(o)

	svc := newTestService(lostRaceStore{s.store}, s.catalog)
	if n, err := svc.ExpireUnpaid(context.Background()); err != nil || n != 0 {
		t.Fatalf("ExpireUnpaid = %d, %v; want 0, nil", n, err)
	}
	if len(s.catalog.released) != 0 {
		t.Error("books were released by a sweep that did not cancel the order")
	}
}

func TestExpireUnpaidStoreDown(t *testing.T) {
	s := newOnlineSetup()
	s.store.listExpiredErr = errors.New("connection lost")

	if _, err := s.svc.ExpireUnpaid(context.Background()); err == nil {
		t.Fatal("err = nil, want the store error")
	}
}

func TestRunExpirySweepsUntilStopped(t *testing.T) {
	s := newOnlineSetup()
	o := s.place(t)
	s.expire(o)

	ctx, stop := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.svc.RunExpiry(ctx, 10*time.Millisecond)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	stop()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunExpiry did not return after its context was cancelled")
	}

	// Read only once the loop has stopped: the fakes are not safe for
	// concurrent use.
	if s.store.byID(o.ID).Status != StatusCancelled {
		t.Error("RunExpiry did not cancel the expired order")
	}
}
