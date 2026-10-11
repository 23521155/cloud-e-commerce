package grpcserver

import (
	"context"
	"errors"

	"github.com/fizzisme/catalog-service/internal/book"
	"github.com/fizzisme/catalog-service/internal/logger"
	catalogv1 "github.com/fizzisme/catalog-service/internal/pb/catalog/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// toStatusError turns an error from the reservation service into the gRPC
// status the caller sees. Anything unexpected is logged here and answered
// with a generic message: an error that is not a status reaches the client
// as codes.Unknown with its full text, so SQL errors would leak.
func toStatusError(method string, err error) error {

	var unavailable *book.UnavailableError

	switch {
	case errors.As(err, &unavailable):
		return unavailableStatus(unavailable)

	// The caller gave up; nothing is wrong with the service.
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")

	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	}

	logger.Log.Error(
		"reservation request failed",
		zap.String("method", method),
		zap.Error(err),
	)

	return status.Error(codes.Internal, "internal error")
}

// unavailableStatus answers FAILED_PRECONDITION: the request was valid but
// the books are not in a state that allows it. The details tell the caller
// which books, and why, so the shop can say "sold" or "someone is paying".
func unavailableStatus(e *book.UnavailableError) error {

	detail := &catalogv1.UnavailableBooks{
		Books: make([]*catalogv1.Unavailable, len(e.Books)),
	}
	for i, b := range e.Books {
		detail.Books[i] = &catalogv1.Unavailable{
			Id:     int32(b.ID),
			Status: toBookStatus(b.Status),
		}
	}

	st := status.New(codes.FailedPrecondition, "books unavailable")

	withDetail, err := st.WithDetails(detail)
	if err != nil {
		// Without the details it is still a correct answer.
		logger.Log.Error("cannot attach the unavailable books to the status", zap.Error(err))
		return st.Err()
	}

	return withDetail.Err()
}

func toBookStatus(s string) catalogv1.BookStatus {

	switch s {
	case book.StatusAvailable:
		return catalogv1.BookStatus_BOOK_STATUS_AVAILABLE
	case book.StatusReserved:
		return catalogv1.BookStatus_BOOK_STATUS_RESERVED
	case book.StatusSold:
		return catalogv1.BookStatus_BOOK_STATUS_SOLD
	case book.StatusMissing:
		return catalogv1.BookStatus_BOOK_STATUS_MISSING
	}

	return catalogv1.BookStatus_BOOK_STATUS_UNSPECIFIED
}
