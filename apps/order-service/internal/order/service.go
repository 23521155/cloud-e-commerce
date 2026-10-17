package order

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fizzisme/order-service/internal/catalogclient"
	"github.com/fizzisme/order-service/internal/payment/vnpay"
	"go.uber.org/zap"
)

var (
	// ErrInvalidInput means the request is malformed; the message says which
	// field and why, and is safe to show to the customer.
	ErrInvalidInput = errors.New("invalid order")

	// ErrKeyReused means the idempotency key already belongs to an order
	// placed by someone else (another email).
	ErrKeyReused = errors.New("idempotency key belongs to another order")

	// ErrNoFreeCode means no unused order code was found after several tries.
	ErrNoFreeCode = errors.New("no free order code")

	// ErrAmountMismatch means the gateway reports another amount than the
	// order's: the payment is not accepted.
	ErrAmountMismatch = errors.New("paid amount does not match the order")

	// ErrAlreadyProcessed means the payment of this order was already
	// handled, by an earlier call or by a parallel one.
	ErrAlreadyProcessed = errors.New("payment already processed")

	// ErrPaidTooLate means the customer paid but the books could not be
	// sold any more: the order is cancelled and must be refunded by hand.
	ErrPaidTooLate = errors.New("paid after the books were released")
)

// Limits of one order. The text limits match the orders columns.
const (
	maxItems      = 20
	maxSlugLen    = 100
	maxEmailLen   = 254
	maxPhoneLen   = 20
	maxNameLen    = 150
	maxAddressLen = 300
	maxCityLen    = 100
	maxPostalLen  = 20
	maxCountryLen = 60
)

// codeAttempts is how many order codes are drawn before giving up.
const codeAttempts = 5

// codReserveTTL is how long the books of a cash-on-delivery order are held:
// only long enough to write the order and sell them.
const codReserveTTL = 2 * time.Minute

// cleanupTimeout bounds the calls that undo a failed order.
const cleanupTimeout = 5 * time.Second

// payWindowShare: the gateway stops taking a payment a fifth of the hold
// before the books are released (3 minutes of a 15-minute hold). A payment
// the gateway accepts then always reaches us while the books are still held.
const payWindowShare = 5

// uuidPattern is the shape of an idempotency key.
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Store is what the service needs from the database. *Repository is the
// real one; tests use a fake.
type Store interface {
	CodeExists(ctx context.Context, code string) (bool, error)
	FindByIdempotencyKey(ctx context.Context, key string) (Order, error)
	Insert(ctx context.Context, key string, o *Order) error
	Cancel(ctx context.Context, id int64, reason string) error
	FindByCode(ctx context.Context, code string) (Order, error)
	MarkPaid(ctx context.Context, id int64, ref string) error
	CancelAwaiting(ctx context.Context, id int64, reason string) error
	ListExpiredAwaiting(ctx context.Context, limit int) ([]Order, error)
	ListByUser(ctx context.Context, userID string, limit int) ([]Order, error)
	LinkGuestOrders(ctx context.Context, userID, email string) (int64, error)
}

// Payments is what the service needs from the payment gateway. The real one
// is *vnpay.Client; tests use a fake. Nil when no gateway is configured.
type Payments interface {
	BuildPayURL(r vnpay.PayRequest) (string, error)
}

// Reserver is what the service needs from the catalogue. The real one is
// *catalogclient.Client; tests use a fake.
type Reserver interface {
	Reserve(ctx context.Context, holder string, ttl time.Duration, slugs []string) (catalogclient.Reservation, error)
	Release(ctx context.Context, holder string, ids []int) (int, error)
	Sell(ctx context.Context, holder string, ids []int) error
}

var (
	_ Store    = (*Repository)(nil)
	_ Reserver = (*catalogclient.Client)(nil)
	_ Payments = (*vnpay.Client)(nil)
)

// Service places orders and records their payments.
type Service struct {
	store    Store
	catalog  Reserver
	payments Payments
	log      *zap.Logger

	// hold is how long the books of an online order stay reserved.
	hold time.Duration
}

// NewService returns a Service. payments is nil when no gateway is
// configured; online payment methods are then refused.
func NewService(store Store, catalog Reserver, payments Payments, hold time.Duration, log *zap.Logger) *Service {
	return &Service{store: store, catalog: catalog, payments: payments, hold: hold, log: log}
}

