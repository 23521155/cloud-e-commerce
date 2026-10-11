package catalogclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	catalogv1 "github.com/fizzisme/order-service/internal/pb/catalog/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// fakeCatalog is a stand-in catalogue whose answers each test decides. An
// RPC a test did not set up is answered as unimplemented.
type fakeCatalog struct {
	catalogv1.UnimplementedReservationServiceServer

	reserve func(context.Context, *catalogv1.ReserveRequest) (*catalogv1.ReserveResponse, error)
	release func(context.Context, *catalogv1.ReleaseRequest) (*catalogv1.ReleaseResponse, error)
	sell    func(context.Context, *catalogv1.SellRequest) (*catalogv1.SellResponse, error)

	calls atomic.Int32
}

func (f *fakeCatalog) Reserve(ctx context.Context, req *catalogv1.ReserveRequest) (*catalogv1.ReserveResponse, error) {
	f.calls.Add(1)
	if f.reserve == nil {
		return f.UnimplementedReservationServiceServer.Reserve(ctx, req)
	}
	return f.reserve(ctx, req)
}

func (f *fakeCatalog) Release(ctx context.Context, req *catalogv1.ReleaseRequest) (*catalogv1.ReleaseResponse, error) {
	f.calls.Add(1)
	if f.release == nil {
		return f.UnimplementedReservationServiceServer.Release(ctx, req)
	}
	return f.release(ctx, req)
}

func (f *fakeCatalog) Sell(ctx context.Context, req *catalogv1.SellRequest) (*catalogv1.SellResponse, error) {
	f.calls.Add(1)
	if f.sell == nil {
		return f.UnimplementedReservationServiceServer.Sell(ctx, req)
	}
	return f.sell(ctx, req)
}

// failing answers every RPC with err, so one table can drive all three methods.
func failing(err error) *fakeCatalog {
	return &fakeCatalog{
		reserve: func(context.Context, *catalogv1.ReserveRequest) (*catalogv1.ReserveResponse, error) {
			return nil, err
		},
		release: func(context.Context, *catalogv1.ReleaseRequest) (*catalogv1.ReleaseResponse, error) {
			return nil, err
		},
		sell: func(context.Context, *catalogv1.SellRequest) (*catalogv1.SellResponse, error) {
			return nil, err
		},
	}
}

// callAll runs the three methods against the same catalogue.
func callAll(c *Client) map[string]error {

	ctx := context.Background()
	out := map[string]error{}

	_, out["Reserve"] = c.Reserve(ctx, "BK-1", time.Minute, []int{1})
	_, out["Release"] = c.Release(ctx, "BK-1", []int{1})
	out["Sell"] = c.Sell(ctx, "BK-1", []int{1})

	return out
}

// unavailableStatus is what the catalogue answers when books are not free:
// FAILED_PRECONDITION with the books attached as details.
func unavailableStatus(t *testing.T, books ...*catalogv1.Unavailable) error {
	t.Helper()

	st, err := status.New(codes.FailedPrecondition, "books unavailable").
		WithDetails(&catalogv1.UnavailableBooks{Books: books})
	if err != nil {
		t.Fatal(err)
	}

	return st.Err()
}

func TestReserveSendsTheRequestAndMapsTheAnswer(t *testing.T) {

	until := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	author := "Johanna Spyri"

	var got *catalogv1.ReserveRequest
	fake := &fakeCatalog{reserve: func(_ context.Context, req *catalogv1.ReserveRequest) (*catalogv1.ReserveResponse, error) {
		got = req
		return &catalogv1.ReserveResponse{
			Items: []*catalogv1.Held{
				{Id: 7, Slug: "heidi", Title: "Heidi", Author: &author, PriceVnd: 150000},
				{Id: 9, Slug: "emma", Title: "Emma", PriceVnd: 90000},
			},
			ReservedUntil: timestamppb.New(until),
		}, nil
	}}
	c := dialSpy(t, Config{Token: "s3cret"}, fake)

	res, err := c.Reserve(context.Background(), "BK-1", 15*time.Minute, []int{7, 9})
	if err != nil {
		t.Fatal(err)
	}

	if got.GetHolder() != "BK-1" || got.GetTtlSeconds() != 900 || fmt.Sprint(got.GetIds()) != "[7 9]" {
		t.Errorf("the catalogue received %v", got)
	}

	if len(res.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(res.Items))
	}
	heidi, emma := res.Items[0], res.Items[1]
	if heidi.ID != 7 || heidi.Slug != "heidi" || heidi.Title != "Heidi" || heidi.PriceVND != 150000 {
		t.Errorf("first book = %+v", heidi)
	}
	if heidi.Author == nil || *heidi.Author != author {
		t.Errorf("author = %v, want %q", heidi.Author, author)
	}
	if emma.Author != nil || emma.CoverURL != nil {
		t.Errorf("a missing author or cover must stay nil, got %v / %v", emma.Author, emma.CoverURL)
	}
	if !res.ReservedUntil.Equal(until) {
		t.Errorf("ReservedUntil = %v, want %v", res.ReservedUntil, until)
	}
}

