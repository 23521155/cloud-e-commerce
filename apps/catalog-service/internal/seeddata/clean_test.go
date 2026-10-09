package seeddata

import (
	"errors"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestParsePriceUSD(t *testing.T) {
	tests := []struct {
		in     any
		want   float64
		wantOK bool
	}{
		{20.0, 20, true},
		{"from 12.50", 12.5, true},
		{"�", 0, false},
		{nil, 0, false},
		{0.0, 0, false},
		{"from abc", 0, false},
	}
	for _, tt := range tests {
		got, ok := parsePriceUSD(tt.in)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("parsePriceUSD(%v) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestSplitPublisher(t *testing.T) {
	tests := []struct {
		in            string
		name, edition string
	}{
		{"Random House; 1st edition (January 1, 1945)", "Random House", "1st edition"},
		{"Routledge (January 1, 1885)", "Routledge", ""},
		{"Oxford (Clarendon Press)", "Oxford (Clarendon Press)", ""},
	}
	for _, tt := range tests {
		name, edition := splitPublisher(ptr(tt.in))
		if deref(name) != tt.name || deref(edition) != tt.edition {
			t.Errorf("splitPublisher(%q) = %q, %q; want %q, %q", tt.in, deref(name), deref(edition), tt.name, tt.edition)
		}
	}
}

func TestSlugify(t *testing.T) {
	tests := map[string]string{
		"Literature & Fiction":   "literature-fiction",
		"Children's Books":       "childrens-books",
		"Les Misérables, Tome I": "les-miserables-tome-i",
		"  --  ":                 "",
	}
	for in, want := range tests {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestBookSlugTruncatesAndAppendsASIN(t *testing.T) {
	long := "a very long title that keeps going and going well past the eighty character limit for slugs"
	got := bookSlug(long, "B000ABC123")
	if len(got) > maxSlugBase+1+10 {
		t.Fatalf("slug too long: %d", len(got))
	}
	if got[len(got)-11:] != "-b000abc123" {
		t.Fatalf("slug %q does not end with the ASIN", got)
	}
}

func TestClean(t *testing.T) {
	raw := RawBook{
		ASIN:        "B00085SZFY",
		Title:       "Marigold garden;: Pictures and rhymes,",
		Subtitle:    ptr("Hardcover – January 1, 1885"),
		Author:      ptr("Kate Greenaway"),
		Year:        1885,
		Publisher:   ptr("Routledge (January 1, 1885)"),
		Pages:       ptr("64 pages"),
		Price:       130.0,
		Rating:      ptr(3.4),
		RatingCount: 27,
		Categories:  []string{"Books", "Children's Books", "Classics"},
		Image:       ptr("https://m.media-amazon.com/images/I/01RmK+J4pJL._BO1,204,203,200_.gif"),
	}

	b, err := Clean(raw)
	if err != nil {
		t.Fatal(err)
	}

	if b.Title != "Marigold garden: Pictures and rhymes" {
		t.Errorf("Title = %q", b.Title)
	}
	if b.Slug != "marigold-garden-pictures-and-rhymes-b00085szfy" {
		t.Errorf("Slug = %q", b.Slug)
	}
	if b.PriceVND != 3_250_000 {
		t.Errorf("PriceVND = %d", b.PriceVND)
	}
	if deref(b.Binding) != "Hardcover" {
		t.Errorf("Binding = %v", b.Binding)
	}
	if b.Pages == nil || *b.Pages != 64 {
		t.Errorf("Pages = %v", b.Pages)
	}
	if b.CoverURL != nil {
		t.Errorf("placeholder image kept: %q", *b.CoverURL)
	}
	if b.SubjectSlug != "childrens-books" || b.SubjectName != "Children's Books" {
		t.Errorf("Subject = %q / %q", b.SubjectSlug, b.SubjectName)
	}
	if !b.Rare {
		t.Error("book from 1885 should be rare")
	}
	if got := SearchText(b); got != "marigold garden: pictures and rhymes kate greenaway" {
		t.Errorf("SearchText = %q", got)
	}
}

func TestCleanDefaultsAndSkips(t *testing.T) {
	b, err := Clean(RawBook{ASIN: "B1", Title: "Iron Heel", Subtitle: ptr("First Edition"), Year: 1907, Price: "from 20.00"})
	if err != nil {
		t.Fatal(err)
	}
	if b.SubjectName != UncategorizedName || b.Binding != nil || b.Rare {
		t.Errorf("got subject %q, binding %v, rare %v", b.SubjectName, b.Binding, b.Rare)
	}

	if _, err := Clean(RawBook{ASIN: "B2", Title: "x", Year: 1900, Price: "�"}); !errors.Is(err, ErrNoPrice) {
		t.Errorf("missing price: err = %v", err)
	}
	if _, err := Clean(RawBook{ASIN: "B3", Title: "x", Year: 1900, Price: 5.0, Subtitle: ptr("Sheet music – January 1, 1920")}); !errors.Is(err, ErrNotABook) {
		t.Errorf("sheet music: err = %v", err)
	}
}

func TestCredibleYear(t *testing.T) {
	tests := []struct {
		name        string
		year, count int
		title       string
		want        bool
	}{
		{"modern placeholder year", 1656, 19, "Toy Story 3", false},
		{"just below the minimum", 1799, 1, "x", false},
		{"minimum", 1800, 1, "x", true},
		{"old and obscure", 1839, 6, "Life of George Washington", true},
		{"old but popular today", 1829, 914, "Body of Lies: A Novel", false},
		{"popular and the title names the year", 1885, 3393, "Personal Memoirs of U.S. Grant - 1st Edition 1885", true},
		{"popular but after 1900", 1944, 5000, "The Little Prince", true},
	}
	for _, tt := range tests {
		if got := credibleYear(tt.year, tt.count, tt.title); got != tt.want {
			t.Errorf("%s: credibleYear(%d, %d, %q) = %v; want %v", tt.name, tt.year, tt.count, tt.title, got, tt.want)
		}
	}
}

func TestSubjectName(t *testing.T) {
	tests := []struct {
		in   []string
		want string
	}{
		{nil, UncategorizedName},
		{[]string{"Books"}, UncategorizedName},
		{[]string{"Books", "History", "Military"}, "History"},
		{[]string{"Books", "Boxed Sets"}, UncategorizedName},
		{[]string{"Books", "Deals in Books"}, UncategorizedName},
		{[]string{"Books", "Libros en espa�ol"}, "Libros en español"},
	}
	for _, tt := range tests {
		if got := subjectName(tt.in); got != tt.want {
			t.Errorf("subjectName(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
	if got := Slugify(subjectName([]string{"Books", "Libros en espa�ol"})); got != "libros-en-espanol" {
		t.Errorf("slug = %q", got)
	}
}

func TestCleanDropsBogusYearAndNonBooks(t *testing.T) {
	price := 10.0

	if _, err := Clean(RawBook{ASIN: "B4", Title: "Prom Nights from Hell", Year: 1656, Price: price}); !errors.Is(err, ErrBadYear) {
		t.Errorf("bogus year: err = %v", err)
	}
	if _, err := Clean(RawBook{ASIN: "B5", Title: "Piano Sonata K123 Sheet Music (Piano)", Subtitle: ptr("Paperback – January 1, 1920"), Year: 1920, Price: price}); !errors.Is(err, ErrNotABook) {
		t.Errorf("sheet music in title: err = %v", err)
	}
	if _, err := Clean(RawBook{ASIN: "B6", Title: "Songs", Year: 1920, Price: price, Categories: []string{"Books", "Sheet Music & Scores"}}); !errors.Is(err, ErrNotABook) {
		t.Errorf("sheet music subject: err = %v", err)
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
