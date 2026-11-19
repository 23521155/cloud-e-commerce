package catalogclient

import "errors"

// What a failed call means for the caller. A sold-out book is not here: it is
// *UnavailableError, which carries which books and why. Use errors.Is for
// these and errors.As for that one; the gRPC error that caused each of them
// stays in the chain, so its message still reaches the logs.
var (
	// ErrCatalogUnavailable means the catalogue did not answer properly: it
	// is down, too slow, or failed inside.
	//
	// The outcome is UNKNOWN. A Reserve that timed out may still have held
	// the books, so a caller that gives up on an order must Release them
	// (releasing twice is safe).
	ErrCatalogUnavailable = errors.New("catalogue unavailable")

	// ErrUnauthenticated means the catalogue refused the token. That is a
	// configuration problem, not something a retry fixes.
	ErrUnauthenticated = errors.New("catalogue rejected the token")

	// ErrInvalidRequest means the request was malformed (no ids, a holder
	// that is too long...). That is a bug in the caller.
	ErrInvalidRequest = errors.New("invalid catalogue request")
)
