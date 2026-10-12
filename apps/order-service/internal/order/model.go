package order

import "time"

// Statuses of an order (orders.status).
const (
	StatusAwaitingPayment = "AWAITING_PAYMENT"
	StatusConfirmed       = "CONFIRMED"
	StatusCancelled       = "CANCELLED"
)

// Payment methods (orders.payment_method). Only cash on delivery is handled
// for now; online methods come with the payment gateway.
const (
	PaymentCOD = "cod"
)

// Payment statuses (orders.payment_status).
const (
	PaymentUnpaid = "UNPAID"
)

// Order is one orders row with its items. The db tags match the columns
// selected by the repository; the json tags are the API shape.
type Order struct {
	ID             int64      `db:"id"               json:"-"`
	Code           string     `db:"code"             json:"code"`
	UserID         *string    `db:"user_id"          json:"-"`
	Status         string     `db:"status"           json:"status"`
	PaymentMethod  string     `db:"payment_method"   json:"paymentMethod"`
	PaymentStatus  string     `db:"payment_status"   json:"paymentStatus"`
	Email          string     `db:"email"            json:"email"`
	Phone          string     `db:"phone"            json:"phone"`
	FullName       string     `db:"full_name"        json:"fullName"`
	Address        string     `db:"address"          json:"address"`
	City           string     `db:"city"             json:"city"`
	PostalCode     *string    `db:"postal_code"      json:"postalCode"`
	Country        string     `db:"country"          json:"country"`
	SubtotalVND    int64      `db:"subtotal_vnd"     json:"subtotalVnd"`
	ShippingFeeVND *int64     `db:"shipping_fee_vnd" json:"shippingFeeVnd"`
	ExpiresAt      *time.Time `db:"expires_at"       json:"expiresAt"`
	CreatedAt      time.Time  `db:"created_at"       json:"createdAt"`
	Items          []Item     `db:"-"                json:"items"`
}

// Item is a book as it was when the order was placed. book_id points to the
// catalogue database, so it is kept but not shown.
type Item struct {
	BookID   int     `db:"book_id"   json:"-"`
	Slug     string  `db:"book_slug" json:"slug"`
	Title    string  `db:"title"     json:"title"`
	Author   *string `db:"author"    json:"author"`
	CoverURL *string `db:"cover_url" json:"coverUrl"`
	PriceVND int64   `db:"price_vnd" json:"priceVnd"`
}
