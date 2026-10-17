package outbox

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/queueerror"
	"go.uber.org/zap"
)

// Relay timing: how often the outbox is read, how many rows at a time, and
// how long a claimed row is hidden from other replicas.
const (
	pollInterval = 2 * time.Second
	batchSize    = 20
	claimLease   = 30 * time.Second
)

// claimer is what a Relay needs from the outbox table; *Store outside tests.
type claimer interface {
	Claim(ctx context.Context, limit int, lease time.Duration) ([]Message, error)
	Delete(ctx context.Context, id int64) error
}

// Sender puts one message on a queue.
type Sender interface {
	Send(ctx context.Context, queue, payload string) error
}

// Relay moves outbox rows to their queues.
type Relay struct {
	store  claimer
	sender Sender
	log    *zap.Logger
}

func NewRelay(store claimer, sender Sender, log *zap.Logger) *Relay {
	return &Relay{store: store, sender: sender, log: log}
}

// Run relays until ctx is cancelled.
func (r *Relay) Run(ctx context.Context) {

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		// Drain a backlog without waiting a tick between full batches.
		for r.relayBatch(ctx) == batchSize {
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// relayBatch sends one batch and returns how many rows it claimed.
func (r *Relay) relayBatch(ctx context.Context) int {

	messages, err := r.store.Claim(ctx, batchSize, claimLease)
	if err != nil {
		if ctx.Err() == nil {
			r.log.Warn("cannot read the outbox", zap.Error(err))
		}
		return 0
	}

	for _, m := range messages {
		if err := r.sender.Send(ctx, m.Queue, m.Payload); err != nil {
			// The claim runs out and the next pass sends it again.
			r.log.Warn("cannot send outbox message", zap.Int64("id", m.ID), zap.String("queue", m.Queue), zap.Error(err))
			continue
		}
		if err := r.store.Delete(ctx, m.ID); err != nil {
			// Sent but still in the outbox: it will be sent twice.
			r.log.Warn("cannot delete sent outbox message", zap.Int64("id", m.ID), zap.Error(err))
		}
	}

	return len(messages)
}

// QueueSender sends to Azure Storage queues (Azurite locally).
type QueueSender struct {
	service *azqueue.ServiceClient
}

// NewQueueSender connects with a storage account connection string. It
// does not call the account: a queue that does not exist yet is created on
// its first message, so the service starts even while storage is down.
func NewQueueSender(connectionString string) (*QueueSender, error) {

	service, err := azqueue.NewServiceClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, err
	}

	return &QueueSender{service: service}, nil
}

// Send enqueues payload base64-encoded, the encoding Azure Functions queue
// triggers expect by default.
func (s *QueueSender) Send(ctx context.Context, queue, payload string) error {

	q := s.service.NewQueueClient(queue)
	content := base64.StdEncoding.EncodeToString([]byte(payload))

	_, err := q.EnqueueMessage(ctx, content, nil)
	if !queueerror.HasCode(err, queueerror.QueueNotFound) {
		return err
	}

	if _, err := q.Create(ctx, nil); err != nil && !queueerror.HasCode(err, queueerror.QueueAlreadyExists) {
		return err
	}
	_, err = q.EnqueueMessage(ctx, content, nil)
	return err
}