func TestReserveRefusesAHoldUnderASecondWithoutCalling(t *testing.T) {

	fake := &fakeCatalog{}
	c := dialSpy(t, Config{Token: "s3cret"}, fake)

	_, err := c.Reserve(context.Background(), "BK-1", 500*time.Millisecond, []int{1})

	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
	if fake.calls.Load() != 0 {
		t.Fatalf("the catalogue was called %d times", fake.calls.Load())
	}
}

func TestReleaseAndSellSendTheirRequests(t *testing.T) {

	var released *catalogv1.ReleaseRequest
	var sold *catalogv1.SellRequest

	fake := &fakeCatalog{
		release: func(_ context.Context, req *catalogv1.ReleaseRequest) (*catalogv1.ReleaseResponse, error) {
			released = req
			return &catalogv1.ReleaseResponse{Released: 2}, nil
		},
		sell: func(_ context.Context, req *catalogv1.SellRequest) (*catalogv1.SellResponse, error) {
			sold = req
			return &catalogv1.SellResponse{}, nil
		},
	}
	c := dialSpy(t, Config{Token: "s3cret"}, fake)
	ctx := context.Background()

	n, err := c.Release(ctx, "BK-1", []int{4, 5})
	if err != nil || n != 2 {
		t.Fatalf("Release = %d, %v; want 2, nil", n, err)
	}
	if released.GetHolder() != "BK-1" || fmt.Sprint(released.GetIds()) != "[4 5]" {
		t.Errorf("Release request = %v", released)
	}

	if err := c.Sell(ctx, "BK-2", []int{6}); err != nil {
		t.Fatalf("Sell: %v", err)
	}
	if sold.GetHolder() != "BK-2" || fmt.Sprint(sold.GetIds()) != "[6]" {
		t.Errorf("Sell request = %v", sold)
	}
}

func TestUnavailableBooksBecomeAnUnavailableError(t *testing.T) {

	c := dialSpy(t, Config{Token: "s3cret"}, failing(unavailableStatus(t,
		&catalogv1.Unavailable{Id: 3, Status: catalogv1.BookStatus_BOOK_STATUS_SOLD},
		&catalogv1.Unavailable{Id: 5, Status: catalogv1.BookStatus_BOOK_STATUS_RESERVED},
		&catalogv1.Unavailable{Id: 8, Status: catalogv1.BookStatus_BOOK_STATUS_MISSING},
		&catalogv1.Unavailable{Id: 9, Status: catalogv1.BookStatus_BOOK_STATUS_UNSPECIFIED},
	)))

	// Only Reserve and Sell can run into this, but the translation is shared.
	for name, err := range callAll(c) {
		t.Run(name, func(t *testing.T) {

			// A caller will wrap the error before it travels on.
			var got *UnavailableError
			if !errors.As(fmt.Errorf("create order: %w", err), &got) {
				t.Fatalf("err = %v, want an *UnavailableError", err)
			}

			want := []Unavailable{
				{3, StatusSold}, {5, StatusReserved}, {8, StatusMissing}, {9, StatusUnknown},
			}
			if fmt.Sprint(got.Books) != fmt.Sprint(want) {
				t.Fatalf("books = %v, want %v", got.Books, want)
			}
			if errors.Is(err, ErrCatalogUnavailable) {
				t.Fatal("a sold-out book must not look like a catalogue failure")
			}
		})
	}
}

// The catalogue falls back to a bare FAILED_PRECONDITION when it cannot
// attach the details. It is still "the books were not free".
func TestAFailedPreconditionWithoutDetailsIsStillUnavailable(t *testing.T) {

	c := dialSpy(t, Config{Token: "s3cret"}, failing(status.Error(codes.FailedPrecondition, "books unavailable")))

	_, err := c.Reserve(context.Background(), "BK-1", time.Minute, []int{1})

	var got *UnavailableError
	if !errors.As(err, &got) {
		t.Fatalf("err = %v, want an *UnavailableError", err)
	}
	if len(got.Books) != 0 {
		t.Fatalf("books = %v, want none", got.Books)
	}
}

