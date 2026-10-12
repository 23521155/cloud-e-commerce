package order

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fizzisme/order-service/internal/catalogclient"
	"github.com/fizzisme/order-service/internal/payment/vnpay"
	"go.uber.org/zap"
)

// validInput is a checkout that passes normalize; tests change one field.
func validInput() CreateInput {
	return CreateInput{
		Key:           "3f9c1a52-7b0e-4d4a-9c1e-8a2b5d6f7e10",
		Slugs:         []string{"nha-gia-kim", "dac-nhan-tam"},
		PaymentMethod: PaymentCOD,
		Email:         "guest@example.com",
		Phone:         "0901234567",
		FullName:      "Nguyễn Văn A",
		Address:       "12 Lê Lợi",
		City:          "Hồ Chí Minh",
		Country:       "Vietnam",
	}
}

func TestNormalizeAcceptsAValidInput(t *testing.T) {

	in := validInput()
	in.FullName = "  Nguyễn Văn A  "
	blank := "   "
	in.PostalCode = &blank

	got, err := in.normalize()
	if err != nil {
		t.Fatal(err)
	}

	if got.FullName != "Nguyễn Văn A" {
		t.Errorf("FullName = %q, want it trimmed", got.FullName)
	}
	if got.PostalCode != nil {
		t.Errorf("PostalCode = %q, want nil for a blank one", *got.PostalCode)
	}
}

func TestNormalizeRejects(t *testing.T) {

	tests := []struct {
		name   string
		change func(in *CreateInput)
	}{
		{"key is not a UUID", func(in *CreateInput) { in.Key = "abc" }},
		{"no key", func(in *CreateInput) { in.Key = "" }},
		{"online payment", func(in *CreateInput) { in.PaymentMethod = "momo" }},
		{"no books", func(in *CreateInput) { in.Slugs = nil }},
		{"too many books", func(in *CreateInput) { in.Slugs = make([]string, maxItems+1) }},
		{"blank slug", func(in *CreateInput) { in.Slugs = []string{" "} }},
		{"long slug", func(in *CreateInput) { in.Slugs = []string{strings.Repeat("a", maxSlugLen+1)} }},
		{"same book twice", func(in *CreateInput) { in.Slugs = []string{"a", "a"} }},
		{"no email", func(in *CreateInput) { in.Email = " " }},
		{"email with a name", func(in *CreateInput) { in.Email = "Guest <guest@example.com>" }},
		{"email without a domain", func(in *CreateInput) { in.Email = "guest" }},
		{"no phone", func(in *CreateInput) { in.Phone = "" }},
		{"long phone", func(in *CreateInput) { in.Phone = strings.Repeat("1", maxPhoneLen+1) }},
		{"no full name", func(in *CreateInput) { in.FullName = "" }},
		{"no address", func(in *CreateInput) { in.Address = "" }},
		{"no city", func(in *CreateInput) { in.City = "" }},
		{"no country", func(in *CreateInput) { in.Country = "" }},
		{"long postal code", func(in *CreateInput) {
			long := strings.Repeat("1", maxPostalLen+1)
			in.PostalCode = &long
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			in := validInput()
			tt.change(&in)

			if _, err := in.normalize(); !errors.Is(err, ErrInvalidInput) {
				t.Errorf("err = %v, want ErrInvalidInput", err)
			}
		})
	}
}

// fakeStore is an in-memory Store. Fields set before the call make a step
// fail; the rest records what the service did.
type fakeStore struct {
	orders map[string]*Order // by idempotency key

	takenCodes map[string]bool

	codeExistsErr error
	insertErr     error
	cancelErr     error
	markPaidErr   error

	listExpiredErr error

	inserted  []*Order
	cancelled map[int64]string // id -> reason
	paid      map[int64]string // id -> payment ref
	nextID    int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		orders:     map[string]*Order{},
		takenCodes: map[string]bool{},
		cancelled:  map[int64]string{},
		paid:       map[int64]string{},
	}
}

func (f *fakeStore) CodeExists(ctx context.Context, code string) (bool, error) {
	return f.takenCodes[code], f.codeExistsErr
}

func (f *fakeStore) FindByIdempotencyKey(ctx context.Context, key string) (Order, error) {
	if o, ok := f.orders[key]; ok {
		return *o, nil
	}
	return Order{}, ErrNotFound
}

func (f *fakeStore) Insert(ctx context.Context, key string, o *Order) error {
	if f.insertErr != nil {
		return f.insertErr
	}

	f.nextID++
	o.ID = f.nextID
	o.CreatedAt = time.Now()

	stored := *o
	f.orders[key] = &stored
	f.takenCodes[o.Code] = true
	f.inserted = append(f.inserted, &stored)

	return nil
}

