package book

// Book is one catalogue row joined with its subject. The db tags match
// the columns selected by the repository; the json tags are the API shape.
type Book struct {
	ID          int      `db:"id"           json:"-"`
	ASIN        string   `db:"asin"         json:"asin"`
	Slug        string   `db:"slug"         json:"slug"`
	Title       string   `db:"title"        json:"title"`
	Author      *string  `db:"author"       json:"author"`
	Year        int      `db:"year"         json:"year"`
	Publisher   *string  `db:"publisher"    json:"publisher"`
	Edition     *string  `db:"edition"      json:"edition"`
	Binding     *string  `db:"binding"      json:"binding"`
	Language    *string  `db:"language"     json:"language"`
	Pages       *int     `db:"pages"        json:"pages"`
	PriceVND    int64    `db:"price_vnd"    json:"priceVnd"`
	Rating      *float64 `db:"rating"       json:"rating"`
	RatingCount int      `db:"rating_count" json:"ratingCount"`
	Description *string  `db:"description"  json:"description"`
	CoverURL    *string  `db:"cover_url"    json:"coverUrl"`
	Rare        bool     `db:"rare"         json:"rare"`
	Status      string   `db:"status"       json:"status"`
	SubjectSlug string   `db:"subject_slug" json:"subjectSlug"`
	SubjectName string   `db:"subject_name" json:"subjectName"`
	// search_text is only used inside SQL, so it is not mapped here.
}

// Filter is the normalised input of a catalogue listing.
type Filter struct {
	Subject string
	Rare    bool
	Sort    string
	Terms   []string // words of the search text, already folded
	Page    int
}
