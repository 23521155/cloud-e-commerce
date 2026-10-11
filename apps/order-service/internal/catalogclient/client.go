package catalogclient

import (
	"context"
	"crypto/tls"
	"errors"
	"time"

	catalogv1 "github.com/fizzisme/order-service/internal/pb/catalog/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// tokenMetadataKey carries the shared secret the catalogue checks on every
// call. gRPC lower-cases metadata keys, so it is written in lower case here.
const tokenMetadataKey = "x-internal-token"

// DefaultTimeout bounds a call whose caller did not set a deadline.
const DefaultTimeout = 3 * time.Second

// Config says where the catalogue's gRPC server is and how to reach it.
type Config struct {
	// Addr is host:port of the catalogue's gRPC server.
	Addr string

	// Token is sent as x-internal-token on every call.
	Token string

	// UseTLS must be on outside local runs: Azure App Service only accepts
	// gRPC over TLS. Local containers speak plain text.
	UseTLS bool

	// Timeout bounds a call whose context has no deadline. Zero means
	// DefaultTimeout.
	Timeout time.Duration
}

// Client calls the catalogue's reservation service. Only this package knows
// the protobuf types; the rest of the order service sees plain Go types.
type Client struct {
	conn *grpc.ClientConn
	rpc  catalogv1.ReservationServiceClient
}

// New prepares a client. The connection is lazy: it is opened by the first
// call, so New does not fail when the catalogue is not up yet.
func New(cfg Config) (*Client, error) {
	return newClient(cfg)
}

// newClient is New with extra dial options, so tests can dial an in-memory
// server.
func newClient(cfg Config, extra ...grpc.DialOption) (*Client, error) {

	if cfg.Addr == "" {
		return nil, errors.New("catalogclient: Addr is required")
	}

	// The catalogue rejects every call without a token, so a client without
	// one could never work: fail at start-up, not on the first order.
	if cfg.Token == "" {
		return nil, errors.New("catalogclient: Token is required")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	creds := insecure.NewCredentials()
	if cfg.UseTLS {
		creds = credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
	}

	opts := append([]grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(callDefaults(cfg.Token, timeout)),
	}, extra...)

	conn, err := grpc.NewClient(cfg.Addr, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn: conn,
		rpc:  catalogv1.NewReservationServiceClient(conn),
	}, nil
}

// Close releases the connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// callDefaults returns the interceptor every call goes through: it adds the
// token, and a deadline when the caller gave none.
func callDefaults(token string, timeout time.Duration) grpc.UnaryClientInterceptor {

	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		ctx = metadata.AppendToOutgoingContext(ctx, tokenMetadataKey, token)

		// A deadline the caller chose wins; without one a call must not be
		// able to hang for ever.
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
