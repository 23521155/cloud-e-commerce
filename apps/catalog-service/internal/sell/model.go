package sell

import "time"

const (
	StatusPendingReview = "PENDING_REVIEW"
	StatusApproved      = "APPROVED"
	StatusRejected      = "REJECTED"
)

// Request represents a seller's submission awaiting admin review.
type Request struct {
	ID             int64     `db:"id" json:"id"`
	Title          string    `db:"title" json:"title"`
	Author         string    `db:"author" json:"author"`
	Year           *int      `db:"year" json:"year"`
	Edition        *string   `db:"edition" json:"edition"`
	Binding        string    `db:"binding" json:"binding"`
	Grade          string    `db:"grade" json:"grade"`
	Faults         string    `db:"faults" json:"faults"`
	Virtues        *string   `db:"virtues" json:"virtues"`
	AskingPriceVND *int64    `db:"asking_price_vnd" json:"askingPriceVnd"`
	SellerName     string    `db:"seller_name" json:"sellerName"`
	SellerEmail    string    `db:"seller_email" json:"sellerEmail"`
	SellerPhone    *string   `db:"seller_phone" json:"sellerPhone"`
	Status         string    `db:"status" json:"status"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`
	Photos         []Photo   `db:"-" json:"photos"`
}

// CreateInput uses the existing /sell form field names.
type CreateInput struct {
	Title   string `form:"title" binding:"required,max=600"`
	Author  string `form:"author" binding:"required,max=150"`
	Year    *int   `form:"year" binding:"omitempty,min=1,max=32767"`
	Edition string `form:"edition" binding:"max=200"`

	Binding string `form:"binding" binding:"required,oneof=Hardcover Paperback Other"`
	Grade   string `form:"grade" binding:"required,oneof=Fine 'Very good' Good Fair Poor 'Not sure'"`

	Faults  string `form:"faults" binding:"required,max=4000"`
	Virtues string `form:"virtues" binding:"max=4000"`
	Asking  *int64 `form:"asking" binding:"omitempty,gt=0"`

	Name  string `form:"name" binding:"required,max=150"`
	Email string `form:"email" binding:"required,email,max=254"`
	Phone string `form:"phone" binding:"max=30"`
}