func (f *fakeStore) Cancel(ctx context.Context, id int64, reason string) error {
	f.cancelled[id] = reason
	return f.cancelErr
}

func (f *fakeStore) FindByCode(ctx context.Context, code string) (Order, error) {
	for _, o := range f.orders {
		if o.Code == code {
			return *o, nil
		}
	}
	return Order{}, ErrNotFound
}

// byID is the stored order with id, or nil.
func (f *fakeStore) byID(id int64) *Order {
	for _, o := range f.orders {
		if o.ID == id {
			return o
		}
	}
	return nil
}

// MarkPaid and CancelAwaiting change only an AWAITING_PAYMENT order, as the
// real UPDATEs do.
func (f *fakeStore) MarkPaid(ctx context.Context, id int64, ref string) error {
	if f.markPaidErr != nil {
		return f.markPaidErr
	}

	o := f.byID(id)
	if o == nil || o.Status != StatusAwaitingPayment {
		return ErrNotAwaitingPayment
	}

	now := time.Now()
	o.Status, o.PaymentStatus, o.PaymentRef, o.PaidAt = StatusConfirmed, PaymentPaid, &ref, &now
	f.paid[id] = ref

	return nil
}

func (f *fakeStore) CancelAwaiting(ctx context.Context, id int64, reason string) error {
	o := f.byID(id)
	if o == nil || o.Status != StatusAwaitingPayment {
		return ErrNotAwaitingPayment
	}

	o.Status = StatusCancelled
	f.cancelled[id] = reason

	return nil
}

// ListExpiredAwaiting returns the AWAITING_PAYMENT orders whose hold has
// ended, as the real query does.
func (f *fakeStore) ListExpiredAwaiting(ctx context.Context, limit int) ([]Order, error) {
	if f.listExpiredErr != nil {
		return nil, f.listExpiredErr
	}

	var expired []Order
	for _, o := range f.orders {
		if o.Status == StatusAwaitingPayment && o.ExpiresAt != nil && o.ExpiresAt.Before(time.Now()) && len(expired) < limit {
			expired = append(expired, *o)
		}
	}
	return expired, nil
}

// fakePayments signs nothing: the URL only names the order, and every
// request is recorded.
type fakePayments struct {
	err      error
	requests []vnpay.PayRequest
}

func (f *fakePayments) BuildPayURL(r vnpay.PayRequest) (string, error) {
	f.requests = append(f.requests, r)
	if f.err != nil {
		return "", f.err
	}
	return "https://pay.example/?ref=" + r.TxnRef, nil
}

// fakeCatalog is a Reserver that answers from fields set by the test and
// records every call.
type fakeCatalog struct {
	books []catalogclient.Book // what Reserve holds, in order

	reserveErr error
	releaseErr error
	sellErr    error

	reserved []reserveCall
	released []holdCall
	sold     []holdCall
}

type reserveCall struct {
	Holder string
	TTL    time.Duration
	Slugs  []string
}

type holdCall struct {
	Holder string
	IDs    []int
}

func (f *fakeCatalog) Reserve(ctx context.Context, holder string, ttl time.Duration, slugs []string) (catalogclient.Reservation, error) {
	f.reserved = append(f.reserved, reserveCall{holder, ttl, slugs})
	if f.reserveErr != nil {
		return catalogclient.Reservation{}, f.reserveErr
	}

	return catalogclient.Reservation{
		Items:         f.books,
		ReservedUntil: time.Now().Add(ttl),
	}, nil
}

func (f *fakeCatalog) Release(ctx context.Context, holder string, ids []int) (int, error) {
	f.released = append(f.released, holdCall{holder, ids})
	return len(ids), f.releaseErr
}

func (f *fakeCatalog) Sell(ctx context.Context, holder string, ids []int) error {
	f.sold = append(f.sold, holdCall{holder, ids})
	return f.sellErr
}

// testHold is the payment hold of the test service.
const testHold = 15 * time.Minute

func newTestService(store Store, catalog *fakeCatalog) *Service {
	return NewService(store, catalog, &fakePayments{}, testHold, zap.NewNop())
}

