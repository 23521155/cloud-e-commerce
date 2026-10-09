package subject

// Subject is a book category with the number of books that can be bought.
type Subject struct {
	ID        int    `db:"id"         json:"-"`
	Slug      string `db:"slug"       json:"slug"`
	Name      string `db:"name"       json:"name"`
	BookCount int    `db:"book_count" json:"bookCount"`
}