func TestStatusCodesAreTranslated(t *testing.T) {

	tests := []struct {
		code codes.Code
		want error
	}{
		{codes.Unauthenticated, ErrUnauthenticated},
		{codes.InvalidArgument, ErrInvalidRequest},
		{codes.Internal, ErrCatalogUnavailable},
		{codes.Unavailable, ErrCatalogUnavailable},
		{codes.DeadlineExceeded, ErrCatalogUnavailable},
		{codes.ResourceExhausted, ErrCatalogUnavailable},
		{codes.Unknown, ErrCatalogUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {

			c := dialSpy(t, Config{Token: "s3cret"}, failing(status.Error(tt.code, "boom")))

			for name, err := range callAll(c) {
				if !errors.Is(err, tt.want) {
					t.Errorf("%s: err = %v, want it to be %v", name, err, tt.want)
				}
				if tt.want != ErrCatalogUnavailable && errors.Is(err, ErrCatalogUnavailable) {
					t.Errorf("%s: %v must not be reported as a catalogue failure", name, tt.code)
				}
			}
		})
	}
}

func TestTheOriginalErrorStaysInTheChain(t *testing.T) {

	c := dialSpy(t, Config{Token: "s3cret"}, failing(status.Error(codes.Internal, "boom: table books is locked")))

	_, err := c.Reserve(context.Background(), "BK-1", time.Minute, []int{1})

	if !errors.Is(err, ErrCatalogUnavailable) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "boom: table books is locked") {
		t.Fatalf("the cause is lost: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "code = Internal") {
		t.Fatalf("the status code is lost: %q", err.Error())
	}
}

func TestASlowCatalogIsUnavailable(t *testing.T) {

	fake := failing(nil)
	fake.reserve = func(ctx context.Context, _ *catalogv1.ReserveRequest) (*catalogv1.ReserveResponse, error) {
		<-ctx.Done()
		return nil, status.FromContextError(ctx.Err()).Err()
	}
	c := dialSpy(t, Config{Token: "s3cret", Timeout: 100 * time.Millisecond}, fake)

	// Run aside so a missing deadline fails the test instead of hanging it.
	done := make(chan error, 1)
	go func() {
		_, err := c.Reserve(context.Background(), "BK-1", time.Minute, []int{1})
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, ErrCatalogUnavailable) {
			t.Fatalf("err = %v, want ErrCatalogUnavailable", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("still waiting after 3s")
	}
}

func TestACatalogThatIsNotRunningIsUnavailable(t *testing.T) {

	c, err := New(Config{Addr: "127.0.0.1:1", Token: "s3cret", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })

	for name, err := range callAll(c) {
		if !errors.Is(err, ErrCatalogUnavailable) {
			t.Errorf("%s: err = %v, want ErrCatalogUnavailable", name, err)
		}
	}
}

// A caller that gives up is not a catalogue failure, and must not be treated
// as one (it would trigger the clean-up for an outcome that is not unknown).
func TestACallerThatGivesUpIsNotACatalogueFailure(t *testing.T) {

	fake := failing(nil)
	fake.reserve = func(ctx context.Context, _ *catalogv1.ReserveRequest) (*catalogv1.ReserveResponse, error) {
		<-ctx.Done()
		return nil, status.FromContextError(ctx.Err()).Err()
	}
	c := dialSpy(t, Config{Token: "s3cret"}, fake)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Reserve(ctx, "BK-1", time.Minute, []int{1})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if errors.Is(err, ErrCatalogUnavailable) {
		t.Fatal("a caller's own cancel was reported as a catalogue failure")
	}
}

func TestNoErrorIsAReallyNilError(t *testing.T) {

	fake := &fakeCatalog{
		sell: func(context.Context, *catalogv1.SellRequest) (*catalogv1.SellResponse, error) {
			return &catalogv1.SellResponse{}, nil
		},
	}
	c := dialSpy(t, Config{Token: "s3cret"}, fake)

	// A nil *UnavailableError stored in an error is not nil: Sell must
	// return the nil error itself on success.
	if err := c.Sell(context.Background(), "BK-1", []int{1}); err != nil {
		t.Fatalf("Sell returned %#v, want a nil error", err)
	}
}
