package accountevents

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

type fakeQueue struct {
	messages []Message
	deleted  []string
	poisoned []string
}

func (f *fakeQueue) Receive(ctx context.Context, n int, hide time.Duration) ([]Message, error) {
	return f.messages, nil
}

func (f *fakeQueue) Delete(ctx context.Context, m Message) error {
	f.deleted = append(f.deleted, m.ID)
	return nil
}

func (f *fakeQueue) Poison(ctx context.Context, m Message) error {
	f.poisoned = append(f.poisoned, m.ID)
	return nil
}

type fakeLinker struct {
	err error
	got []string
}

func (f *fakeLinker) LinkGuestOrders(ctx context.Context, userID, email string) (int64, error) {
	f.got = append(f.got, userID+" "+email)
	return 2, f.err
}

const event = `{"type":"user.email_verified","userId":"u1","email":"reader@example.com","verifiedAt":"2026-10-16T09:00:00Z"}`

func TestConsumerLinksAndDeletes(t *testing.T) {

	queue := &fakeQueue{messages: []Message{{ID: "m1", Text: event, DequeueCount: 1}}}
	linker := &fakeLinker{}

	NewConsumer(queue, linker, zap.NewNop()).receiveBatch(context.Background())

	if len(linker.got) != 1 || linker.got[0] != "u1 reader@example.com" {
		t.Errorf("linked %v", linker.got)
	}
	if len(queue.deleted) != 1 || len(queue.poisoned) != 0 {
		t.Errorf("deleted %v, poisoned %v", queue.deleted, queue.poisoned)
	}
}

func TestConsumerLeavesAFailureForAnotherTry(t *testing.T) {

	queue := &fakeQueue{messages: []Message{{ID: "m1", Text: event, DequeueCount: 1}}}

	NewConsumer(queue, &fakeLinker{err: errors.New("database down")}, zap.NewNop()).receiveBatch(context.Background())

	if len(queue.deleted) != 0 || len(queue.poisoned) != 0 {
		t.Errorf("deleted %v, poisoned %v; want the message left on the queue", queue.deleted, queue.poisoned)
	}
}

func TestConsumerPoisonsWhatCannotWork(t *testing.T) {

	queue := &fakeQueue{messages: []Message{
		{ID: "not-json", Text: "{", DequeueCount: 1},
		{ID: "other-type", Text: `{"type":"user.deleted","userId":"u1","email":"a@b.c"}`, DequeueCount: 1},
		{ID: "too-many-tries", Text: event, DequeueCount: maxDequeueCount},
	}}

	NewConsumer(queue, &fakeLinker{err: errors.New("database down")}, zap.NewNop()).receiveBatch(context.Background())

	if len(queue.poisoned) != 3 {
		t.Errorf("poisoned %v, want all three", queue.poisoned)
	}
}