// CreateInput is a checkout as the customer filled it in. It carries no
// prices: they are read from the catalogue.
type CreateInput struct {
	// Key is the idempotency key (a UUID) the web made for this checkout.
	Key string

	// Slugs are the books to buy; every listing is a single copy.
	Slugs []string

	PaymentMethod string

	Email      string
	Phone      string
	FullName   string
	Address    string
	City       string
	PostalCode *string
	Country    string

	// ClientIP is the customer's address; the payment gateway asks for it.
	ClientIP string

	// UserID is the signed-in customer (the JWT subject the gateway
	// forwards); nil for a guest checkout.
	UserID *string
}

// isOnline reports whether method is paid through the gateway.
func isOnline(method string) bool {
	return method == PaymentVietQR || method == PaymentCard
}

// Create places a guest order and returns it. replayed is true when the key
// had already placed an order and that one is returned instead of a new one.
//
// Cash on delivery is CONFIRMED and its books are sold at once. An online
// order is AWAITING_PAYMENT: its books stay reserved for s.hold, nothing is
// sold, and the order carries the PaymentURL to pay it on; ConfirmPayment
// finishes it.
//
// The steps, in order:
//
//  1. input.normalize(); a failure is ErrInvalidInput.
//  2. FindByIdempotencyKey: found with the same email is a replay, found with
//     another email is ErrKeyReused.
//  3. Draw an order code until CodeExists says it is free (codeAttempts
//     tries, then ErrNoFreeCode). The code is the holder of the books.
//  4. catalog.Reserve(code, ttl, slugs): codReserveTTL for cash on delivery,
//     s.hold online. Nothing was held when it returns an *UnavailableError;
//     after ErrCatalogUnavailable the outcome is unknown (see
//     catalogclient.Reserve).
//  5. store.Insert an UNPAID order built from the reservation: CONFIRMED, or
//     AWAITING_PAYMENT until the end of the hold online. Prices come from the
//     reservation, never from the client. When it fails the books must be
//     released; ErrDuplicateKey means a parallel request won, so return its
//     order as a replay.
//  6. Online: return the order with its PaymentURL; when signing fails,
//     cancel the order and release.
//  7. Cash on delivery: catalog.Sell(code, ids). When it fails, cancel the
//     order and release.
//
// Release and Cancel after a failure must not use ctx: the request may be
// the thing that was cancelled.
func (s *Service) Create(ctx context.Context, in CreateInput) (order Order, replayed bool, err error) {

	in, err = in.normalize()
	if err != nil {
		return Order{}, false, err
	}

	// Refused before any book is held: there is no gateway to pay it on.
	online := isOnline(in.PaymentMethod)
	if online && s.payments == nil {
		return Order{}, false, invalid("payment method %q is not available right now", in.PaymentMethod)
	}

	// Sorted, so two orders that share books lock them in the same order.
	slices.Sort(in.Slugs)

	existing, err := s.store.FindByIdempotencyKey(ctx, in.Key)
	switch {
	case err == nil:
		return s.replay(existing, in)
	case !errors.Is(err, ErrNotFound):
		return Order{}, false, err
	}

	code, err := s.freeCode(ctx)
	if err != nil {
		return Order{}, false, err
	}

	// An online order keeps its books while the customer pays.
	ttl := codReserveTTL
	if online {
		ttl = s.hold
	}

	res, err := s.reserve(ctx, code, ttl, in.Slugs)
	if err != nil {

		var unavailable *catalogclient.UnavailableError
		if errors.As(err, &unavailable) {
			// A parallel request with the same key may have taken the
			// books and placed the order: that one is the answer.
			if won, findErr := s.store.FindByIdempotencyKey(ctx, in.Key); findErr == nil {
				return s.replay(won, in)
			}
		}

		// After ErrCatalogUnavailable the books may be held, but their ids
		// are not known, so they cannot be released: the hold runs out by
		// itself after ttl.
		if errors.Is(err, catalogclient.ErrCatalogUnavailable) {
			s.log.Warn("reserve outcome unknown, leaving the hold to expire",
				zap.String("code", code), zap.Error(err))
		}

		return Order{}, false, err
	}

	ids := make([]int, len(res.Items))
	items := make([]Item, len(res.Items))
	var subtotal int64
	for i, b := range res.Items {
		ids[i] = b.ID
		items[i] = Item{
			BookID:   b.ID,
			Slug:     b.Slug,
			Title:    b.Title,
			Author:   b.Author,
			CoverURL: b.CoverURL,
			PriceVND: b.PriceVND,
		}
		subtotal += b.PriceVND
	}

	order = Order{
		Code:          code,
		UserID:        in.UserID,
		Status:        StatusConfirmed,
		PaymentMethod: in.PaymentMethod,
		PaymentStatus: PaymentUnpaid,
		Email:         in.Email,
		Phone:         in.Phone,
		FullName:      in.FullName,
		Address:       in.Address,
		City:          in.City,
		PostalCode:    in.PostalCode,
		Country:       in.Country,
		SubtotalVND:   subtotal,
		Items:         items,
	}

	// The hold ends when the catalogue says so: its clock releases the books,
	// not ours.
	if online {
		until := res.ReservedUntil
		order.Status = StatusAwaitingPayment
		order.ExpiresAt = &until
	}

	// The order is written before the books are sold: the catalogue cannot
	// take a sale back, while an order row can still be cancelled.
	if err := s.store.Insert(ctx, in.Key, &order); err != nil {

		s.release(ctx, code, ids)

		switch {
		case errors.Is(err, ErrDuplicateKey):
			// A parallel request with the same key wrote its order first.
			won, findErr := s.store.FindByIdempotencyKey(ctx, in.Key)
			if findErr != nil {
				return Order{}, false, findErr
			}
			return s.replay(won, in)

		case errors.Is(err, ErrDuplicateCode):
			// Another order drew the same code after the check above.
			return Order{}, false, ErrNoFreeCode
		}

		return Order{}, false, err
	}

	// An online order stops here: its books are sold once it is paid
	// (ConfirmPayment). Without a payment page it cannot be paid at all.
	if online {
		withURL, err := s.withPaymentURL(order, in.ClientIP)
		if err != nil {
			s.cancel(ctx, order, "payment could not be started")
			s.release(ctx, code, ids)

			return Order{}, false, err
		}

		return withURL, false, nil
	}

	if err := s.sell(ctx, code, ids); err != nil {

		s.cancel(ctx, order, "books could not be sold")
		s.release(ctx, code, ids)

		return Order{}, false, err
	}

	return order, false, nil
}

