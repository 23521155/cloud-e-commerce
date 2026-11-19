package catalogclient

import (
	"context"
	"net"
	"testing"
	"time"

	catalogv1 "github.com/fizzisme/order-service/internal/pb/catalog/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// spyServer is a stand-in catalogue. It records what the client sent, and
// can be told to hold a call open to exercise deadlines.
type spyServer struct {
	catalogv1.UnimplementedReservationServiceServer

	tokens      []string
	hadDeadline bool
	remaining   time.Duration

	// delay is how long a call is held before it is answered; block holds
	// it until the client gives up.
	delay time.Duration
	block bool
}

func (s *spyServer) ReleaseExpired(ctx context.Context, _ *catalogv1.ReleaseExpiredRequest) (*catalogv1.ReleaseExpiredResponse, error) {

	md, _ := metadata.FromIncomingContext(ctx)
	s.tokens = md.Get(tokenMetadataKey)

	if d, ok := ctx.Deadline(); ok {
		s.hadDeadline = true
		s.remaining = time.Until(d)
	}

	if s.block {
		<-ctx.Done()
		return nil, status.FromContextError(ctx.Err()).Err()
	}

	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return nil, status.FromContextError(ctx.Err()).Err()
		}
	}

	return &catalogv1.ReleaseExpiredResponse{}, nil
}

// dialSpy runs a stand-in catalogue behind an in-memory connection and
// returns a client to it.
func dialSpy(t *testing.T, cfg Config, spy catalogv1.ReservationServiceServer) *Client {
	t.Helper()

	lis := bufconn.Listen(1 << 20)

	srv := grpc.NewServer()
	catalogv1.RegisterReservationServiceServer(srv, spy)
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	cfg.Addr = "passthrough:///bufnet"

	c, err := newClient(cfg, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })

	return c
}

func TestNewRejectsAnIncompleteConfig(t *testing.T) {

	tests := []struct {
		name string
		cfg  Config
	}{
		{"no address", Config{Token: "s3cret"}},
		{"no token", Config{Addr: "catalog-service:9083"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if c, err := New(tt.cfg); err == nil {
				c.Close()
				t.Fatal("New accepted an incomplete config")
			}
		})
	}
}

// The connection is lazy, so a catalogue that is not up yet must not stop
// the order service from starting.
func TestNewDoesNotNeedTheServer(t *testing.T) {

	c, err := New(Config{Addr: "127.0.0.1:1", Token: "s3cret"})
	if err != nil {
		t.Fatalf("New failed without a server: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestEveryCallCarriesTheToken(t *testing.T) {

	spy := &spyServer{}
	c := dialSpy(t, Config{Token: "s3cret"}, spy)

	if _, err := c.rpc.ReleaseExpired(context.Background(), &catalogv1.ReleaseExpiredRequest{}); err != nil {
		t.Fatal(err)
	}

	if len(spy.tokens) != 1 || spy.tokens[0] != "s3cret" {
		t.Fatalf("server saw tokens %v, want [s3cret]", spy.tokens)
	}
}

func TestACallWithoutADeadlineGetsOne(t *testing.T) {

	spy := &spyServer{}
	c := dialSpy(t, Config{Token: "s3cret", Timeout: 2 * time.Second}, spy)

	if _, err := c.rpc.ReleaseExpired(context.Background(), &catalogv1.ReleaseExpiredRequest{}); err != nil {
		t.Fatal(err)
	}

	if !spy.hadDeadline {
		t.Fatal("the server saw no deadline")
	}
	if spy.remaining <= 0 || spy.remaining > 2*time.Second {
		t.Fatalf("remaining = %v, want within (0, 2s]", spy.remaining)
	}
}

func TestTheDefaultTimeoutApplies(t *testing.T) {

	spy := &spyServer{}
	c := dialSpy(t, Config{Token: "s3cret"}, spy)

	if _, err := c.rpc.ReleaseExpired(context.Background(), &catalogv1.ReleaseExpiredRequest{}); err != nil {
		t.Fatal(err)
	}

	if spy.remaining <= 2*time.Second || spy.remaining > DefaultTimeout {
		t.Fatalf("remaining = %v, want close to %v", spy.remaining, DefaultTimeout)
	}
}

// A deadline the caller chose is never shortened by the client's default.
// Only a caller deadline LONGER than the default tells the two apart:
// context.WithTimeout never lengthens a parent's deadline.
func TestALongerCallerDeadlineIsNotCutByTheDefault(t *testing.T) {

	// The server answers after 300ms; the default would cut it at 100ms.
	spy := &spyServer{delay: 300 * time.Millisecond}
	c := dialSpy(t, Config{Token: "s3cret", Timeout: 100 * time.Millisecond}, spy)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := c.rpc.ReleaseExpired(ctx, &catalogv1.ReleaseExpiredRequest{}); err != nil {
		t.Fatalf("the caller allowed 5s but the call failed: %v", err)
	}
}

func TestAShorterCallerDeadlineIsKept(t *testing.T) {

	spy := &spyServer{}
	c := dialSpy(t, Config{Token: "s3cret", Timeout: 10 * time.Second}, spy)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if _, err := c.rpc.ReleaseExpired(ctx, &catalogv1.ReleaseExpiredRequest{}); err != nil {
		t.Fatal(err)
	}

	if spy.remaining > time.Second {
		t.Fatalf("remaining = %v: the client replaced the caller's 1s deadline", spy.remaining)
	}
}

func TestASlowServerEndsWithDeadlineExceeded(t *testing.T) {

	spy := &spyServer{block: true}
	c := dialSpy(t, Config{Token: "s3cret", Timeout: 100 * time.Millisecond}, spy)

	// Run the call aside: if no deadline is ever applied it would wait for
	// ever, and the test must fail instead of hanging.
	done := make(chan error, 1)
	go func() {
		_, err := c.rpc.ReleaseExpired(context.Background(), &catalogv1.ReleaseExpiredRequest{})
		done <- err
	}()

	select {
	case err := <-done:
		if status.Code(err) != codes.DeadlineExceeded {
			t.Fatalf("code = %v, want DeadlineExceeded (err: %v)", status.Code(err), err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the call was still waiting after 3s: no deadline was applied")
	}
}
