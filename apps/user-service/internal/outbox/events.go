package outbox

import (
	"encoding/json"
	"time"
)

// QueueUserEmailVerified is read by the order service: an address confirmed
// by its owner lets the orders placed as a guest with it join the account.
const QueueUserEmailVerified = "user-email-verified"

// EmailVerified is the payload of a QueueUserEmailVerified message. The order
// service has its own copy of this shape.
type EmailVerified struct {
	Type       string    `json:"type"`
	UserID     string    `json:"userId"`
	Email      string    `json:"email"`
	VerifiedAt time.Time `json:"verifiedAt"`
}

// NewEmailVerified returns the message telling that userID confirmed email.
func NewEmailVerified(userID, email string, verifiedAt time.Time) (Message, error) {
	b, err := json.Marshal(EmailVerified{Type: "user.email_verified", UserID: userID, Email: email, VerifiedAt: verifiedAt.UTC()})
	if err != nil {
		return Message{}, err
	}
	return Message{Queue: QueueUserEmailVerified, Payload: string(b)}, nil
}