// maxListed bounds one "my orders" answer; a reader of a second-hand
// bookshop is not expected to have more.
const maxListed = 100

// ListMine returns the orders of a signed-in customer, newest first. Online
// orders count only once paid: one still awaiting payment, or abandoned
// unpaid, never became an order for the customer.
func (s *Service) ListMine(ctx context.Context, userID string) ([]Order, error) {
	return s.store.ListByUser(ctx, userID, maxListed)
}

// LinkGuestOrders gives a reader the orders placed as a guest with an email
// address they have confirmed (internal/accountevents), and returns how many.
// Orders that already belong to an account are left alone, so a repeated
// call links nothing more.
func (s *Service) LinkGuestOrders(ctx context.Context, userID, email string) (int64, error) {
	return s.store.LinkGuestOrders(ctx, userID, email)
}

// replay answers a repeated request with the order the key already placed,
// as long as the same customer is asking.
func (s *Service) replay(existing Order, in CreateInput) (Order, bool, error) {

	if !strings.EqualFold(existing.Email, in.Email) {
		return Order{}, false, ErrKeyReused
	}

	// A customer who pressed "Place order" twice still gets to the payment
	// page. The URL is signed again: it is never stored.
	if existing.Status == StatusAwaitingPayment && s.payments != nil {
		withURL, err := s.withPaymentURL(existing, in.ClientIP)
		if err != nil {
			return Order{}, false, err
		}
		return withURL, true, nil
	}

	return existing, true, nil
}

// withPaymentURL returns o with a signed PaymentURL for it. An order whose
// payment window has closed gets none: paying it would come too late.
//
// The window closes s.hold/payWindowShare before the hold ends (see
// payWindowShare); that deadline is the gateway's vnp_ExpireDate.
func (s *Service) withPaymentURL(o Order, ip string) (Order, error) {

	if o.ExpiresAt == nil {
		return o, nil
	}

	deadline := o.ExpiresAt.Add(-s.hold / payWindowShare)
	if !time.Now().Before(deadline) {
		return o, nil
	}

	url, err := s.payments.BuildPayURL(vnpay.PayRequest{
		TxnRef:    o.Code,
		AmountVND: o.SubtotalVND,
		// The gateway refuses accents and most punctuation here.
		OrderInfo: "Thanh toan don hang " + o.Code,
		IPAddr:    ip,
		ExpiresAt: deadline,
	})
	if err != nil {
		return Order{}, fmt.Errorf("sign payment URL of %s: %w", o.Code, err)
	}

	o.PaymentURL = url
	return o, nil
}

