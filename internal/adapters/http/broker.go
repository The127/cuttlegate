package httpadapter

import (
	"context"
	"errors"
	"sync"

	"github.com/karo/cuttlegate/internal/domain/ports"
)

// ErrBrokerClosed is returned when Publish is called after Shutdown.
var ErrBrokerClosed = errors.New("broker is closed")

// Broker is an in-process event fanout that implements [ports.EventPublisher].
// It delivers domain events to all subscribers via buffered channels.
// All methods are safe for concurrent use.
type Broker struct {
	mu          sync.RWMutex
	subscribers map[*subscriber]struct{}
	bufSize     int
	closed      bool
}

type subscriber struct {
	ch chan ports.DomainEvent
}

// NewBroker creates a Broker with the given per-subscriber channel buffer size.
func NewBroker(bufferSize int) *Broker {
	return &Broker{
		subscribers: make(map[*subscriber]struct{}),
		bufSize:     bufferSize,
	}
}

// Publish fans out the event to all current subscribers. If a subscriber's
// channel buffer is full, the event is dropped for that subscriber.
// Returns an error if the broker has been shut down.
func (b *Broker) Publish(_ context.Context, event ports.DomainEvent) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return ErrBrokerClosed
	}

	for sub := range b.subscribers {
		select {
		case sub.ch <- event:
		default:
		}
	}

	return nil
}

// Subscribe returns a read-only channel that receives published events and an
// unsubscribe function. Calling unsubscribe closes the channel and removes the
// subscriber. The unsubscribe function is safe to call multiple times.
// Returns nil, nil if the broker has been shut down.
func (b *Broker) Subscribe() (<-chan ports.DomainEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, nil
	}

	sub := &subscriber{
		ch: make(chan ports.DomainEvent, b.bufSize),
	}
	b.subscribers[sub] = struct{}{}

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			delete(b.subscribers, sub)
			close(sub.ch)
		})
	}

	return sub.ch, unsubscribe
}

// Shutdown closes all subscriber channels and prevents new subscriptions or
// publishes. Safe to call multiple times.
func (b *Broker) Shutdown() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.closed = true
	for sub := range b.subscribers {
		close(sub.ch)
		delete(b.subscribers, sub)
	}
}
