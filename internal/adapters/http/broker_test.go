package httpadapter_test

import (
	"context"
	"testing"
	"time"

	httpadapter "github.com/karo/cuttlegate/internal/adapters/http"
	"github.com/karo/cuttlegate/internal/domain/ports"
)

// testEvent is a minimal DomainEvent for broker tests.
type testEvent struct {
	typ        string
	occurredAt time.Time
}

func (e testEvent) EventType() string     { return e.typ }
func (e testEvent) OccurredAt() time.Time { return e.occurredAt }

func newTestEvent(typ string) testEvent {
	return testEvent{typ: typ, occurredAt: time.Now()}
}

func TestBroker_SingleSubscriberReceivesEvent(t *testing.T) {
	broker := httpadapter.NewBroker(8)
	defer broker.Shutdown()

	ch, unsub := broker.Subscribe()
	defer unsub()

	evt := newTestEvent("flag.state_changed")
	if err := broker.Publish(context.Background(), evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case got := <-ch:
		if got.EventType() != "flag.state_changed" {
			t.Errorf("EventType = %q, want %q", got.EventType(), "flag.state_changed")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestBroker_MultipleSubscribersReceiveSameEvent(t *testing.T) {
	broker := httpadapter.NewBroker(8)
	defer broker.Shutdown()

	const n = 3
	channels := make([]<-chan ports.DomainEvent, n)
	unsubs := make([]func(), n)
	for i := range n {
		channels[i], unsubs[i] = broker.Subscribe()
		defer unsubs[i]()
	}

	evt := newTestEvent("test.event")
	if err := broker.Publish(context.Background(), evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	for i, ch := range channels {
		select {
		case got := <-ch:
			if got.EventType() != "test.event" {
				t.Errorf("subscriber %d: EventType = %q, want %q", i, got.EventType(), "test.event")
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out waiting for event", i)
		}
	}
}

func TestBroker_UnsubscribedClientStopsReceiving(t *testing.T) {
	broker := httpadapter.NewBroker(8)
	defer broker.Shutdown()

	ch, unsub := broker.Subscribe()
	unsub()

	evt := newTestEvent("test.event")
	if err := broker.Publish(context.Background(), evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Channel should be closed — read should return zero value immediately.
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed after unsubscribe")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out — channel should be closed")
	}
}

func TestBroker_ShutdownClosesAllChannels(t *testing.T) {
	broker := httpadapter.NewBroker(8)

	ch1, _ := broker.Subscribe()
	ch2, _ := broker.Subscribe()

	broker.Shutdown()

	for i, ch := range []<-chan ports.DomainEvent{ch1, ch2} {
		select {
		case _, ok := <-ch:
			if ok {
				t.Errorf("subscriber %d: channel should be closed after shutdown", i)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out — channel should be closed", i)
		}
	}
}

func TestBroker_PublishDoesNotBlockOnFullBuffer(t *testing.T) {
	broker := httpadapter.NewBroker(1)
	defer broker.Shutdown()

	ch, unsub := broker.Subscribe()
	defer unsub()

	// Fill the buffer.
	if err := broker.Publish(context.Background(), newTestEvent("first")); err != nil {
		t.Fatalf("first Publish: %v", err)
	}

	// Second publish should not block — event is dropped.
	done := make(chan struct{})
	go func() {
		_ = broker.Publish(context.Background(), newTestEvent("second"))
		close(done)
	}()

	select {
	case <-done:
		// success — Publish returned without blocking
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on full buffer")
	}

	// Only the first event should be in the channel.
	got := <-ch
	if got.EventType() != "first" {
		t.Errorf("EventType = %q, want %q", got.EventType(), "first")
	}

	// Channel should be empty now.
	select {
	case evt := <-ch:
		t.Errorf("unexpected event in channel: %v", evt.EventType())
	default:
		// expected — second event was dropped
	}
}

func TestBroker_PublishAfterShutdownReturnsError(t *testing.T) {
	broker := httpadapter.NewBroker(8)
	broker.Shutdown()

	err := broker.Publish(context.Background(), newTestEvent("test.event"))
	if err == nil {
		t.Fatal("expected error from Publish after Shutdown")
	}
}

func TestBroker_SubscribeAfterShutdownReturnsNil(t *testing.T) {
	broker := httpadapter.NewBroker(8)
	broker.Shutdown()

	ch, unsub := broker.Subscribe()
	if ch != nil {
		t.Error("expected nil channel from Subscribe after Shutdown")
	}
	if unsub != nil {
		t.Error("expected nil unsubscribe from Subscribe after Shutdown")
	}
}

func TestBroker_ShutdownIsIdempotent(t *testing.T) {
	broker := httpadapter.NewBroker(8)
	broker.Subscribe()
	broker.Shutdown()
	broker.Shutdown() // should not panic
}

func TestBroker_UnsubscribeIsIdempotent(t *testing.T) {
	broker := httpadapter.NewBroker(8)
	defer broker.Shutdown()

	_, unsub := broker.Subscribe()
	unsub()
	unsub() // should not panic
}
