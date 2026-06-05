package audit

import (
	"context"
	"errors"
	"sync"
	"time"
)

const defaultQueueSize = 1024

// Event describes a single audit event emitted by the service.
type Event struct {
	TS     int64  `json:"ts"`
	Action string `json:"action"`
	UserID string `json:"user_id,omitempty"`
	URL    string `json:"url"`
}

// Observer receives audit events from Notifier.
type Observer interface {
	Notify(ctx context.Context, e Event) error
}

// ErrQueueFull is returned when an audit event cannot be queued.
var ErrQueueFull = errors.New("audit queue is full")

// Notifier broadcasts audit events to all configured observers.
type Notifier struct {
	observers []Observer
	queue     chan Event
	wg        sync.WaitGroup
}

// NewNotifier creates a Notifier with the provided observers.
// If no observers are provided, the notifier is effectively disabled.
func NewNotifier(observers ...Observer) *Notifier {
	n := &Notifier{observers: observers}
	if len(observers) == 0 {
		return n
	}

	n.queue = make(chan Event, defaultQueueSize)
	n.wg.Add(1)
	go n.run()

	return n
}

// Enabled reports whether at least one audit observer is configured.
func (n *Notifier) Enabled() bool {
	return n != nil && len(n.observers) > 0
}

func (n *Notifier) notifyAll(ctx context.Context, e Event) error {
	if n == nil || len(n.observers) == 0 {
		return nil
	}

	e = withTimestamp(e)

	var firstErr error
	for _, obs := range n.observers {
		if err := obs.Notify(ctx, e); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// Enqueue schedules the event for background delivery without blocking the HTTP response.
func (n *Notifier) Enqueue(e Event) error {
	if n == nil || len(n.observers) == 0 {
		return nil
	}

	e = withTimestamp(e)
	select {
	case n.queue <- e:
		return nil
	default:
		return ErrQueueFull
	}
}

// Close waits until queued audit events are processed and closes observers that own resources.
func (n *Notifier) Close() error {
	if n == nil || len(n.observers) == 0 {
		return nil
	}

	close(n.queue)
	n.wg.Wait()

	var firstErr error
	for _, obs := range n.observers {
		closer, ok := obs.(interface{ Close() error })
		if !ok {
			continue
		}
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (n *Notifier) run() {
	defer n.wg.Done()
	for e := range n.queue {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = n.notifyAll(ctx, e)
		cancel()
	}
}

func withTimestamp(e Event) Event {
	if e.TS == 0 {
		e.TS = time.Now().Unix()
	}
	return e
}
