package grpcserver

import (
	"context"
	"crypto/subtle"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// tokenMetadataKey carries the shared secret of service-to-service calls.
// gRPC lower-cases metadata keys, so it is written in lower case here too.
const tokenMetadataKey = "x-internal-token"

// UnaryTokenAuth returns an interceptor that lets a call through only when
// its x-internal-token metadata equals token. Only unary calls go through
// it: add a stream interceptor too if a streaming RPC is ever defined.
func UnaryTokenAuth(token string) grpc.UnaryServerInterceptor {

	want := []byte(token)

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

		// No metadata at all gives a nil MD, and Get on a nil MD is safe.
		md, _ := metadata.FromIncomingContext(ctx)

		var got []byte
		if values := md.Get(tokenMetadataKey); len(values) > 0 {
			got = []byte(values[0])
		}

		// An empty configured token must never match a missing one, or
		// every call without metadata would be let in.
		if len(want) == 0 || subtle.ConstantTimeCompare(got, want) != 1 {
			// The same answer for a missing and a wrong token, and the
			// token itself is never logged.
			return nil, status.Error(codes.Unauthenticated, "unauthenticated")
		}

		return handler(ctx, req)
	}
}
