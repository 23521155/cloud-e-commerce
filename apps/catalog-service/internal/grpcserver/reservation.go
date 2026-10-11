package grpcserver

import (
	"context"

	"github.com/fizzisme/catalog-service/internal/book"
	catalogv1 "github.com/fizzisme/catalog-service/internal/pb/catalog/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ReservationServer struct {
	catalogv1.UnimplementedReservationServiceServer
	svc *book.ReservationService
}

// Fails the build, instead of the registration at start-up, if a method
// has the wrong signature.
var _ catalogv1.ReservationServiceServer = (*ReservationServer)(nil)

func NewReservationServer(svc *book.ReservationService) *ReservationServer {
	return &ReservationServer{svc: svc}
}

func toInts(in []int32) []int {
	out := make([]int, len(in))
	for i, v := range in {
		out[i] = int(v)
	}
	return out
}

// validateHolderAndIDs rejects requests that would otherwise reach SQL Server
// and fail there ("IN ()" is a syntax error, a long holder overflows its
// column). gRPC checks nothing about the values inside a message.
func validateHolderAndIDs(holder string, ids []int32) error {

	if holder == "" || len(holder) > book.MaxHolderLen {
		return status.Errorf(codes.InvalidArgument, "holder must be 1 to %d characters", book.MaxHolderLen)
	}

	if len(ids) == 0 || len(ids) > book.MaxBatch {
		return status.Errorf(codes.InvalidArgument, "ids must hold 1 to %d books", book.MaxBatch)
	}

	for _, id := range ids {
		if id <= 0 {
			return status.Error(codes.InvalidArgument, "ids must be positive")
		}
	}

	return nil
}

func (s *ReservationServer) Release(ctx context.Context, req *catalogv1.ReleaseRequest) (*catalogv1.ReleaseResponse, error) {
	if err := validateHolderAndIDs(req.GetHolder(), req.GetIds()); err != nil {
		return nil, err
	}

	n, err := s.svc.Release(ctx, req.GetHolder(), toInts(req.GetIds()))
	if err != nil {
		return nil, toStatusError("Release", err)
	}
	return &catalogv1.ReleaseResponse{Released: int32(n)}, nil
}

func (s *ReservationServer) Sell(ctx context.Context, req *catalogv1.SellRequest) (*catalogv1.SellResponse, error) {
	if err := validateHolderAndIDs(req.GetHolder(), req.GetIds()); err != nil {
		return nil, err
	}

	err := s.svc.Sell(ctx, req.GetHolder(), toInts(req.GetIds()))
	if err != nil {
		return nil, toStatusError("Sell", err)
	}
	return &catalogv1.SellResponse{}, nil
}

func (s *ReservationServer) ReleaseExpired(ctx context.Context, req *catalogv1.ReleaseExpiredRequest) (*catalogv1.ReleaseExpiredResponse, error) {
	n, err := s.svc.ReleaseExpired(ctx)
	if err != nil {
		return nil, toStatusError("ReleaseExpired", err)
	}
	return &catalogv1.ReleaseExpiredResponse{Released: int32(n)}, nil
}

func (s *ReservationServer) Reserve(ctx context.Context, req *catalogv1.ReserveRequest) (*catalogv1.ReserveResponse, error) {
	if err := validateHolderAndIDs(req.GetHolder(), req.GetIds()); err != nil {
		return nil, err
	}

	// The upper bound matters: the repository narrows the ttl to int32.
	if ttl := req.GetTtlSeconds(); ttl <= 0 || ttl > book.MaxHoldSeconds {
		return nil, status.Errorf(codes.InvalidArgument, "ttl_seconds must be 1 to %d", book.MaxHoldSeconds)
	}

	r, err := s.svc.Reserve(ctx, req.GetHolder(), int(req.GetTtlSeconds()), toInts(req.GetIds()))
	if err != nil {
		return nil, toStatusError("Reserve", err)
	}

	items := make([]*catalogv1.Held, len(r.Items))
	for i, h := range r.Items {
		items[i] = &catalogv1.Held{
			Id:       int32(h.ID),
			Slug:     h.Slug,
			Title:    h.Title,
			Author:   h.Author,
			CoverUrl: h.CoverURL,
			PriceVnd: h.PriceVND,
		}
	}
	return &catalogv1.ReserveResponse{
		Items:         items,
		ReservedUntil: timestamppb.New(r.ReservedUntil),
	}, nil
}
