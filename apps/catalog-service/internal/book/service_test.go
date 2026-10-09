package book

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeStore struct {
	gotFilter  Filter
	gotLimit   int
	listErr    error
	exists     bool
	relatedHit bool
}

func (f *fakeStore) List(_ context.Context, filter Filter) ([]Book, int, error) {
	f.gotFilter = filter
	return []Book{{Slug: "a"}}, 42, f.listErr
}

func (f *fakeStore) GetBySlug(_ context.Context, slug string) (Book, error) {
	if !f.exists {
		return Book{}, ErrNotFound
	}
	return Book{Slug: slug}, nil
}

func (f *fakeStore) Related(_ context.Context, _ string, limit int) ([]Book, error) {
	f.relatedHit = true
	f.gotLimit = limit
	return nil, nil
}

func TestListNormalisesInput(t *testing.T) {
	tests := []struct {
		name string
		in   ListInput
		want Filter
	}{
		{
			"defaults",
			ListInput{},
			Filter{Sort: "new", Page: 1},
		},
		{
			"page below 1",
			ListInput{Page: -3, Sort: "year"},
			Filter{Sort: "year", Page: 1},
		},
		{
			"subject is trimmed and lowercased",
			ListInput{Subject: "  History ", Rare: true, Page: 2},
			Filter{Subject: "history", Rare: true, Sort: "new", Page: 2},
		},
		{
			"search is folded and split",
			ListInput{Q: "  Les   MISÉRABLES ", Page: 1},
			Filter{Terms: []string{"les", "miserables"}, Sort: "new", Page: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}

			if _, err := NewService(store).List(context.Background(), tt.in); err != nil {
				t.Fatal(err)
			}

			got := store.gotFilter
			// An empty search yields an empty, not nil, slice; both mean "no terms".
			if len(got.Terms) == 0 {
				got.Terms = nil
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filter = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListBuildsPage(t *testing.T) {
	page, err := NewService(&fakeStore{}).List(context.Background(), ListInput{Page: 3})
	if err != nil {
		t.Fatal(err)
	}

	if page.Total != 42 || page.Page != 3 || page.PageSize != PageSize || len(page.Items) != 1 {
		t.Errorf("page = %+v", page)
	}
}

func TestListPropagatesError(t *testing.T) {
	boom := errors.New("boom")

	if _, err := NewService(&fakeStore{listErr: boom}).List(context.Background(), ListInput{}); !errors.Is(err, boom) {
		t.Errorf("err = %v, want %v", err, boom)
	}
}

func TestRelatedLimit(t *testing.T) {
	tests := []struct{ in, want int }{
		{0, DefaultRelatedLimit},
		{-1, DefaultRelatedLimit},
		{6, 6},
		{500, MaxRelatedLimit},
	}

	for _, tt := range tests {
		store := &fakeStore{exists: true}

		if _, err := NewService(store).Related(context.Background(), "x", tt.in); err != nil {
			t.Fatal(err)
		}

		if store.gotLimit != tt.want {
			t.Errorf("limit %d → %d, want %d", tt.in, store.gotLimit, tt.want)
		}
	}
}

func TestRelatedUnknownSlug(t *testing.T) {
	store := &fakeStore{exists: false}

	_, err := NewService(store).Related(context.Background(), "nope", 4)

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
	if store.relatedHit {
		t.Error("Related query ran for a slug that does not exist")
	}
}
