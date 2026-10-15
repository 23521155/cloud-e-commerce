package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

type fakeClaimer struct {
	rows    []Message
	deleted []int64
}

func (f *fakeClaimer) Claim(ctx context.Context, limit int, lease time.Duration) ([]Message, error) {
	var out []Message
	for _, m := range f.rows {
		if len(out) == limit {
			break
		}
		out = append(out, m)
	}
	return out, nil
}

func (f *fakeClaimer) Delete(ctx context.Context, id int64) error {
	f.deleted = append(f.deleted, id)
	for i, m := range f.rows {
		if m.ID == id {
			f.rows = append(f.rows[:i], f.rows[i+1:]...)
			break
		}
	}
	return nil
}

type fakeSender struct {
	sent   []string
	failOn string
}

func (f *fakeSender) Send(ctx context.Context, queue, payload string) error {
	if payload == f.failOn {
		return errors.New("queue unreachable")
	}
	f.sent = append(f.sent, queue+":"+payload)
	return nil
}

func TestRelaySendsThenDeletes(t *testing.T) {

	store := &fakeClaimer{rows: []Message{{ID: 1, Queue: QueueEmail, Payload: "a"}, {ID: 2, Queue: QueueEmail, Payload: "b"}}}
	sender := &fakeSender{}

	n := NewRelay(store, sender, zap.NewNop()).relayBatch(context.Background())

	if n != 2 || len(sender.sent) != 2 || sender.sent[0] != "email-outbox:a" {
		t.Errorf("claimed %d, sent %v", n, sender.sent)
	}
	if len(store.rows) != 0 {
		t.Errorf("rows left in the outbox: %v", store.rows)
	}
}

func TestRelayKeepsWhatItCouldNotSend(t *testing.T) {

	store := &fakeClaimer{rows: []Message{{ID: 1, Queue: QueueEmail, Payload: "a"}, {ID: 2, Queue: QueueEmail, Payload: "b"}}}
	sender := &fakeSender{failOn: "a"}

	NewRelay(store, sender, zap.NewNop()).relayBatch(context.Background())

	if len(store.rows) != 1 || store.rows[0].ID != 1 {
		t.Errorf("outbox = %v, want only the unsent message", store.rows)
	}
	if len(sender.sent) != 1 || sender.sent[0] != "email-outbox:b" {
		t.Errorf("sent %v, want the other message still sent", sender.sent)
	}
}

func TestNewEmailIsAnEmailOutboxMessage(t *testing.T) {

	want := Email{To: "reader@example.com", Subject: "Hi", Text: "t", HTML: "<p>t</p>"}
	m, err := NewEmail(want)
	if err != nil {
		t.Fatal(err)
	}

	var got Email
	if err := json.Unmarshal([]byte(m.Payload), &got); err != nil {
		t.Fatal(err)
	}
	if m.Queue != QueueEmail || got != want {
		t.Errorf("message = %+v, payload %+v", m, got)
	}
}
