// Package accountevents reads what the user service reports about accounts.
//
// When a reader confirms their email address, the user service puts a
// user.email_verified message on the user-email-verified Azure Storage
// queue. The orders placed as a guest with that address then join the
// reader's account, as WooCommerce does after an email is confirmed: only a
// confirmed address is trusted, or anyone could register someone else's
// email and see their orders.
package accountevents

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/queueerror"
	"go.uber.org/zap"
)

// Queue names, as the user service writes them.
const (
	QueueEmailVerified = "user-email-verified"

	// poisonSuffix names the queue a message that keeps failing is moved
	// to, the same convention as Azure Functions queue triggers.
	poisonSuffix = "-poison"
)

// Consumer timing: how many messages one read takes, how long a read
// message stays hidden from other replicas while it is handled, how long to
// wait when the queue is empty, and how many tries a message gets.
const (
	batchSize         = 16
	visibilityTimeout = 30 * time.Second
	idleWait          = 5 * time.Second
	maxDequeueCount   = 5
)

// EmailVerified is the payload of a QueueEmailVerified message, the shape
// the user service writes (its outbox.EmailVerified).
type EmailVerified struct {
	Type       string    `json:"type"`
	UserID     string    `json:"userId"`
	Email      string    `json:"email"`
	VerifiedAt time.Time `json:"verifiedAt"`
}

// errMalformed means a message can never be handled.
var errMalformed = errors.New("malformed message")

// Linker gives a reader the orders placed as a guest with their address.
// *order.Service outside tests.
type Linker interface {
	LinkGuestOrders(ctx context.Context, userID, email string) (int64, error)
}

// Queue is the part of an Azure Storage queue the consumer uses.
type Queue interface {
	Receive(ctx context.Context, n int, hide time.Duration) ([]Message, error)
	Delete(ctx context.Context, m Message) error
	// Poison moves m to the poison queue.
	Poison(ctx context.Context, m Message) error
}

// Message is one received queue message.
type Message struct {
	ID           string
	PopReceipt   string
	Text         string
	DequeueCount int64
}

// Consumer handles QueueEmailVerified messages. Handling one twice is
// harmless: only orders that belong to nobody are linked.
type Consumer struct {
	queue  Queue
	linker Linker
	log    *zap.Logger
}

func NewConsumer(queue Queue, linker Linker, log *zap.Logger) *Consumer {
	return &Consumer{queue: queue, linker: linker, log: log}
}

// Run reads the queue until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) {
	for {
		n := c.receiveBatch(ctx)
		if n == batchSize {
			// More may be waiting.
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(idleWait):
		}
	}
}

// receiveBatch handles one read of the queue and returns how many messages
// it got.
func (c *Consumer) receiveBatch(ctx context.Context) int {

	messages, err := c.queue.Receive(ctx, batchSize, visibilityTimeout)
	if err != nil {
		if ctx.Err() == nil {
			c.log.Warn("cannot read the account events queue", zap.Error(err))
		}
		return 0
	}

	for _, m := range messages {
		c.handle(ctx, m)
	}

	return len(messages)
}

// handle links the orders of one message and removes it. A message that
// fails stays on the queue and comes back after visibilityTimeout; one that
// keeps failing, or can never work, goes to the poison queue.
func (c *Consumer) handle(ctx context.Context, m Message) {

	err := c.link(ctx, m)
	switch {
	case err == nil:
		if err := c.queue.Delete(ctx, m); err != nil {
			// Handled but still queued: it comes back and links nothing.
			c.log.Warn("cannot delete handled account event", zap.String("id", m.ID), zap.Error(err))
		}

	case errors.Is(err, errMalformed) || m.DequeueCount >= maxDequeueCount:
		c.log.Error("moving account event to the poison queue", zap.String("id", m.ID), zap.Int64("tries", m.DequeueCount), zap.Error(err))
		if err := c.queue.Poison(ctx, m); err != nil {
			c.log.Warn("cannot move account event to the poison queue", zap.String("id", m.ID), zap.Error(err))
		}

	default:
		c.log.Warn("cannot handle account event, it will be retried", zap.String("id", m.ID), zap.Int64("tries", m.DequeueCount), zap.Error(err))
	}
}

func (c *Consumer) link(ctx context.Context, m Message) error {

	var event EmailVerified
	if err := json.Unmarshal([]byte(m.Text), &event); err != nil {
		return errors.Join(errMalformed, err)
	}
	if event.Type != "user.email_verified" || event.UserID == "" || event.Email == "" {
		return errMalformed
	}

	n, err := c.linker.LinkGuestOrders(ctx, event.UserID, event.Email)
	if err != nil {
		return err
	}

	// The address is personal data: log the account only.
	c.log.Info("linked guest orders to a confirmed account", zap.String("user_id", event.UserID), zap.Int64("orders", n))
	return nil
}

// StorageQueue is a Queue on Azure Storage (Azurite locally).
type StorageQueue struct {
	queue  *azqueue.QueueClient
	poison *azqueue.QueueClient
}

// NewStorageQueue connects to queue name with a storage account connection
// string. Nothing is called until the first read.
func NewStorageQueue(connectionString, name string) (*StorageQueue, error) {

	service, err := azqueue.NewServiceClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, err
	}

	return &StorageQueue{
		queue:  service.NewQueueClient(name),
		poison: service.NewQueueClient(name + poisonSuffix),
	}, nil
}

// Receive reads up to n messages and hides them for hide. A queue that does
// not exist yet (nothing was ever written) has no messages. Texts are
// base64, as the user service writes them.
func (q *StorageQueue) Receive(ctx context.Context, n int, hide time.Duration) ([]Message, error) {

	count, seconds := int32(n), int32(hide.Seconds())
	resp, err := q.queue.DequeueMessages(ctx, &azqueue.DequeueMessagesOptions{NumberOfMessages: &count, VisibilityTimeout: &seconds})
	if queueerror.HasCode(err, queueerror.QueueNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	messages := make([]Message, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		text, err := base64.StdEncoding.DecodeString(*m.MessageText)
		if err != nil {
			// Kept as is: handling it fails and it goes to the poison queue.
			text = []byte(*m.MessageText)
		}
		messages = append(messages, Message{ID: *m.MessageID, PopReceipt: *m.PopReceipt, Text: string(text), DequeueCount: *m.DequeueCount})
	}
	return messages, nil
}

func (q *StorageQueue) Delete(ctx context.Context, m Message) error {
	_, err := q.queue.DeleteMessage(ctx, m.ID, m.PopReceipt, nil)
	return err
}

func (q *StorageQueue) Poison(ctx context.Context, m Message) error {

	content := base64.StdEncoding.EncodeToString([]byte(m.Text))
	_, err := q.poison.EnqueueMessage(ctx, content, nil)
	if queueerror.HasCode(err, queueerror.QueueNotFound) {
		if _, err := q.poison.Create(ctx, nil); err != nil && !queueerror.HasCode(err, queueerror.QueueAlreadyExists) {
			return err
		}
		_, err = q.poison.EnqueueMessage(ctx, content, nil)
	}
	if err != nil {
		return err
	}

	return q.Delete(ctx, m)
}