func TestCreateCOD(t *testing.T) {

	store, catalog := newFakeStore(), &fakeCatalog{books: twoBooks()}
	svc := newTestService(store, catalog)

	got, replayed, err := svc.Create(context.Background(), validInput())
	if err != nil {
		t.Fatal(err)
	}

	if replayed {
		t.Error("replayed = true for a new order")
	}
	if got.Status != StatusConfirmed || got.PaymentStatus != PaymentUnpaid || got.UserID != nil {
		t.Errorf("order = %s / %s / user %v, want a CONFIRMED UNPAID guest order", got.Status, got.PaymentStatus, got.UserID)
	}
	if got.SubtotalVND != 200000 || len(got.Items) != 2 || got.Items[0].BookID != 1 {
		t.Errorf("subtotal %d, items %+v; want 200000 and the reserved books", got.SubtotalVND, got.Items)
	}

	if len(catalog.reserved) != 1 || catalog.reserved[0].Holder != got.Code ||
		catalog.reserved[0].TTL != codReserveTTL ||
		!slices.Equal(catalog.reserved[0].Slugs, []string{"dac-nhan-tam", "nha-gia-kim"}) {
		t.Errorf("reserve calls = %+v, want one for %s with sorted slugs", catalog.reserved, got.Code)
	}
	if len(catalog.sold) != 1 || catalog.sold[0].Holder != got.Code || !slices.Equal(catalog.sold[0].IDs, []int{1, 2}) {
		t.Errorf("sell calls = %+v, want one for %s", catalog.sold, got.Code)
	}
	if len(catalog.released) != 0 || len(store.cancelled) != 0 {
		t.Errorf("released %+v, cancelled %v; want neither", catalog.released, store.cancelled)
	}
}

func TestCreateRejectsBeforeTouchingTheCatalogue(t *testing.T) {

	store, catalog := newFakeStore(), &fakeCatalog{books: twoBooks()}
	in := validInput()
	in.PaymentMethod = "momo"

	if _, _, err := newTestService(store, catalog).Create(context.Background(), in); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if len(catalog.reserved) != 0 {
		t.Error("books were reserved for an invalid order")
	}
}

func TestCreateReplaysTheSameKey(t *testing.T) {

	store, catalog := newFakeStore(), &fakeCatalog{books: twoBooks()}
	svc := newTestService(store, catalog)

	first, _, err := svc.Create(context.Background(), validInput())
	if err != nil {
		t.Fatal(err)
	}

	again, replayed, err := svc.Create(context.Background(), validInput())
	if err != nil {
		t.Fatal(err)
	}

	if !replayed || again.Code != first.Code {
		t.Errorf("replayed = %v, code %s; want the first order %s", replayed, again.Code, first.Code)
	}
	if len(catalog.reserved) != 1 || len(store.inserted) != 1 {
		t.Errorf("%d reserves, %d inserts; a replay must not repeat them", len(catalog.reserved), len(store.inserted))
	}
}

func TestCreateKeyOfAnotherCustomer(t *testing.T) {

	store, catalog := newFakeStore(), &fakeCatalog{books: twoBooks()}
	svc := newTestService(store, catalog)

	if _, _, err := svc.Create(context.Background(), validInput()); err != nil {
		t.Fatal(err)
	}

	other := validInput()
	other.Email = "someone-else@example.com"

	if _, _, err := svc.Create(context.Background(), other); !errors.Is(err, ErrKeyReused) {
		t.Errorf("err = %v, want ErrKeyReused", err)
	}
}

func TestCreateBooksUnavailable(t *testing.T) {

	store := newFakeStore()
	catalog := &fakeCatalog{reserveErr: &catalogclient.UnavailableError{
		Books: []catalogclient.Unavailable{{Slug: "nha-gia-kim", Status: catalogclient.StatusSold}},
	}}

	_, _, err := newTestService(store, catalog).Create(context.Background(), validInput())

	var unavailable *catalogclient.UnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("err = %v, want *UnavailableError", err)
	}
	if len(store.inserted) != 0 || len(catalog.sold) != 0 {
		t.Error("an order was written although the books were not held")
	}
}

func TestCreateRetriesReserveOnce(t *testing.T) {

	store := newFakeStore()
	catalog := &fakeCatalog{reserveErr: catalogclient.ErrCatalogUnavailable}

	_, _, err := newTestService(store, catalog).Create(context.Background(), validInput())

	if !errors.Is(err, catalogclient.ErrCatalogUnavailable) {
		t.Fatalf("err = %v, want ErrCatalogUnavailable", err)
	}
	if len(catalog.reserved) != 2 || catalog.reserved[0].Holder != catalog.reserved[1].Holder {
		t.Errorf("reserve calls = %+v, want two with the same holder", catalog.reserved)
	}
	if len(store.inserted) != 0 {
		t.Error("an order was written without held books")
	}
}

