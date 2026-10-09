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

	// MaxBatch is the most slugs one BySlugs call accepts.
	MaxBatch = 50
)

// store is what the service needs from the repository; tests replace it
// with a fake so they do not need a database.
type store interface {
	List(ctx context.Context, f Filter) ([]Book, int, error)
	GetBySlug(ctx context.Context, slug string) (Book, error)
	Related(ctx context.Context, slug string, limit int) ([]Book, error)
	BySlugs(ctx context.Context, slugs []string) ([]Book, error)
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

// BySlugs returns the books for slugs in the order requested. Duplicates are
// collapsed and unknown slugs are left out.
func (s *Service) BySlugs(ctx context.Context, slugs []string) ([]Book, error) {

	seen := make(map[string]bool, len(slugs))
	unique := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		if slug != "" && !seen[slug] {
			seen[slug] = true
			unique = append(unique, slug)
		}
	}

	found, err := s.store.BySlugs(ctx, unique)
	if err != nil {
		return nil, err
	}

	bySlug := make(map[string]Book, len(found))
	for _, b := range found {
		bySlug[b.Slug] = b
	}

	ordered := make([]Book, 0, len(found))
	for _, slug := range unique {
		if b, ok := bySlug[slug]; ok {
			ordered = append(ordered, b)
		}
	}

	return ordered, nil
}
