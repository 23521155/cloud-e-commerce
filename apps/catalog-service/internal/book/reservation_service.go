package book

import (
	"cmp"
	"context"
	"slices"
	"strings"
)

// reservationStore is what the reservation service needs from the
// repository; tests replace it with a fake.
type reservationStore interface {
	Reserve(ctx context.Context, holder string, ttlSeconds int, slugs []string) ([]Held, error)
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

// normalise drops duplicate values and sorts the rest. Sorting makes every
// order lock its books in the same sequence, which avoids deadlocks.
func normalise[T cmp.Ordered](values []T) []T {

	out := slices.Clone(values)
	slices.Sort(out)

	return slices.Compact(out)
}

// normaliseSlugs trims and lower-cases the slugs before normalise. SQL Server
// compares them without regard to case, so "Heidi" and "heidi" would match
// one row and make a request for two books look like a request that came
// up short.
func normaliseSlugs(slugs []string) []string {

	out := make([]string, len(slugs))
	for i, slug := range slugs {
		out[i] = strings.ToLower(strings.TrimSpace(slug))
	}

	return normalise(out)
}

// Reserve holds the books with the given slugs for holder. See Repository.Reserve.
func (s *ReservationService) Reserve(ctx context.Context, holder string, ttlSeconds int, slugs []string) (Reservation, error) {

	held, err := s.store.Reserve(ctx, holder, ttlSeconds, normaliseSlugs(slugs))
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
