package catalogclient

import (
	"context"
	"fmt"
	"time"

	catalogv1 "github.com/fizzisme/order-service/internal/pb/catalog/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Reserve holds every book in ids for holder (an order code) for ttl, or none
// of them. ttl is counted in whole seconds and must be at least one.
//
// When some books cannot be held the error is an *UnavailableError. An
// ErrCatalogUnavailable error means the outcome is unknown: Release the books.
func (c *Client) Reserve(ctx context.Context, holder string, ttl time.Duration, ids []int) (Reservation, error) {

	if ttl < time.Second {
		return Reservation{}, fmt.Errorf("%w: the hold must last at least one second", ErrInvalidRequest)
	}

	res, err := c.rpc.Reserve(ctx, &catalogv1.ReserveRequest{
		Holder:     holder,
		TtlSeconds: int64(ttl / time.Second),
		Ids:        toInt32s(ids),
	})
	if err != nil {
		return Reservation{}, translate(ctx, err)
	}

	items := make([]Book, len(res.GetItems()))
	for i, it := range res.GetItems() {
		items[i] = Book{
			ID:       int(it.GetId()),
			Slug:     it.GetSlug(),
			Title:    it.GetTitle(),
			Author:   it.Author,
			CoverURL: it.CoverUrl,
			PriceVND: it.GetPriceVnd(),
		}
	}

	return Reservation{
		Items:         items,
		ReservedUntil: res.GetReservedUntil().AsTime(),
	}, nil
}

// Release gives back the books holder holds and returns how many it freed.
// Books that are not held by holder are left alone, so it is safe to call
// again.
func (c *Client) Release(ctx context.Context, holder string, ids []int) (int, error) {

	res, err := c.rpc.Release(ctx, &catalogv1.ReleaseRequest{
		Holder: holder,
		Ids:    toInt32s(ids),
	})
	if err != nil {
		return 0, translate(ctx, err)
	}

	return int(res.GetReleased()), nil
}

// Sell turns the books holder holds into sold ones, all or none. Selling what
// holder has already sold counts as done, so it is safe to call again. When a
// book is no longer held by holder (its hold ran out and someone else took
// it) the error is an *UnavailableError.
func (c *Client) Sell(ctx context.Context, holder string, ids []int) error {

	_, err := c.rpc.Sell(ctx, &catalogv1.SellRequest{
		Holder: holder,
		Ids:    toInt32s(ids),
	})

	return translate(ctx, err)
}

// toInt32s is how the ids go over the wire.
func toInt32s(ids []int) []int32 {

	out := make([]int32, len(ids))
	for i, id := range ids {
		out[i] = int32(id)
	}

	return out
}

// translate turns what a gRPC call returned into the errors of this package.
// It is the one place that knows which status code means what.
func translate(ctx context.Context, err error) error {

	if err == nil {
		return nil
	}

	st := status.Convert(err)

	switch st.Code() {

	case codes.FailedPrecondition:
		return unavailableFrom(st)

	case codes.Unauthenticated:
		return fmt.Errorf("%w: %w", ErrUnauthenticated, err)

	case codes.InvalidArgument:
		return fmt.Errorf("%w: %w", ErrInvalidRequest, err)

	// Only the caller giving up is not the catalogue's fault. A cancel
	// that did not come from the caller's own context is a catalogue
	// failure like any other.
	case codes.Canceled:
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("%w: %w", ctxErr, err)
		}
	}

	// A deadline, a server that is down, an internal error, anything else.
	return fmt.Errorf("%w: %w", ErrCatalogUnavailable, err)
}

// unavailableFrom reads which books stopped the request out of the details
// the catalogue attaches to a FAILED_PRECONDITION. Without details (the
// server could not attach them) the books are simply not known, but the
// request still failed for that reason.
func unavailableFrom(st *status.Status) error {

	e := &UnavailableError{}

	for _, d := range st.Details() {

		// An entry that failed to decode is an error, not a message.
		detail, ok := d.(*catalogv1.UnavailableBooks)
		if !ok {
			continue
		}

		for _, b := range detail.GetBooks() {
			e.Books = append(e.Books, Unavailable{
				ID:     int(b.GetId()),
				Status: fromBookStatus(b.GetStatus()),
			})
		}
	}

	return e
}

func fromBookStatus(s catalogv1.BookStatus) string {

	switch s {
	case catalogv1.BookStatus_BOOK_STATUS_AVAILABLE:
		return StatusAvailable
	case catalogv1.BookStatus_BOOK_STATUS_RESERVED:
		return StatusReserved
	case catalogv1.BookStatus_BOOK_STATUS_SOLD:
		return StatusSold
	case catalogv1.BookStatus_BOOK_STATUS_MISSING:
		return StatusMissing
	}

	// UNSPECIFIED, or a status a newer catalogue added.
	return StatusUnknown
}
