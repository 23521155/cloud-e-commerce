package book

import (
	"fmt"
	"time"
)

// Book statuses. MISSING is not stored: it is reported for a book that is not
// in the catalogue at all.
const (
	StatusAvailable = "AVAILABLE"
	StatusReserved  = "RESERVED"
	StatusSold      = "SOLD"
	StatusMissing   = "MISSING"
)

// Limits of one reservation request. The gRPC server checks them before a
// request reaches the database (see MaxBatch for the number of books).
const (
	MaxHolderLen   = 16 // reserved_by is NVARCHAR(16): an order code
	MaxHoldSeconds = 3600
	MaxSlugLen     = 100 // books.slug is NVARCHAR(100)
)

// Held is a reserved book as the order service needs it: the snapshot it
// stores in the order, including the price read at the moment of the hold.
type Held struct {
	ID       int     `db:"id"        json:"id"`
	Slug     string  `db:"slug"      json:"slug"`
	Title    string  `db:"title"     json:"title"`
	Author   *string `db:"author"    json:"author"`
	CoverURL *string `db:"cover_url" json:"coverUrl"`
	PriceVND int64   `db:"price_vnd" json:"priceVnd"`

	ReservedUntil time.Time `db:"reserved_until" json:"-"`
}

// Reservation is the result of a successful Reserve.
type Reservation struct {
	Items         []Held    `json:"items"`
	ReservedUntil time.Time `json:"reservedUntil"`
}

// Unavailable is one book that could not be reserved or sold, with the
// status that stopped it so the shop can say "sold" or "someone is paying".
//
// A request names books by slug (Reserve) or by id (Sell), and the other one
// is filled in when the book exists. For a MISSING book only what the caller
// asked with is known: ID is 0 for a slug, Slug is "" for an id.
type Unavailable struct {
	ID     int    `json:"id"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
}

// UnavailableError is returned when a request could not be applied to every
// book. Nothing is changed in that case: reservations are all or nothing.
type UnavailableError struct {
	Books []Unavailable
}

func (e *UnavailableError) Error() string {
	return fmt.Sprintf("%d books unavailable", len(e.Books))
}
