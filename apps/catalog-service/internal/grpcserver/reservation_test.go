package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/fizzisme/catalog-service/internal/book"
	"github.com/fizzisme/catalog-service/internal/logger"
	catalogv1 "github.com/fizzisme/catalog-service/internal/pb/catalog/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// fakeStore stands in for the repository, so these tests need no database.
type fakeStore struct {
	held []book.Held
	err  error

	gotHolder string
	gotTTL    int
	gotSlugs  []string
	gotIDs    []int
	calls     int
}

func (f *fakeStore) Reserve(_ context.Context, holder string, ttl int, slugs []string) ([]book.Held, error) {
	f.calls++
	f.gotHolder, f.gotTTL, f.gotSlugs = holder, ttl, slugs
	return f.held, f.err
}

func (f *fakeStore) Release(_ context.Context, holder string, ids []int) (int, error) {
	f.calls++
	f.gotHolder, f.gotIDs = holder, ids
	return len(ids), f.err
}

func (f *fakeStore) Sell(_ context.Context, holder string, ids []int) error {
	f.calls++
	f.gotHolder, f.gotIDs = holder, ids
	return f.err
}

func (f *fakeStore) ReleaseExpired(context.Context) (int, error) {
	f.calls++
	return 3, f.err
}

// newClient runs the real server behind an in-memory connection, so a call
// goes through the same encoding, interceptors and status handling as on
// the network. serverOpts lets a test add interceptors.
func newClient(t *testing.T, store *fakeStore, serverOpts ...grpc.ServerOption) catalogv1.ReservationServiceClient {
	t.Helper()

	if err := logger.Init(); err != nil {
		t.Fatal(err)
	}

	lis := bufconn.Listen(1 << 20)

	srv := grpc.NewServer(serverOpts...)
	catalogv1.RegisterReservationServiceServer(srv, NewReservationServer(book.NewReservationService(store)))
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

	return catalogv1.NewReservationServiceClient(conn)
}

func TestValidateHolderAndIDs(t *testing.T) {

	ids := func(n int) []int32 {
		out := make([]int32, n)
		for i := range out {
			out[i] = int32(i + 1)
		}
		return out
	}

	tests := []struct {
		name   string
		holder string
		ids    []int32
		ok     bool
	}{
		{"valid", "BK-1", ids(2), true},
		{"holder at the limit", strings.Repeat("x", book.MaxHolderLen), ids(1), true},
		{"ids at the limit", "BK-1", ids(book.MaxBatch), true},
		{"empty holder", "", ids(1), false},
		{"holder too long", strings.Repeat("x", book.MaxHolderLen+1), ids(1), false},
		{"no ids", "BK-1", nil, false},
		{"too many ids", "BK-1", ids(book.MaxBatch + 1), false},
		{"zero id", "BK-1", []int32{1, 0}, false},
		{"negative id", "BK-1", []int32{-3}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := validateHolderAndIDs(tt.holder, tt.ids)

			if tt.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.ok && status.Code(err) != codes.InvalidArgument {
				t.Fatalf("code = %v, want InvalidArgument (err: %v)", status.Code(err), err)
			}
		})
	}
}

func TestValidateSlugs(t *testing.T) {

	slugs := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = fmt.Sprintf("book-%d", i)
		}
		return out
	}

	tests := []struct {
		name  string
		slugs []string
		ok    bool
	}{
		{"valid", slugs(2), true},
		{"at the batch limit", slugs(book.MaxBatch), true},
		{"a slug at the length limit", []string{strings.Repeat("a", book.MaxSlugLen)}, true},
		{"none", nil, false},
		{"too many", slugs(book.MaxBatch + 1), false},
		{"an empty slug", []string{"heidi-b1", ""}, false},
		{"a blank slug", []string{"   "}, false},
		{"a slug that is too long", []string{strings.Repeat("a", book.MaxSlugLen+1)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := validateSlugs(tt.slugs)

			if tt.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.ok && status.Code(err) != codes.InvalidArgument {
				t.Fatalf("code = %v, want InvalidArgument (err: %v)", status.Code(err), err)
			}
		})
	}
}

