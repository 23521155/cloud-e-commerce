// Package seeddata turns the scraped Amazon antique-books dataset into
// clean catalog rows for seeding the database.
package seeddata

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	// USDToVND matches the conversion used by the web's placeholder catalogue.
	USDToVND = 25_000

	// RareBeforeYear marks books printed before this year as rare.
	RareBeforeYear = 1900

	// UncategorizedName is the subject for books without an Amazon category.
	UncategorizedName = "Uncategorized"

	// placeholderImageID is Amazon's "no image available" picture.
	placeholderImageID = "01RmK+J4pJL"

	maxSlugBase = 80
)

var (
	ErrNoPrice  = errors.New("no usable price")
	ErrNotABook = errors.New("format is not a book")
)

// nonBookFormats are listing formats sold alongside books on Amazon that
// do not belong in a bookshop catalogue.
var nonBookFormats = map[string]bool{
	"Sheet music":           true,
	"Single Issue Magazine": true,
	"Print Magazine":        true,
	"Kindle Edition":        true,
	"CD-ROM":                true,
	"DVD-ROM":               true,
	"MP3 CD":                true,
	"Audio CD":              true,
	"Cards":                 true,
}

// publisherDate matches the trailing "(January 1, 1945)" on publisher strings.
var publisherDate = regexp.MustCompile(`\s*\([^()]*\d{4}\)$`)

// RawBook is one line of apps/web/ref/antique_books.jsonl.
type RawBook struct {
	ASIN        string   `json:"asin"`
	Title       string   `json:"title"`
	Subtitle    *string  `json:"subtitle"`
	Author      *string  `json:"author"`
	Year        int      `json:"year"`
	Publisher   *string  `json:"publisher"`
	Language    *string  `json:"language"`
	Pages       *string  `json:"pages"`
	Binding     *string  `json:"binding"`
	Price       any      `json:"price"` // number, "from 20.00", "�" or null
	Rating      *float64 `json:"rating"`
	RatingCount int      `json:"rating_count"`
	Categories  []string `json:"categories"`
	Image       *string  `json:"image"`
	Description *string  `json:"description"`
}

