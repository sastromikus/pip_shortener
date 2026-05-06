package audit

import (
	"context"
	"time"
)

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

// Notifier broadcasts audit events to all configured observers.
type Notifier struct {
	observers []Observer
}

// NewNotifier creates a Notifier with the provided observers.
// If no observers are provided, the notifier is effectively disabled.
func NewNotifier(observers ...Observer) *Notifier {
	return &Notifier{observers: observers}
}

func (n *Notifier) Enabled() bool {
	return n != nil && len(n.observers) > 0
}

func (n *Notifier) NotifyAllAsync(ctx context.Context, e Event) {
	if n == nil || len(n.observers) == 0 {
		return
	}

	if e.TS == 0 {
		e.TS = time.Now().Unix()
	}

	for _, obs := range n.observers {
		o := obs
		go func() {
			_ = o.Notify(ctx, e)
		}()
	}
}
