package catalogclient

import (
	"fmt"
	"time"
)

// Book is a book as the catalogue reported it when it was reserved: the
// snapshot an order stores, including the price at that moment.
type Book struct {
	ID       int
	Slug     string
	Title    string
	Author   *string
	CoverURL *string
	PriceVND int64
}

// Reservation is a successful hold on books.
type Reservation struct {
	Items []Book

	// ReservedUntil is when the hold ends; after that the books can be
	// taken by someone else.
	ReservedUntil time.Time
}

// Statuses of a book in the catalogue. MISSING is not stored there: it is
// reported for an id that is not in the catalogue at all. UNKNOWN is what
// this client says when the catalogue sent a status it does not recognise.
const (
	StatusAvailable = "AVAILABLE"
	StatusReserved  = "RESERVED"
	StatusSold      = "SOLD"
	StatusMissing   = "MISSING"
	StatusUnknown   = "UNKNOWN"
)

// Unavailable is one book that stopped a request, and why.
//
// Reserve names books by slug and Sell by id; the other name is filled in
// when the book exists. For a MISSING book only the name the caller used is
// known: ID is 0 for a slug, Slug is "" for an id.
type Unavailable struct {
	ID     int
	Slug   string
	Status string
}

// UnavailableError is returned when a reservation or a sale could not be
// applied to every book. Nothing was changed in the catalogue.
type UnavailableError struct {
	Books []Unavailable
}

func (e *UnavailableError) Error() string {
	return fmt.Sprintf("%d books unavailable", len(e.Books))
}
