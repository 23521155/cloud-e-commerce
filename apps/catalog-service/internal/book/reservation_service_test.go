package book

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeReservationStore struct {
	gotHolder string
	gotTTL    int
	gotIDs    []int
	held      []Held
	err       error
}

func (f *fakeReservationStore) Reserve(_ context.Context, holder string, ttl int, ids []int) ([]Held, error) {
	f.gotHolder, f.gotTTL, f.gotIDs = holder, ttl, ids
	return f.held, f.err
}

func (f *fakeReservationStore) Release(_ context.Context, holder string, ids []int) (int, error) {
	f.gotHolder, f.gotIDs = holder, ids
	return len(ids), f.err
}

func (f *fakeReservationStore) Sell(_ context.Context, holder string, ids []int) error {
	f.gotHolder, f.gotIDs = holder, ids
	return f.err
}

func (f *fakeReservationStore) ReleaseExpired(context.Context) (int, error) {
	return 3, f.err
}

func TestNormaliseSortsAndDropsDuplicates(t *testing.T) {
	tests := []struct {
		name string
		in   []int
		want []int
	}{
		{"already sorted", []int{1, 2, 3}, []int{1, 2, 3}},
		{"unsorted", []int{9, 2, 5}, []int{2, 5, 9}},
		{"duplicates", []int{4, 4, 1, 4}, []int{1, 4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			in := append([]int(nil), tt.in...)

			if got := normalise(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalise(%v) = %v, want %v", in, got, tt.want)
			}
			if !reflect.DeepEqual(tt.in, in) {
				t.Fatalf("input was modified: %v, was %v", tt.in, in)
			}
		})
	}
}

func TestServiceReserveNormalisesAndReturnsExpiry(t *testing.T) {

	until := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	store := &fakeReservationStore{held: []Held{
		{ID: 1, ReservedUntil: until},
		{ID: 3, ReservedUntil: until},
	}}

	got, err := NewReservationService(store).Reserve(context.Background(), "BK-1", 900, []int{3, 1, 3})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(store.gotIDs, []int{1, 3}) {
		t.Errorf("store got ids %v, want [1 3]", store.gotIDs)
	}
	if store.gotHolder != "BK-1" || store.gotTTL != 900 {
		t.Errorf("store got holder %q ttl %d", store.gotHolder, store.gotTTL)
	}
	if !got.ReservedUntil.Equal(until) || len(got.Items) != 2 {
		t.Errorf("reservation = %+v", got)
	}
}

func TestServicePassesUnavailableThrough(t *testing.T) {

	want := &UnavailableError{Books: []Unavailable{{ID: 3, Status: StatusSold}}}
	svc := NewReservationService(&fakeReservationStore{err: want})

	ctx := context.Background()

	if _, err := svc.Reserve(ctx, "BK-1", 60, []int{3}); !errors.Is(err, want) {
		t.Errorf("Reserve error = %v, want %v", err, want)
	}
	if err := svc.Sell(ctx, "BK-1", []int{3}); !errors.Is(err, want) {
		t.Errorf("Sell error = %v, want %v", err, want)
	}
}