func TestCreateInsertFailureReleasesTheBooks(t *testing.T) {

	store := newFakeStore()
	store.insertErr = errors.New("connection lost")
	catalog := &fakeCatalog{books: twoBooks()}

	if _, _, err := newTestService(store, catalog).Create(context.Background(), validInput()); err == nil {
		t.Fatal("err = nil, want the insert error")
	}

	if len(catalog.released) != 1 || !slices.Equal(catalog.released[0].IDs, []int{1, 2}) {
		t.Errorf("release calls = %+v, want one for both books", catalog.released)
	}
	if len(catalog.sold) != 0 {
		t.Error("books were sold for an order that was not written")
	}
}

func TestCreateSellFailureCancelsAndReleases(t *testing.T) {

	store := newFakeStore()
	catalog := &fakeCatalog{
		books: twoBooks(),
		sellErr: &catalogclient.UnavailableError{
			Books: []catalogclient.Unavailable{{ID: 1, Status: catalogclient.StatusReserved}},
		},
	}

	_, _, err := newTestService(store, catalog).Create(context.Background(), validInput())

	var unavailable *catalogclient.UnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("err = %v, want *UnavailableError", err)
	}
	if len(store.cancelled) != 1 || len(catalog.released) != 1 {
		t.Errorf("cancelled %v, released %+v; want one of each", store.cancelled, catalog.released)
	}
}

func TestCreateRetriesSellOnce(t *testing.T) {

	store := newFakeStore()
	catalog := &fakeCatalog{books: twoBooks(), sellErr: catalogclient.ErrCatalogUnavailable}

	_, _, err := newTestService(store, catalog).Create(context.Background(), validInput())

	if !errors.Is(err, catalogclient.ErrCatalogUnavailable) {
		t.Fatalf("err = %v, want ErrCatalogUnavailable", err)
	}
	if len(catalog.sold) != 2 {
		t.Errorf("sell calls = %d, want 2", len(catalog.sold))
	}
	if len(store.cancelled) != 1 || len(catalog.released) != 1 {
		t.Errorf("cancelled %v, released %+v; want one of each", store.cancelled, catalog.released)
	}
}

func TestCreateParallelRequestWithTheSameKey(t *testing.T) {

	store, catalog := newFakeStore(), &fakeCatalog{books: twoBooks()}

	// The other request writes its order between our key lookup and our insert.
	winner := Order{ID: 99, Code: "MA-111111", Email: "guest@example.com"}
	store.insertErr = ErrDuplicateKey
	store.orders[validInput().Key] = &winner

	// FindByIdempotencyKey must miss the first time only.
	svc := newTestService(&lateKeyStore{fakeStore: store}, catalog)

	got, replayed, err := svc.Create(context.Background(), validInput())
	if err != nil {
		t.Fatal(err)
	}

	if !replayed || got.Code != "MA-111111" {
		t.Errorf("replayed = %v, code %s; want the winner's order", replayed, got.Code)
	}
	if len(catalog.released) != 1 || len(catalog.sold) != 0 {
		t.Errorf("released %+v, sold %+v; want our books released and none sold", catalog.released, catalog.sold)
	}
}

// lateKeyStore hides the order from the first FindByIdempotencyKey call, as
// if a parallel request wrote it right after.
type lateKeyStore struct {
	*fakeStore
	looked bool
}

func (s *lateKeyStore) FindByIdempotencyKey(ctx context.Context, key string) (Order, error) {
	if !s.looked {
		s.looked = true
		return Order{}, ErrNotFound
	}
	return s.fakeStore.FindByIdempotencyKey(ctx, key)
}

func TestCreateNoFreeCode(t *testing.T) {

	catalog := &fakeCatalog{books: twoBooks()}

	// Every code looks taken.
	svc := newTestService(&allTakenStore{newFakeStore()}, catalog)

	if _, _, err := svc.Create(context.Background(), validInput()); !errors.Is(err, ErrNoFreeCode) {
		t.Fatalf("err = %v, want ErrNoFreeCode", err)
	}
	if len(catalog.reserved) != 0 {
		t.Error("books were reserved without a free code")
	}
}

type allTakenStore struct{ *fakeStore }

func (s *allTakenStore) CodeExists(ctx context.Context, code string) (bool, error) {
	return true, nil
}

// twoBooks is what a catalogue holds for validInput.
func twoBooks() []catalogclient.Book {
	return []catalogclient.Book{
		{ID: 1, Slug: "nha-gia-kim", Title: "Nhà giả kim", PriceVND: 120000},
		{ID: 2, Slug: "dac-nhan-tam", Title: "Đắc nhân tâm", PriceVND: 80000},
	}
}