// ConfirmPayment records what the gateway says about the payment of an
// order: r comes from a verified IPN call or return redirect. It returns
// the order as it is afterwards.
//
// Errors, which the handler turns into the gateway's answer codes:
//   - ErrNotFound: no order has r.TxnRef as its code.
//   - ErrAmountMismatch: r.AmountVND is not the order's subtotal.
//   - ErrAlreadyProcessed: the order is no longer AWAITING_PAYMENT, or a
//     parallel call recorded the payment first.
//   - ErrPaidTooLate: paid, but the books could not be sold any more.
//   - anything else (catalogue or database down): nothing was recorded, so
//     the gateway should call again.
//
// A paid order has its books sold before it is marked paid: selling twice
// is harmless, while a paid order without books is not.
func (s *Service) ConfirmPayment(ctx context.Context, r vnpay.Result) (Order, error) {

	o, err := s.store.FindByCode(ctx, r.TxnRef)
	if err != nil {
		return Order{}, err
	}

	// Before the status, so a forged amount is reported even for an order
	// that is already finished.
	if r.AmountVND != o.SubtotalVND {
		return Order{}, ErrAmountMismatch
	}

	// The gateway repeats its call until it gets an answer: a repeat finds
	// the order finished and changes nothing.
	if o.Status != StatusAwaitingPayment {
		if o.Status == StatusCancelled && r.Succeeded() {
			s.log.Error("customer paid for a cancelled order, refund by hand",
				zap.String("code", o.Code), zap.String("transaction", r.TransactionNo))
		}
		return o, ErrAlreadyProcessed
	}

	ids := make([]int, len(o.Items))
	for i, it := range o.Items {
		ids[i] = it.BookID
	}

	if !r.Succeeded() {
		return s.paymentFailed(ctx, o, ids, r)
	}

	if err := s.sell(ctx, o.Code, ids); err != nil {

		// The catalogue did not answer: the books are still held, so the
		// next call of the gateway can sell them.
		var unavailable *catalogclient.UnavailableError
		if !errors.As(err, &unavailable) {
			return Order{}, err
		}

		return s.paidTooLate(ctx, o, r)
	}

	if err := s.store.MarkPaid(ctx, o.ID, r.TransactionNo); err != nil {
		if errors.Is(err, ErrNotAwaitingPayment) {
			// A parallel call recorded the payment first.
			return o, ErrAlreadyProcessed
		}
		return Order{}, err
	}

	now := time.Now()
	ref := r.TransactionNo
	o.Status, o.PaymentStatus, o.PaymentRef, o.PaidAt = StatusConfirmed, PaymentPaid, &ref, &now

	return o, nil
}

// FindByCode returns the order with code, or ErrNotFound.
func (s *Service) FindByCode(ctx context.Context, code string) (Order, error) {
	return s.store.FindByCode(ctx, code)
}

// paymentFailed cancels an order whose payment did not go through and gives
// its books back. Only the call that cancels the order releases them: if a
// parallel call marked it paid, the books are that order's.
func (s *Service) paymentFailed(ctx context.Context, o Order, ids []int, r vnpay.Result) (Order, error) {

	reason := "payment failed, gateway code " + r.ResponseCode
	if err := s.store.CancelAwaiting(ctx, o.ID, reason); err != nil {
		if errors.Is(err, ErrNotAwaitingPayment) {
			return o, ErrAlreadyProcessed
		}
		return Order{}, err
	}

	s.release(ctx, o.Code, ids)

	o.Status = StatusCancelled
	return o, nil
}

// paidTooLate handles a paid order whose books are gone: it is cancelled
// with the gateway's transaction in the reason, so it can be refunded.
func (s *Service) paidTooLate(ctx context.Context, o Order, r vnpay.Result) (Order, error) {

	s.log.Error("customer paid but the books are gone, refund by hand",
		zap.String("code", o.Code), zap.String("transaction", r.TransactionNo))

	reason := "paid after the books were released, refund VNPay transaction " + r.TransactionNo
	if err := s.store.CancelAwaiting(ctx, o.ID, reason); err != nil {
		if errors.Is(err, ErrNotAwaitingPayment) {
			return o, ErrAlreadyProcessed
		}
		return Order{}, err
	}

	return Order{}, ErrPaidTooLate
}

// freeCode draws order codes until one is not used by any order.
func (s *Service) freeCode(ctx context.Context) (string, error) {

	for range codeAttempts {

		code := newCode()

		taken, err := s.store.CodeExists(ctx, code)
		if err != nil {
			return "", err
		}
		if !taken {
			return code, nil
		}
	}

	return "", ErrNoFreeCode
}

