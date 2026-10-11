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
	gotSlugs  []string
	gotIDs    []int
	held      []Held
	err       error
}

func (f *fakeReservationStore) Reserve(_ context.Context, holder string, ttl int, slugs []string) ([]Held, error) {
	f.gotHolder, f.gotTTL, f.gotSlugs = holder, ttl, slugs
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

func TestNormaliseSlugs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"sorted and unique", []string{"a-1", "b-2"}, []string{"a-1", "b-2"}},
		{"unsorted", []string{"c-3", "a-1", "b-2"}, []string{"a-1", "b-2", "c-3"}},
		{"duplicates", []string{"a-1", "a-1", "b-2"}, []string{"a-1", "b-2"}},
		// SQL Server matches slugs without regard to case: both spellings
		// are one row, so they must be one entry.
		{"the same slug in another case", []string{"Heidi-B1", "heidi-b1"}, []string{"heidi-b1"}},
		{"surrounding spaces", []string{"  heidi-b1 ", "heidi-b1"}, []string{"heidi-b1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			in := append([]string(nil), tt.in...)

			if got := normaliseSlugs(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normaliseSlugs(%q) = %q, want %q", in, got, tt.want)
			}
			if !reflect.DeepEqual(tt.in, in) {
				t.Fatalf("input was modified: %q, was %q", tt.in, in)
			}
		})
	}
}

func TestServiceReserveNormalisesAndReturnsExpiry(t *testing.T) {

	until := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	store := &fakeReservationStore{held: []Held{
		{ID: 1, Slug: "emma-b1", ReservedUntil: until},
		{ID: 3, Slug: "heidi-b3", ReservedUntil: until},
	}}

	got, err := NewReservationService(store).Reserve(context.Background(), "BK-1", 900, []string{"Heidi-B3", "emma-b1", "heidi-b3"})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(store.gotSlugs, []string{"emma-b1", "heidi-b3"}) {
		t.Errorf("store got slugs %q, want [emma-b1 heidi-b3]", store.gotSlugs)
	}
	if store.gotHolder != "BK-1" || store.gotTTL != 900 {
		t.Errorf("store got holder %q ttl %d", store.gotHolder, store.gotTTL)
	}
	if !got.ReservedUntil.Equal(until) || len(got.Items) != 2 {
		t.Errorf("reservation = %+v", got)
	}
}

func TestServiceReleaseAndSellNormaliseIDs(t *testing.T) {

	store := &fakeReservationStore{}
	svc := NewReservationService(store)
	ctx := context.Background()

	if _, err := svc.Release(ctx, "BK-1", []int{9, 2, 9}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.gotIDs, []int{2, 9}) {
		t.Errorf("Release: store got ids %v, want [2 9]", store.gotIDs)
	}

	if err := svc.Sell(ctx, "BK-1", []int{5, 4, 5}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.gotIDs, []int{4, 5}) {
		t.Errorf("Sell: store got ids %v, want [4 5]", store.gotIDs)
	}
}

func TestServicePassesUnavailableThrough(t *testing.T) {

	want := &UnavailableError{Books: []Unavailable{{ID: 3, Slug: "heidi-b3", Status: StatusSold}}}
	svc := NewReservationService(&fakeReservationStore{err: want})

	ctx := context.Background()

	if _, err := svc.Reserve(ctx, "BK-1", 60, []string{"heidi-b3"}); !errors.Is(err, want) {
		t.Errorf("Reserve error = %v, want %v", err, want)
	}
	if err := svc.Sell(ctx, "BK-1", []int{3}); !errors.Is(err, want) {
		t.Errorf("Sell error = %v, want %v", err, want)
	}
}

func TestUnavailableOfSlugs(t *testing.T) {

	reservedBy := "BK-9"
	rows := []bookState{
		{ID: 1, Slug: "emma-b1", Status: StatusAvailable},
		{ID: 2, Slug: "heidi-b2", Status: StatusSold},
		{ID: 3, Slug: "kim-b3", Status: StatusReserved, ReservedBy: &reservedBy},
	}

	// What the repository treats as fine: here, only the first book.
	ok := func(row bookState) bool { return row.ID == 1 }

	got := unavailableOfSlugs([]string{"emma-b1", "heidi-b2", "kim-b3", "no-such-book"}, rows, ok)

	want := []Unavailable{
		{ID: 2, Slug: "heidi-b2", Status: StatusSold},
		{ID: 3, Slug: "kim-b3", Status: StatusReserved},
		{Slug: "no-such-book", Status: StatusMissing}, // no row: the id is not known
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}

	// The database matches slugs without regard to case; so does the report.
	got = unavailableOfSlugs([]string{"HEIDI-B2"}, rows, ok)
	if len(got) != 1 || got[0].ID != 2 || got[0].Status != StatusSold {
		t.Fatalf("a slug in another case was not matched to its row: %+v", got)
	}
}

func TestUnavailableOfIDsFillsInTheSlug(t *testing.T) {

	rows := []bookState{{ID: 2, Slug: "heidi-b2", Status: StatusSold}}

	got := unavailableOf([]int{2, 99}, rows, func(bookState) bool { return false })

	want := []Unavailable{
		{ID: 2, Slug: "heidi-b2", Status: StatusSold},
		{ID: 99, Status: StatusMissing}, // no row: the slug is not known
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
}
