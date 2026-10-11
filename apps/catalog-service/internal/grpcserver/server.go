package grpcserver

import (
	"github.com/fizzisme/catalog-service/internal/book"
	catalogv1 "github.com/fizzisme/catalog-service/internal/pb/catalog/v1"
	"google.golang.org/grpc"
)

// New builds the gRPC server: the reservation service behind the token
// check. Serving it is left to the caller, which owns the listener.
func New(token string, svc *book.ReservationService) *grpc.Server {

	srv := grpc.NewServer(grpc.UnaryInterceptor(UnaryTokenAuth(token)))

	catalogv1.RegisterReservationServiceServer(srv, NewReservationServer(svc))

	return srv
}
