package grpcserver

import (
	"context"
	"testing"

	catalogv1 "github.com/fizzisme/catalog-service/internal/pb/catalog/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// releaseExpired is the simplest call: no request data, one RPC.
func releaseExpired(ctx context.Context, client catalogv1.ReservationServiceClient) error {
	_, err := client.ReleaseExpired(ctx, &catalogv1.ReleaseExpiredRequest{})
	return err
}

func TestTokenAuth(t *testing.T) {

	const secret = "s3cret"

	tests := []struct {
		name       string
		configured string
		send       func(ctx context.Context) context.Context
		wantOK     bool
	}{
		{
			"right token", secret,
			func(ctx context.Context) context.Context {
				return metadata.AppendToOutgoingContext(ctx, "x-internal-token", secret)
			},
			true,
		},
		{
			"right token with other letter case in the key", secret,
			func(ctx context.Context) context.Context {
				return metadata.AppendToOutgoingContext(ctx, "X-Internal-Token", secret)
			},
			true,
		},
		{
			"wrong token", secret,
			func(ctx context.Context) context.Context {
				return metadata.AppendToOutgoingContext(ctx, "x-internal-token", "guess")
			},
			false,
		},
		{
			"no metadata at all", secret,
			func(ctx context.Context) context.Context { return ctx },
			false,
		},
		{
			"other metadata but no token", secret,
			func(ctx context.Context) context.Context {
				return metadata.AppendToOutgoingContext(ctx, "x-request-id", "abc")
			},
			false,
		},
		{
			"empty token", secret,
			func(ctx context.Context) context.Context {
				return metadata.AppendToOutgoingContext(ctx, "x-internal-token", "")
			},
			false,
		},
		{
			"token sent with a different length", secret,
			func(ctx context.Context) context.Context {
				return metadata.AppendToOutgoingContext(ctx, "x-internal-token", secret+"x")
			},
			false,
		},
		// An unset INTERNAL_TOKEN must lock the door, not open it.
		{
			"nothing configured, nothing sent", "",
			func(ctx context.Context) context.Context { return ctx },
			false,
		},
		{
			"nothing configured, empty token sent", "",
			func(ctx context.Context) context.Context {
				return metadata.AppendToOutgoingContext(ctx, "x-internal-token", "")
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			store := &fakeStore{}
			client := newClient(t, store, grpc.UnaryInterceptor(UnaryTokenAuth(tt.configured)))

			err := releaseExpired(tt.send(context.Background()), client)

			if tt.wantOK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if status.Code(err) != codes.Unauthenticated {
				t.Fatalf("code = %v, want Unauthenticated (err: %v)", status.Code(err), err)
			}
			if store.calls != 0 {
				t.Fatalf("the store was reached %d times by a rejected call", store.calls)
			}
		})
	}
}

// A missing and a wrong token must look the same from outside.
func TestTokenAuthAnswersTheSameForMissingAndWrong(t *testing.T) {

	client := newClient(t, &fakeStore{}, grpc.UnaryInterceptor(UnaryTokenAuth("s3cret")))

	missing := releaseExpired(context.Background(), client)
	wrong := releaseExpired(metadata.AppendToOutgoingContext(context.Background(), "x-internal-token", "guess"), client)

	if status.Convert(missing).Message() != status.Convert(wrong).Message() {
		t.Fatalf("messages differ: %q vs %q", status.Convert(missing).Message(), status.Convert(wrong).Message())
	}
}

// The interceptor covers every RPC of the service, not only one.
func TestTokenAuthGuardsEveryRPC(t *testing.T) {

	store := &fakeStore{}
	client := newClient(t, store, grpc.UnaryInterceptor(UnaryTokenAuth("s3cret")))
	ctx := context.Background()

	calls := map[string]func() error{
		"Reserve": func() error {
			_, err := client.Reserve(ctx, &catalogv1.ReserveRequest{Holder: "BK-1", TtlSeconds: 60, Slugs: []string{"heidi-b1"}})
			return err
		},
		"Release": func() error {
			_, err := client.Release(ctx, &catalogv1.ReleaseRequest{Holder: "BK-1", Ids: []int32{1}})
			return err
		},
		"Sell": func() error {
			_, err := client.Sell(ctx, &catalogv1.SellRequest{Holder: "BK-1", Ids: []int32{1}})
			return err
		},
		"ReleaseExpired": func() error { return releaseExpired(ctx, client) },
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			if got := status.Code(call()); got != codes.Unauthenticated {
				t.Fatalf("code = %v, want Unauthenticated", got)
			}
		})
	}

	if store.calls != 0 {
		t.Fatalf("the store was reached %d times without a token", store.calls)
	}
}
