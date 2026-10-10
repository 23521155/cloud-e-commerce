package book

import (
	"context"
	"slices"
)

// reservationStore is what the reservation service needs from the
// repository; tests replace it with a fake.
type reservationStore interface {
	Reserve(ctx context.Context, holder string, ttlSeconds int, ids []int) ([]Held, error)
	Release(ctx context.Context, holder string, ids []int) (int, error)
	Sell(ctx context.Context, holder string, ids []int) error
	ReleaseExpired(ctx context.Context) (int, error)
}

type ReservationService struct {
	store reservationStore
}

func NewReservationService(store reservationStore) *ReservationService {
	return &ReservationService{store: store}
}

// normalise drops duplicate ids and sorts the rest. Sorting makes every
// order lock its books in the same sequence, which avoids deadlocks.
func normalise(ids []int) []int {

	out := slices.Clone(ids)
	slices.Sort(out)

	return slices.Compact(out)
}

// Reserve holds the books for holder. See Repository.Reserve.
func (s *ReservationService) Reserve(ctx context.Context, holder string, ttlSeconds int, ids []int) (Reservation, error) {

	held, err := s.store.Reserve(ctx, holder, ttlSeconds, normalise(ids))
	if err != nil {
		return Reservation{}, err
	}

	// One UPDATE set the same expiry on every row.
	return Reservation{Items: held, ReservedUntil: held[0].ReservedUntil}, nil
}

// Release gives back the books holder holds and returns how many it freed.
func (s *ReservationService) Release(ctx context.Context, holder string, ids []int) (int, error) {
	return s.store.Release(ctx, holder, normalise(ids))
}

// Sell marks the books holder holds as sold.
func (s *ReservationService) Sell(ctx context.Context, holder string, ids []int) error {
	return s.store.Sell(ctx, holder, normalise(ids))
}

// ReleaseExpired frees every hold that has run out.
func (s *ReservationService) ReleaseExpired(ctx context.Context) (int, error) {
	return s.store.ReleaseExpired(ctx)
}
