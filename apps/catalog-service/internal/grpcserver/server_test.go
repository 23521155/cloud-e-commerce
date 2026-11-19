package grpcserver

import (
	"context"
	"net"
	"testing"

	"github.com/fizzisme/catalog-service/internal/book"
	"github.com/fizzisme/catalog-service/internal/logger"
	catalogv1 "github.com/fizzisme/catalog-service/internal/pb/catalog/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// New is what main.go runs, so this checks the wiring as a whole: that the
// token check is really installed and the service is really registered.
func TestNewGuardsTheServiceWithTheToken(t *testing.T) {

	if err := logger.Init(); err != nil {
		t.Fatal(err)
	}

	store := &fakeStore{}
	lis := bufconn.Listen(1 << 20)

	srv := New("s3cret", book.NewReservationService(store))
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	client := catalogv1.NewReservationServiceClient(conn)

	// No token: refused, and the store is never reached.
	if err := releaseExpired(context.Background(), client); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("without a token: code = %v, want Unauthenticated", status.Code(err))
	}
	if store.calls != 0 {
		t.Fatalf("the store was reached %d times without a token", store.calls)
	}

	// Right token: the service answers.
	ctx := metadata.AppendToOutgoingContext(context.Background(), "x-internal-token", "s3cret")
	if err := releaseExpired(ctx, client); err != nil {
		t.Fatalf("with the token: %v", err)
	}
	if store.calls != 1 {
		t.Fatalf("store calls = %d, want 1", store.calls)
	}
}
