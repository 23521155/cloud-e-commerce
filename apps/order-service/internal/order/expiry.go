package order

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

// expiryBatch is how many ended holds one sweep handles; the next sweep
// takes the rest.
const expiryBatch = 100

// expiryTimeout bounds one sweep.
const expiryTimeout = 30 * time.Second

// expiredReason is the cancel reason of an order nobody paid in time.
const expiredReason = "not paid before the hold ended"

// ExpireUnpaid cancels the online orders whose hold has ended without a
// payment and gives their books back. It returns how many it cancelled.
//
// The catalogue lets a hold lapse by itself, so the books may already be
// free; releasing them again is harmless, as Release only frees books still
// held by this order. An order a parallel call has just paid or cancelled is
// left alone: CancelAwaiting decides, so several instances may sweep at once.
func (s *Service) ExpireUnpaid(ctx context.Context) (int, error) {

	orders, err := s.store.ListExpiredAwaiting(ctx, expiryBatch)
	if err != nil {
		return 0, err
	}

	cancelled := 0
	for _, o := range orders {

		err := s.store.CancelAwaiting(ctx, o.ID, expiredReason)
		switch {
		case errors.Is(err, ErrNotAwaitingPayment):
			continue
		case err != nil:
			// The next sweep tries this order again.
			s.log.Warn("cannot cancel an unpaid order", zap.String("code", o.Code), zap.Error(err))
			continue
		}
		cancelled++

		ids := make([]int, len(o.Items))
		for i, it := range o.Items {
			ids[i] = it.BookID
		}
		s.release(ctx, o.Code, ids)
	}

	return cancelled, nil
}

// RunExpiry calls ExpireUnpaid every interval until ctx is done. Meant to run
// in its own goroutine for the life of the service.
func (s *Service) RunExpiry(ctx context.Context, interval time.Duration) {

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweep(ctx)
		}
	}
}

// sweep runs one ExpireUnpaid and logs what it did.
func (s *Service) sweep(ctx context.Context) {

	ctx, cancel := context.WithTimeout(ctx, expiryTimeout)
	defer cancel()

	n, err := s.ExpireUnpaid(ctx)
	switch {
	case err != nil:
		s.log.Warn("cannot sweep unpaid orders", zap.Error(err))
	case n > 0:
		s.log.Info("cancelled unpaid orders", zap.Int("count", n))
	}
}
