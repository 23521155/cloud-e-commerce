package book

import (
	"context"
	"strings"

	"github.com/fizzisme/catalog-service/internal/seeddata"
)

const (
	DefaultSort         = "new"
	DefaultRelatedLimit = 4
	MaxRelatedLimit     = 12
)

// store is what the service needs from the repository; tests replace it
// with a fake so they do not need a database.
type store interface {
	List(ctx context.Context, f Filter) ([]Book, int, error)
	GetBySlug(ctx context.Context, slug string) (Book, error)
	Related(ctx context.Context, slug string, limit int) ([]Book, error)
}

// ListInput is the raw listing request, as read from the query string.
type ListInput struct {
	Subject string
	Rare    bool
	Sort    string
	Q       string
	Page    int
}

// Page is one page of a listing.
type Page struct {
	Items    []Book `json:"items"`
	Total    int    `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type Service struct {
	store store
}

func NewService(store store) *Service {
	return &Service{store: store}
}

// List normalises the request into a Filter and returns the matching page.
func (s *Service) List(ctx context.Context, in ListInput) (Page, error) {

	f := Filter{
		Subject: strings.ToLower(strings.TrimSpace(in.Subject)),
		Rare:    in.Rare,
		Sort:    in.Sort,
		// search_text is stored lowercase without diacritics, so the
		// search words must be folded the same way.
		Terms: strings.Fields(seeddata.Fold(in.Q)),
		Page:  max(in.Page, 1),
	}
	if f.Sort == "" {
		f.Sort = DefaultSort
	}

	books, total, err := s.store.List(ctx, f)
	if err != nil {
		return Page{}, err
	}

	return Page{
		Items:    books,
		Total:    total,
		Page:     f.Page,
		PageSize: PageSize,
	}, nil
}

// Get returns one book, or ErrNotFound.
func (s *Service) Get(ctx context.Context, slug string) (Book, error) {
	return s.store.GetBySlug(ctx, slug)
}

// Related returns books similar to slug, or ErrNotFound when slug itself
// does not exist.
func (s *Service) Related(ctx context.Context, slug string, limit int) ([]Book, error) {

	if limit < 1 {
		limit = DefaultRelatedLimit
	}
	limit = min(limit, MaxRelatedLimit)

	// The repository query happily returns books for an unknown slug,
	// so check it exists first.
	if _, err := s.store.GetBySlug(ctx, slug); err != nil {
		return nil, err
	}

	return s.store.Related(ctx, slug, limit)
}