// reserve holds the books for code for ttl. A catalogue that did not answer
// is asked once more with the same code: it lets the holder that already has
// the books take them again, so a repeat is safe.
func (s *Service) reserve(ctx context.Context, code string, ttl time.Duration, slugs []string) (catalogclient.Reservation, error) {

	res, err := s.catalog.Reserve(ctx, code, ttl, slugs)
	if errors.Is(err, catalogclient.ErrCatalogUnavailable) {
		res, err = s.catalog.Reserve(ctx, code, ttl, slugs)
	}

	return res, err
}

// sell turns the held books into sold ones. Selling twice counts as done, so
// a catalogue that did not answer is asked once more.
func (s *Service) sell(ctx context.Context, code string, ids []int) error {

	err := s.catalog.Sell(ctx, code, ids)
	if errors.Is(err, catalogclient.ErrCatalogUnavailable) {
		err = s.catalog.Sell(ctx, code, ids)
	}

	return err
}

// cleanupContext is for the calls that undo a failed order. It keeps the
// values of ctx but not its cancellation: the request may be the very thing
// that was cancelled, and the books must still be given back.
func cleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
}

// release gives the books back. When that fails there is nothing more to do
// but log it: the hold runs out by itself.
func (s *Service) release(ctx context.Context, code string, ids []int) {

	ctx, cancel := cleanupContext(ctx)
	defer cancel()

	if _, err := s.catalog.Release(ctx, code, ids); err != nil {
		s.log.Error("cannot release books of a failed order",
			zap.String("code", code), zap.Error(err))
	}
}

// cancel marks the order CANCELLED. When that fails it is logged: someone has
// to look at the order by hand.
func (s *Service) cancel(ctx context.Context, o Order, reason string) {

	ctx, cancel := cleanupContext(ctx)
	defer cancel()

	if err := s.store.Cancel(ctx, o.ID, reason); err != nil {
		s.log.Error("cannot cancel an order that could not be sold",
			zap.String("code", o.Code), zap.Error(err))
	}
}

// normalize trims the text fields and checks the input against the same
// rules as the database, so a bad request is refused before any book is held.
func (in CreateInput) normalize() (CreateInput, error) {

	if !uuidPattern.MatchString(in.Key) {
		return in, invalid("idempotency key must be a UUID")
	}

	if in.PaymentMethod != PaymentCOD && !isOnline(in.PaymentMethod) {
		return in, invalid("payment method %q is not supported yet", in.PaymentMethod)
	}

	if len(in.Slugs) < 1 || len(in.Slugs) > maxItems {
		return in, invalid("an order needs 1 to %d books", maxItems)
	}

	seen := make(map[string]bool, len(in.Slugs))
	slugs := make([]string, len(in.Slugs))
	for i, slug := range in.Slugs {
		slug = strings.TrimSpace(slug)
		if slug == "" || utf8.RuneCountInString(slug) > maxSlugLen {
			return in, invalid("a book slug must be 1 to %d characters", maxSlugLen)
		}
		if seen[slug] {
			return in, invalid("book %q is listed twice", slug)
		}
		seen[slug] = true
		slugs[i] = slug
	}
	in.Slugs = slugs

	var err error
	if in.Email, err = required("email", in.Email, maxEmailLen); err != nil {
		return in, err
	}

	// ParseAddress also accepts "Name <a@b.com>"; only a bare address is wanted.
	if addr, err := mail.ParseAddress(in.Email); err != nil || addr.Address != in.Email {
		return in, invalid("email is not a valid address")
	}

	if in.Phone, err = required("phone", in.Phone, maxPhoneLen); err != nil {
		return in, err
	}
	if in.FullName, err = required("full name", in.FullName, maxNameLen); err != nil {
		return in, err
	}
	if in.Address, err = required("address", in.Address, maxAddressLen); err != nil {
		return in, err
	}
	if in.City, err = required("city", in.City, maxCityLen); err != nil {
		return in, err
	}
	if in.Country, err = required("country", in.Country, maxCountryLen); err != nil {
		return in, err
	}

	// An empty postal code is the same as none.
	if in.PostalCode != nil {
		postal := strings.TrimSpace(*in.PostalCode)
		switch {
		case postal == "":
			in.PostalCode = nil
		case utf8.RuneCountInString(postal) > maxPostalLen:
			return in, invalid("postal code must be at most %d characters", maxPostalLen)
		default:
			in.PostalCode = &postal
		}
	}

	return in, nil
}

// required trims value and checks it is 1 to max characters long.
func required(field, value string, max int) (string, error) {

	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > max {
		return value, invalid("%s must be 1 to %d characters", field, max)
	}

	return value, nil
}

// invalid builds an ErrInvalidInput with a message for the customer.
func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, fmt.Sprintf(format, args...))
}