// Book is one cleaned row, as stored in seed/books.jsonl.gz.
type Book struct {
	ASIN        string   `json:"asin"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Author      *string  `json:"author"`
	Year        int      `json:"year"`
	Publisher   *string  `json:"publisher"`
	Edition     *string  `json:"edition"`
	Binding     *string  `json:"binding"`
	Language    *string  `json:"language"`
	Pages       *int     `json:"pages"`
	PriceVND    int64    `json:"price_vnd"`
	Rating      *float64 `json:"rating"`
	RatingCount int      `json:"rating_count"`
	Description *string  `json:"description"`
	CoverURL    *string  `json:"cover_url"`
	SubjectSlug string   `json:"subject_slug"`
	SubjectName string   `json:"subject_name"`
	Rare        bool     `json:"rare"`
}

// Clean converts a raw row into a Book. It returns ErrNoPrice or
// ErrNotABook when the row should be left out of the catalogue.
func Clean(r RawBook) (Book, error) {

	if nonBookFormats[format(r.Subtitle)] {
		return Book{}, ErrNotABook
	}

	usd, ok := parsePriceUSD(r.Price)
	if !ok {
		return Book{}, ErrNoPrice
	}

	title := cleanTitle(r.Title)
	publisher, edition := splitPublisher(r.Publisher)

	subject := UncategorizedName
	if len(r.Categories) > 1 {
		subject = r.Categories[1]
	}

	return Book{
		ASIN:        r.ASIN,
		Slug:        bookSlug(title, r.ASIN),
		Title:       title,
		Author:      optional(r.Author),
		Year:        r.Year,
		Publisher:   publisher,
		Edition:     edition,
		Binding:     binding(r.Binding, r.Subtitle),
		Language:    optional(r.Language),
		Pages:       parsePages(r.Pages),
		PriceVND:    int64(usd*USDToVND + 0.5),
		Rating:      r.Rating,
		RatingCount: r.RatingCount,
		Description: optional(r.Description),
		CoverURL:    coverURL(r.Image),
		SubjectSlug: Slugify(subject),
		SubjectName: subject,
		Rare:        r.Year < RareBeforeYear,
	}, nil
}

// Fold lowercases s and strips diacritics, so "Pháp" matches "phap".
func Fold(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	folded, _, err := transform.String(t, s)
	if err != nil {
		folded = s
	}
	return strings.ToLower(folded)
}

// SearchText is the value stored in books.search_text for LIKE queries.
func SearchText(b Book) string {
	if b.Author == nil {
		return Fold(b.Title)
	}
	return Fold(b.Title + " " + *b.Author)
}

// Slugify turns s into lowercase ASCII words joined by "-".
func Slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range Fold(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		// Apostrophes join words: "Children's" → "childrens".
		if r == '\'' || r == '’' {
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// bookSlug appends the ASIN so books sharing a title get distinct slugs.
func bookSlug(title, asin string) string {
	base := Slugify(title)
	if len(base) > maxSlugBase {
		base = strings.TrimRight(base[:maxSlugBase], "-")
	}
	if base == "" {
		return strings.ToLower(asin)
	}
	return base + "-" + strings.ToLower(asin)
}

// parsePriceUSD accepts a positive number or a "from 20.00" string.
func parsePriceUSD(v any) (float64, bool) {
	switch p := v.(type) {
	case float64:
		return p, p > 0
	case string:
		amount, ok := strings.CutPrefix(strings.TrimSpace(p), "from ")
		if !ok {
			return 0, false
		}
		f, err := strconv.ParseFloat(amount, 64)
		return f, err == nil && f > 0
	}
	return 0, false
}

// cleanTitle fixes the library-catalogue punctuation in scraped titles,
// e.g. "Marigold garden;: Pictures and rhymes," → "Marigold garden: Pictures and rhymes".
func cleanTitle(t string) string {
	t = strings.ReplaceAll(t, ";:", ":")
	return strings.TrimRight(strings.TrimSpace(t), ",;:/ ")
}

// splitPublisher turns "Random House; 1st edition (January 1, 1945)"
// into ("Random House", "1st edition").
func splitPublisher(s *string) (publisher, edition *string) {
	if s == nil {
		return nil, nil
	}
	trimmed := publisherDate.ReplaceAllString(strings.TrimSpace(*s), "")
	name, ed, _ := strings.Cut(trimmed, ";")
	return optionalString(name), optionalString(ed)
}

// format is the listing format, the part of "Hardcover – January 1, 1945"
// before the dash.
func format(subtitle *string) string {
	if subtitle == nil {
		return ""
	}
	f, _, _ := strings.Cut(*subtitle, " – ")
	return strings.TrimSpace(f)
}

// binding prefers the binding field and falls back to the subtitle format.
func binding(field, subtitle *string) *string {
	if b := optional(field); b != nil {
		return b
	}
	// Without a dash the subtitle is an edition note ("First Edition"), not a binding.
	if subtitle == nil || !strings.Contains(*subtitle, " – ") {
		return nil
	}
	f := format(subtitle)
	if f == "Unknown Binding" {
		return nil
	}
	return optionalString(f)
}

// parsePages reads "272 pages" as 272.
func parsePages(s *string) *int {
	if s == nil {
		return nil
	}
	fields := strings.Fields(*s)
	if len(fields) == 0 {
		return nil
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil || n <= 0 {
		return nil
	}
	return &n
}

// coverURL drops Amazon's placeholder image so the UI shows its own fallback.
func coverURL(image *string) *string {
	if image == nil || strings.Contains(*image, placeholderImageID) {
		return nil
	}
	return optional(image)
}

func optional(s *string) *string {
	if s == nil {
		return nil
	}
	return optionalString(*s)
}

func optionalString(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
