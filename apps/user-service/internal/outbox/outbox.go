// Package outbox delivers messages to Azure Storage queues after the
// database change that caused them has committed.
//
// A message is written to the outbox table in the same transaction as the
// change (a new account and its confirmation email, say), so neither can
// happen without the other. A Relay then sends each row to its queue and
// deletes it. A row whose send fails stays and is sent again: delivery is
// at least once, and every consumer must cope with a message twice.
package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

// Queues the user service writes to.
const (
	// QueueEmail is read by the sendEmail Azure Function, which mails each
	// message (an Email) through SMTP.
	QueueEmail = "email-outbox"
)

// Message is one outbox row.
type Message struct {
	ID      int64  `db:"id"`
	Queue   string `db:"queue"`
	Payload string `db:"payload"`
}

// Email is the payload of a QueueEmail message: a whole email, already
// written, so the Function only has to send it.
type Email struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	HTML    string `json:"html"`
}

// NewEmail returns the outbox message that sends e.
func NewEmail(e Email) (Message, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return Message{}, err
	}
	return Message{Queue: QueueEmail, Payload: string(b)}, nil
}

// Add writes messages in tx, the transaction of the change they report.
func Add(ctx context.Context, tx *sqlx.Tx, messages ...Message) error {
	for _, m := range messages {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO outbox (queue, payload) VALUES (@queue, @payload)`,
			sql.Named("queue", m.Queue), sql.Named("payload", m.Payload)); err != nil {
			return err
		}
	}
	return nil
}

// Store claims and removes outbox rows for a Relay.
type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// Claim takes up to limit messages that no relay is sending, oldest first,
// and hides them from other relays (replicas) for lease. A message not
// deleted by then is claimed again.
func (s *Store) Claim(ctx context.Context, limit int, lease time.Duration) ([]Message, error) {

	// READPAST skips rows another replica is claiming right now.
	var claimed []Message
	err := s.db.SelectContext(ctx, &claimed, `
WITH next AS (
    SELECT TOP (@limit) id, queue, payload, locked_until, attempts
    FROM outbox WITH (UPDLOCK, READPAST, ROWLOCK)
    WHERE locked_until IS NULL OR locked_until < SYSUTCDATETIME()
    ORDER BY id
)
UPDATE next
SET locked_until = DATEADD(millisecond, @lease_ms, SYSUTCDATETIME()), attempts = attempts + 1
OUTPUT inserted.id, inserted.queue, inserted.payload`,
		sql.Named("limit", limit), sql.Named("lease_ms", lease.Milliseconds()))

	return claimed, err
}

// Delete removes a message once it is on its queue.
func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM outbox WHERE id = @id`, sql.Named("id", id))
	return err
}
