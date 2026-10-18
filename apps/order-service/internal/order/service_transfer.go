package order

import (
	"context"
	"errors"
	"time"

	"github.com/fizzisme/order-service/internal/catalogclient"
	"go.uber.org/zap"
)

// ConfirmTransfer records a bank transfer to the shop's account: code is the
// order code found in the transfer text, amountVND what arrived, and ref the
// bank's or SePay's id of the transfer (stored as the order's payment_ref). It
// returns the order as it is afterwards.
//
// A transfer is always a success: unlike a card payment there is no failed
// answer to handle, so an order is never cancelled here for a bad payment.
//
// Errors, which the webhook handler turns into an answer for SePay:
//   - ErrNotFound: no order has this code.
//   - ErrAmountMismatch: amountVND is not the order's subtotal. The order is
//     left as it is, for the owner to settle by hand.
//   - ErrAlreadyProcessed: the order is no longer AWAITING_PAYMENT, or a
//     parallel call recorded the transfer first. SePay repeats a webhook
//     until it gets an answer, so this is the normal answer to a repeat.
//   - ErrPaidTooLate: the money arrived but the books could not be sold any
//     more: the order is cancelled and the transfer must be refunded by hand.
//   - anything else (catalogue or database down): nothing was recorded, so
//     SePay should call again.
//
// As in ConfirmPayment, the books are sold before the order is marked paid:
// selling twice is harmless, while a paid order without books is not.
func (s *Service) ConfirmTransfer(ctx context.Context, code string, amountVND int64, ref string) (Order, error) {

	o, err := s.store.FindByCode(ctx, code)
	if err != nil {
		return Order{}, err
	}

	// Before the status, so a wrong amount is reported even for an order that
	// is already finished.
	if amountVND != o.SubtotalVND {
		return Order{}, ErrAmountMismatch
	}

	if o.Status != StatusAwaitingPayment {
		if o.Status == StatusCancelled {
			s.log.Error("customer paid for a cancelled order, refund by hand",
				zap.String("code", o.Code), zap.String("transfer", ref))
		}
		return o, ErrAlreadyProcessed
	}

	ids := make([]int, len(o.Items))
	for i, it := range o.Items {
		ids[i] = it.BookID
	}

	if err := s.sell(ctx, o.Code, ids); err != nil {

		// The catalogue did not answer: the books are still held, so the
		// next call of SePay can sell them.
		var unavailable *catalogclient.UnavailableError
		if !errors.As(err, &unavailable) {
			return Order{}, err
		}

		s.log.Error("customer paid but the books are gone, refund by hand",
			zap.String("code", o.Code), zap.String("transfer", ref))

		reason := "paid after the books were released, refund transfer " + ref
		if err := s.store.CancelAwaiting(ctx, o.ID, reason); err != nil {
			if errors.Is(err, ErrNotAwaitingPayment) {
				return o, ErrAlreadyProcessed
			}
			return Order{}, err
		}

		return Order{}, ErrPaidTooLate
	}

	if err := s.store.MarkPaid(ctx, o.ID, ref); err != nil {
		if errors.Is(err, ErrNotAwaitingPayment) {
			// A parallel call recorded the transfer first.
			return o, ErrAlreadyProcessed
		}
		return Order{}, err
	}

	now := time.Now()
	o.Status, o.PaymentStatus, o.PaymentRef, o.PaidAt = StatusConfirmed, PaymentPaid, &ref, &now

	return o, nil
}