func TestReserveReturnsSnapshotAndExpiry(t *testing.T) {

	until := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	author := "Johanna Spyri"

	store := &fakeStore{held: []book.Held{
		{ID: 7, Slug: "heidi", Title: "Heidi", Author: &author, PriceVND: 150000, ReservedUntil: until},
		{ID: 9, Slug: "emma", Title: "Emma", PriceVND: 90000, ReservedUntil: until},
	}}
	client := newClient(t, store)

	res, err := client.Reserve(context.Background(), &catalogv1.ReserveRequest{
		Holder: "BK-1", TtlSeconds: 900, Slugs: []string{"Emma", "heidi", "HEIDI"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(res.GetItems()) != 2 {
		t.Fatalf("items = %d, want 2", len(res.GetItems()))
	}

	heidi, emma := res.GetItems()[0], res.GetItems()[1]
	if heidi.GetId() != 7 || heidi.GetTitle() != "Heidi" || heidi.GetPriceVnd() != 150000 {
		t.Errorf("first item = %v", heidi)
	}
	if heidi.Author == nil || *heidi.Author != author {
		t.Errorf("author = %v, want %q", heidi.Author, author)
	}
	if emma.Author != nil || emma.CoverUrl != nil {
		t.Errorf("a missing author or cover must stay unset, got %v / %v", emma.Author, emma.CoverUrl)
	}
	if !res.GetReservedUntil().AsTime().Equal(until) {
		t.Errorf("reservedUntil = %v, want %v", res.GetReservedUntil().AsTime(), until)
	}

	// The service lower-cases, deduplicates and sorts before the store.
	if store.gotHolder != "BK-1" || store.gotTTL != 900 || fmt.Sprint(store.gotSlugs) != "[emma heidi]" {
		t.Errorf("store got %q %d %v", store.gotHolder, store.gotTTL, store.gotSlugs)
	}
	if heidi.GetSlug() != "heidi" || emma.GetSlug() != "emma" {
		t.Errorf("slugs = %q / %q", heidi.GetSlug(), emma.GetSlug())
	}
}

func TestUnavailableBooksAreFailedPreconditionWithDetails(t *testing.T) {

	// A slug that does not exist has no id; an id that does not exist has no slug.
	conflict := &book.UnavailableError{Books: []book.Unavailable{
		{ID: 3, Slug: "heidi-b3", Status: book.StatusSold},
		{ID: 5, Slug: "kim-b5", Status: book.StatusReserved},
		{Slug: "no-such-book", Status: book.StatusMissing},
	}}

	// Reserve and Sell can both fail this way; wrapping must not hide it.
	client := newClient(t, &fakeStore{err: fmt.Errorf("reserve: %w", conflict)})
	ctx := context.Background()

	calls := map[string]func() error{
		"Reserve": func() error {
			_, err := client.Reserve(ctx, &catalogv1.ReserveRequest{Holder: "BK-1", TtlSeconds: 60, Slugs: []string{"heidi-b3", "kim-b5", "no-such-book"}})
			return err
		},
		"Sell": func() error {
			_, err := client.Sell(ctx, &catalogv1.SellRequest{Holder: "BK-1", Ids: []int32{3, 5, 8}})
			return err
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {

			st := status.Convert(call())
			if st.Code() != codes.FailedPrecondition {
				t.Fatalf("code = %v, want FailedPrecondition", st.Code())
			}

			var got *catalogv1.UnavailableBooks
			for _, d := range st.Details() {
				if ub, ok := d.(*catalogv1.UnavailableBooks); ok {
					got = ub
				}
			}
			if got == nil || len(got.GetBooks()) != 3 {
				t.Fatalf("details = %v, want 3 unavailable books", st.Details())
			}

			want := []struct {
				id     int32
				slug   string
				status catalogv1.BookStatus
			}{
				{3, "heidi-b3", catalogv1.BookStatus_BOOK_STATUS_SOLD},
				{5, "kim-b5", catalogv1.BookStatus_BOOK_STATUS_RESERVED},
				{0, "no-such-book", catalogv1.BookStatus_BOOK_STATUS_MISSING},
			}
			for i, w := range want {
				b := got.Books[i]
				if b.GetId() != w.id || b.GetSlug() != w.slug || b.GetStatus() != w.status {
					t.Errorf("book %d = %v, want id %d slug %q status %v", i, b, w.id, w.slug, w.status)
				}
			}
		})
	}
}

func TestUnexpectedErrorsDoNotLeak(t *testing.T) {

	client := newClient(t, &fakeStore{err: errors.New("mssql: Invalid object name 'books' (password=hunter2)")})
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
		"ReleaseExpired": func() error {
			_, err := client.ReleaseExpired(ctx, &catalogv1.ReleaseExpiredRequest{})
			return err
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {

			st := status.Convert(call())

			if st.Code() != codes.Internal {
				t.Fatalf("code = %v, want Internal", st.Code())
			}
			if strings.Contains(st.Message(), "hunter2") || strings.Contains(st.Message(), "mssql") {
				t.Fatalf("the message leaks the underlying error: %q", st.Message())
			}
		})
	}
}

func TestContextErrorsKeepTheirMeaning(t *testing.T) {

	if err := logger.Init(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"canceled", fmt.Errorf("query: %w", context.Canceled), codes.Canceled},
		{"deadline", fmt.Errorf("query: %w", context.DeadlineExceeded), codes.DeadlineExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := status.Code(toStatusError("Reserve", tt.err)); got != tt.want {
				t.Fatalf("code = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvalidRequestsNeverReachTheStore(t *testing.T) {

	store := &fakeStore{}
	client := newClient(t, store)
	ctx := context.Background()

	_, err := client.Reserve(ctx, &catalogv1.ReserveRequest{Holder: "BK-1", TtlSeconds: book.MaxHoldSeconds + 1, Slugs: []string{"heidi-b1"}})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("ttl above the limit: code = %v, want InvalidArgument", status.Code(err))
	}

	_, err = client.Reserve(ctx, &catalogv1.ReserveRequest{Holder: "BK-1", TtlSeconds: 0, Slugs: []string{"heidi-b1"}})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("zero ttl: code = %v, want InvalidArgument", status.Code(err))
	}

	_, err = client.Reserve(ctx, &catalogv1.ReserveRequest{Holder: "BK-1", TtlSeconds: 60})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("reserve without slugs: code = %v, want InvalidArgument", status.Code(err))
	}

	_, err = client.Reserve(ctx, &catalogv1.ReserveRequest{Holder: "BK-1", TtlSeconds: 60, Slugs: []string{"heidi-b1", " "}})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("reserve with a blank slug: code = %v, want InvalidArgument", status.Code(err))
	}

	_, err = client.Reserve(ctx, &catalogv1.ReserveRequest{Holder: "", TtlSeconds: 60, Slugs: []string{"heidi-b1"}})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("reserve without holder: code = %v, want InvalidArgument", status.Code(err))
	}

	_, err = client.Release(ctx, &catalogv1.ReleaseRequest{Holder: "BK-1"})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("release without ids: code = %v, want InvalidArgument", status.Code(err))
	}

	_, err = client.Sell(ctx, &catalogv1.SellRequest{Holder: "", Ids: []int32{1}})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("sell without holder: code = %v, want InvalidArgument", status.Code(err))
	}

	if store.calls != 0 {
		t.Errorf("the store was called %d times for invalid requests", store.calls)
	}
}
